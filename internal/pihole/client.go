package pihole

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/eko/pihole-exporter/config"
	"github.com/eko/pihole-exporter/internal/metrics"
)

type AuthenticationResponse struct {
	Session struct {
		Valid    bool   `json:"valid"`
		Totp     bool   `json:"totp"`
		Sid      string `json:"sid"`
		Csrf     string `json:"csrf"`
		Validity int    `json:"validity"`
		Message  string `json:"message"`
	} `json:"session"`
}

// Client struct is a Pi-hole client to request an instance of a Pi-hole ad blocker.
type Client struct {
	apiClient APIClient
	config    *config.Config
}

// NewClient method initializes a new Pi-hole client.
func NewClient(config *config.Config, envConfig *config.EnvConfig) *Client {
	err := config.Validate()
	if err != nil {
		log.Fatalf("err: couldn't validate passed Config: %v", err)
	}

	log.Debugf("Creating client for host %s with protocol %s and port %d", config.PIHoleHostname, config.PIHoleProtocol, config.PIHolePort)

	return &Client{
		config:    config,
		apiClient: *NewAPIClient(fmt.Sprintf("%s://%s:%d", config.PIHoleProtocol, config.PIHoleHostname, config.PIHolePort), config.PIHolePassword, envConfig.Timeout, envConfig.SkipTLSVerification),
	}
}

func (c *Client) String() string {
	return c.config.PIHoleHostname
}

// statistics holds everything fetched from the Pi-hole API during one collection.
type statistics struct {
	summary          StatsSummary
	blockedDomains   TopDomains
	permittedDomains TopDomains
	clients          []PiHoleClient
	blockedClients   []PiHoleClient
	upstreams        Upstreams
	blocking         BlockingStatus
}

// CollectMetrics fetches the Pi-hole statistics and updates the exported metrics.
func (c *Client) CollectMetrics() error {
	stats, err := c.getStatistics()
	if err != nil {
		return err
	}
	c.setMetrics(stats)
	log.Debugf("New tick of statistics from %s: %s", c.config.PIHoleHostname, &stats.summary)
	return nil
}

func (c *Client) GetHostname() string {
	return c.config.PIHoleHostname
}

