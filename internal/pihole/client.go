package pihole

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/eko/pihole-exporter/config"
	"github.com/eko/pihole-exporter/internal/metrics"
)

type ClientStatus byte

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

const (
	MetricsCollectionInProgress ClientStatus = iota
	MetricsCollectionSuccess
	MetricsCollectionError
	MetricsCollectionTimeout
)

func (status ClientStatus) String() string {
	return []string{"MetricsCollectionInProgress", "MetricsCollectionSuccess", "MetricsCollectionError", "MetricsCollectionTimeout"}[status]
}

type ClientChannel struct {
	Status ClientStatus
	Err    error
}

func (c *ClientChannel) String() string {
	if c.Err != nil {
		return fmt.Sprintf("ClientChannel<Status: %s, Err: '%s'>", c.Status, c.Err.Error())
	} else {
		return fmt.Sprintf("ClientChannel<Status: %s, Err: <nil>>", c.Status)
	}
}

// Client struct is a Pi-hole client to request an instance of a Pi-hole ad blocker.
type Client struct {
	apiClient APIClient
	config    *config.Config
	Status    chan *ClientChannel
}

// NewClient method initializes a new Pi-hole client.
func NewClient(config *config.Config, envConfig *config.EnvConfig) *Client {
	err := config.Validate()
	if err != nil {
		log.Fatalf("err: couldn't validate passed Config: %v", err)
	}

	log.Debugf("Creating client with config %+v", config)

	return &Client{
		config:    config,
		apiClient: *NewAPIClient(fmt.Sprintf("%s://%s:%d", config.PIHoleProtocol, config.PIHoleHostname, config.PIHolePort), config.PIHolePassword, envConfig.Timeout, config.SkipTLSVerification),
		Status:    make(chan *ClientChannel, 1),
	}
}

func (c *Client) String() string {
	return c.config.PIHoleHostname
}

func (c *Client) CollectMetricsAsync(writer http.ResponseWriter, request *http.Request) {
	log.Debugf("Collecting from %s", c.config.PIHoleHostname)
	if stats, blockedDomains, permittedDomains, clients, upstreams, piHoleStatus, err := c.getStatistics(); err == nil {
		c.setMetrics(stats, blockedDomains, permittedDomains, clients, upstreams, piHoleStatus)
		c.Status <- &ClientChannel{Status: MetricsCollectionSuccess, Err: nil}
		log.Debugf("New tick of statistics from %s: %s", c.config.PIHoleHostname, stats)
	} else {
		c.Status <- &ClientChannel{Status: MetricsCollectionError, Err: err}
	}
}

func (c *Client) CollectMetrics(writer http.ResponseWriter, request *http.Request) error {
	stats, blockedDomains, permittedDomains, clients, upstreams, piHoleStatus, err := c.getStatistics()
	if err != nil {
		return err
	}
	c.setMetrics(stats, blockedDomains, permittedDomains, clients, upstreams, piHoleStatus)
	log.Debugf("New tick of statistics from %s: %s", c.config.PIHoleHostname, stats)
	return nil
}

func (c *Client) GetHostname() string {
	return c.config.PIHoleHostname
}