func (c *Client) setMetrics(stats *statistics) {
	hostname := c.config.PIHoleHostname
	summary := stats.summary

	metrics.DomainsBlocked.WithLabelValues(hostname).Set(float64(summary.Gravity.DomainsBeingBlocked))
	metrics.DNSQueriesToday.WithLabelValues(hostname).Set(float64(summary.Queries.Total))
	metrics.AdsBlockedToday.WithLabelValues(hostname).Set(float64(summary.Queries.Blocked))
	metrics.AdsPercentageToday.WithLabelValues(hostname).Set(float64(summary.Queries.PercentBlocked))
	metrics.UniqueDomains.WithLabelValues(hostname).Set(float64(summary.Queries.UniqueDomains))
	metrics.QueriesForwarded.WithLabelValues(hostname).Set(float64(summary.Queries.Forwarded))
	metrics.QueriesCached.WithLabelValues(hostname).Set(float64(summary.Queries.Cached))
	metrics.RequestRate.WithLabelValues(hostname).Set(float64(summary.Queries.Frequency))
	metrics.ClientsEverSeen.WithLabelValues(hostname).Set(float64(summary.Clients.Total))
	metrics.UniqueClients.WithLabelValues(hostname).Set(float64(summary.Clients.Active))
	metrics.DNSQueriesAllTypes.WithLabelValues(hostname).Set(float64(summary.Queries.Total))
	if stats.blocking.Blocking == "enabled" {
		metrics.Status.WithLabelValues(hostname).Set(1)
	} else {
		metrics.Status.WithLabelValues(hostname).Set(0)
	}

	metrics.Reply.WithLabelValues(hostname, "unknown").Set(float64(summary.Queries.Replies.UNKNOWN))
	metrics.Reply.WithLabelValues(hostname, "no_data").Set(float64(summary.Queries.Replies.NODATA))
	metrics.Reply.WithLabelValues(hostname, "nx_domain").Set(float64(summary.Queries.Replies.NXDOMAIN))
	metrics.Reply.WithLabelValues(hostname, "cname").Set(float64(summary.Queries.Replies.CNAME))
	metrics.Reply.WithLabelValues(hostname, "ip").Set(float64(summary.Queries.Replies.IP))
	metrics.Reply.WithLabelValues(hostname, "domain").Set(float64(summary.Queries.Replies.DOMAIN))
	metrics.Reply.WithLabelValues(hostname, "rr_name").Set(float64(summary.Queries.Replies.RRNAME))
	metrics.Reply.WithLabelValues(hostname, "serv_fail").Set(float64(summary.Queries.Replies.SERVFAIL))
	metrics.Reply.WithLabelValues(hostname, "refused").Set(float64(summary.Queries.Replies.REFUSED))
	metrics.Reply.WithLabelValues(hostname, "not_imp").Set(float64(summary.Queries.Replies.NOTIMP))
	metrics.Reply.WithLabelValues(hostname, "other").Set(float64(summary.Queries.Replies.OTHER))
	metrics.Reply.WithLabelValues(hostname, "dnssec").Set(float64(summary.Queries.Replies.DNSSEC))
	metrics.Reply.WithLabelValues(hostname, "none").Set(float64(summary.Queries.Replies.NONE))
	metrics.Reply.WithLabelValues(hostname, "blob").Set(float64(summary.Queries.Replies.BLOB))

	// Domains, clients and upstreams drop out of the top lists over time: remove
	// this host's previous series so they don't linger with stale values.
	hostLabels := prometheus.Labels{"hostname": hostname}
	for _, vec := range []*prometheus.GaugeVec{
		metrics.TopQueries,
		metrics.TopAds,
		metrics.TopSources,
		metrics.TopSourcesBlocked,
		metrics.ForwardDestinations,
		metrics.ForwardDestinationsResponseTime,
		metrics.ForwardDestinationsResponseVariance,
		metrics.QueryTypes,
	} {
		vec.DeletePartialMatch(hostLabels)
	}

	for _, domain := range stats.permittedDomains.Domains {
		metrics.TopQueries.WithLabelValues(hostname, domain.Domain).Set(float64(domain.Count))
	}

	for _, domain := range stats.blockedDomains.Domains {
		metrics.TopAds.WithLabelValues(hostname, domain.Domain).Set(float64(domain.Count))
	}

	for _, client := range stats.clients {
		metrics.TopSources.WithLabelValues(hostname, client.IP, client.Name).Set(float64(client.Count))
	}

	for _, client := range stats.blockedClients {
		metrics.TopSourcesBlocked.WithLabelValues(hostname, client.IP, client.Name).Set(float64(client.Count))
	}

	for _, upstream := range stats.upstreams.Upstreams {
		metrics.ForwardDestinations.WithLabelValues(hostname, upstream.IP, upstream.Name).Set(float64(upstream.Count))
		metrics.ForwardDestinationsResponseTime.WithLabelValues(hostname, upstream.IP, upstream.Name).Set(upstream.Statistics.Response)
		metrics.ForwardDestinationsResponseVariance.WithLabelValues(hostname, upstream.IP, upstream.Name).Set(upstream.Statistics.Variance)
	}

	for queryType, value := range summary.Queries.Types {
		metrics.QueryTypes.WithLabelValues(hostname, queryType).Set(value)
	}
}

func (c *Client) getStatistics() (*statistics, error) {
	var stats statistics
	var permittedClients TopClients
	var blockedClients TopClients

	fetches := []struct {
		endpoint string
		result   any
		name     string
	}{
		{"/api/stats/summary", &stats.summary, "stats summary"},
		{"/api/stats/top_domains?blocked=true&count=10", &stats.blockedDomains, "blocked domains"},
		{"/api/stats/top_domains?blocked=false&count=10", &stats.permittedDomains, "permitted domains"},
		{"/api/stats/top_clients?blocked=true&count=10", &blockedClients, "blocked clients"},
		{"/api/stats/top_clients?blocked=false&count=10", &permittedClients, "permitted clients"},
		{"/api/stats/upstreams", &stats.upstreams, "upstream stats"},
		{"/api/dns/blocking", &stats.blocking, "status"},
	}
	for _, fetch := range fetches {
		if err := c.apiClient.FetchData(fetch.endpoint, fetch.result); err != nil {
			return nil, fmt.Errorf("error fetching %s: %w", fetch.name, err)
		}
	}

	stats.clients = MergeClients(permittedClients.Clients, blockedClients.Clients)
	stats.blockedClients = blockedClients.Clients
	return &stats, nil
}

// Close cleans up resources used by the client
func (c *Client) Close() {
	log.Debugf("Closing client %s", c.config.PIHoleHostname)
	c.apiClient.Close() // Close the API client
}