func (c *Client) setMetrics(stats *StatsSummary, blockedDomains *TopDomains, permittedDomains *TopDomains, clients *[]PiHoleClient, upstreams *Upstreams, piHoleStatus *BlockingStatus) {
	metrics.DomainsBlocked.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.Gravity.DomainsBeingBlocked))
	metrics.DNSQueriesToday.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.Queries.Total))
	metrics.AdsBlockedToday.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.Queries.Blocked))
	metrics.AdsPercentageToday.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.Queries.PercentBlocked))
	metrics.UniqueDomains.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.Queries.UniqueDomains))
	metrics.QueriesForwarded.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.Queries.Forwarded))
	metrics.QueriesCached.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.Queries.Cached))
	metrics.RequestRate.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.Queries.Frequency))
	metrics.ClientsEverSeen.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.Clients.Total))
	metrics.UniqueClients.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.Clients.Active))
	metrics.DNSQueriesAllTypes.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.Queries.Total))
	if piHoleStatus.Blocking == "enabled" {
		metrics.Status.WithLabelValues(c.config.PIHoleHostname).Set(1)
	} else {
		metrics.Status.WithLabelValues(c.config.PIHoleHostname).Set(0)
	}

	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "unknown").Set(float64(stats.Queries.Replies.UNKNOWN))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "no_data").Set(float64(stats.Queries.Replies.NODATA))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "nx_domain").Set(float64(stats.Queries.Replies.NXDOMAIN))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "cname").Set(float64(stats.Queries.Replies.CNAME))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "ip").Set(float64(stats.Queries.Replies.IP))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "domain").Set(float64(stats.Queries.Replies.DOMAIN))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "rr_name").Set(float64(stats.Queries.Replies.RRNAME))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "serv_fail").Set(float64(stats.Queries.Replies.SERVFAIL))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "refused").Set(float64(stats.Queries.Replies.REFUSED))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "not_imp").Set(float64(stats.Queries.Replies.NOTIMP))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "other").Set(float64(stats.Queries.Replies.OTHER))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "dnssec").Set(float64(stats.Queries.Replies.DNSSEC))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "none").Set(float64(stats.Queries.Replies.NONE))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "blob").Set(float64(stats.Queries.Replies.BLOB))

	for _, domain := range permittedDomains.Domains {
		metrics.TopQueries.WithLabelValues(c.config.PIHoleHostname, domain.Domain).Set(float64(domain.Count))
	}

	for _, domain := range blockedDomains.Domains {
		metrics.TopAds.WithLabelValues(c.config.PIHoleHostname, domain.Domain).Set(float64(domain.Count))
	}

	for _, client := range *clients {
		metrics.TopSources.WithLabelValues(c.config.PIHoleHostname, client.IP, client.Name).Set(float64(client.Count))
	}

	for _, upstream := range upstreams.Upstreams {
		metrics.ForwardDestinations.WithLabelValues(c.config.PIHoleHostname, upstream.IP, upstream.Name).Set(float64(upstream.Count))
		metrics.ForwardDestinationsResponseTime.WithLabelValues(c.config.PIHoleHostname, upstream.IP, upstream.Name).Set(upstream.Statistics.Response)
		metrics.ForwardDestinationsResponseVariance.WithLabelValues(c.config.PIHoleHostname, upstream.IP, upstream.Name).Set(upstream.Statistics.Variance)
	}

	for queryType, value := range stats.Queries.Types {
		metrics.QueryTypes.WithLabelValues(c.config.PIHoleHostname, queryType).Set(value)
	}
}

func (c *Client) getStatistics() (*StatsSummary, *TopDomains, *TopDomains, *[]PiHoleClient, *Upstreams, *BlockingStatus, error) {
	var statsSummary StatsSummary
	var permittedDomains TopDomains
	var blockedDomains TopDomains
	var permittedClients TopClients
	var blockedClients TopClients
	var upstreams Upstreams
	var piHoleStatus BlockingStatus

	err := c.apiClient.FetchData("/api/stats/summary", &statsSummary)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching stats summary: %w", err)
	}

	err = c.apiClient.FetchData("/api/stats/top_domains?blocked=true&count=10", &blockedDomains)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching blocked domains: %w", err)
	}
	err = c.apiClient.FetchData("/api/stats/top_domains?blocked=false&count=10", &permittedDomains)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching permitted domains: %w", err)
	}

	err = c.apiClient.FetchData("/api/stats/top_clients?blocked=true&count=10", &blockedClients)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching blocked clients: %w", err)
	}
	err = c.apiClient.FetchData("/api/stats/top_clients?blocked=false&count=10", &permittedClients)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching permitted clients: %w", err)
	}

	clients := MergeClients(permittedClients.Clients, blockedClients.Clients)

	err = c.apiClient.FetchData("/api/stats/upstreams", &upstreams)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching upstream stats: %w", err)
	}

	err = c.apiClient.FetchData("/api/dns/blocking", &piHoleStatus)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching status: %w", err)
	}

	return &statsSummary, &blockedDomains, &permittedDomains, &clients, &upstreams, &piHoleStatus, nil
}

// Close cleans up resources used by the client
func (c *Client) Close() {
	// Drain the status channel if needed
	select {
	case <-c.Status:
		// Channel had something, now it's drained
	default:
		// Channel was already empty
	}

	log.Debugf("Closing client %s", c.config.PIHoleHostname)
	c.apiClient.Close() // Close the API client
}
