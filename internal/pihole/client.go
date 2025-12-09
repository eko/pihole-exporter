package pihole

import (
	"fmt"
	"net/http"
	"time"

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

	log.Debugf("Creating client for host %s with protocol %s and port %d", config.PIHoleHostname, config.PIHoleProtocol, config.PIHolePort)

	return &Client{
		config:    config,
		apiClient: *NewAPIClient(fmt.Sprintf("%s://%s:%d", config.PIHoleProtocol, config.PIHoleHostname, config.PIHolePort), config.PIHolePassword, envConfig.Timeout, envConfig.SkipTLSVerification),
		Status:    make(chan *ClientChannel, 1),
	}
}

func (c *Client) String() string {
	return c.config.PIHoleHostname
}

func (c *Client) CollectMetricsAsync(writer http.ResponseWriter, request *http.Request) {
	log.Debugf("Collecting from %s", c.config.PIHoleHostname)
	if queryHistoryResponse, stats, blockedDomains, permittedDomains, clients, upstreams, piHoleStatus, err := c.getStatistics(); err == nil {
		c.setMetrics(queryHistoryResponse, stats, blockedDomains, permittedDomains, clients, upstreams, piHoleStatus)
		c.Status <- &ClientChannel{Status: MetricsCollectionSuccess, Err: nil}
		log.Debugf("New tick of statistics from %s: %s", c.config.PIHoleHostname, stats)
	} else {
		c.Status <- &ClientChannel{Status: MetricsCollectionError, Err: err}
	}
}

func (c *Client) CollectMetrics(writer http.ResponseWriter, request *http.Request) error {
	queryHistoryResponse, stats, blockedDomains, permittedDomains, clients, upstreams, piHoleStatus, err := c.getStatistics()
	if err != nil {
		return err
	}
	c.setMetrics(queryHistoryResponse, stats, blockedDomains, permittedDomains, clients, upstreams, piHoleStatus)
	log.Debugf("New tick of statistics from %s: %s", c.config.PIHoleHostname, stats)
	return nil
}

func (c *Client) GetHostname() string {
	return c.config.PIHoleHostname
}

func (c *Client) setMetrics(queryHistoryResponse *QueryHistoryResponse, stats *StatsSummary, blockedDomains *TopDomains, permittedDomains *TopDomains, clients *[]PiHoleClient, upstreams *Upstreams, piHoleStatus *BlockingStatus) {

	// go thru each entry in queryHistoryResponse.History and add the metrics.
	lastQuery := getLastQueryEntry(c.config.PIHoleHostname)
	total := lastQuery.Total
	cached := lastQuery.Cached
	blocked := lastQuery.Blocked
	forwarded := lastQuery.Forwarded

	// if response history is empty, we skip processing
	if len(queryHistoryResponse.History) > 0 {
		for _, entry := range queryHistoryResponse.History {
			total += float64(entry.Total)
			cached += float64(entry.Cached)
			blocked += float64(entry.Blocked)
			forwarded += float64(entry.Forwarded)
			log.Infof("Values - Total: %.0f, Cached: %.0f, Blocked: %.0f, Forwarded: %.0f", entry.Total, entry.Cached, entry.Blocked, entry.Forwarded)
		}

		// Update last query metrics
		lastQuery.Timestamp = time.Now().Unix() - 60
		lastQuery.Total = total
		lastQuery.Cached = cached
		lastQuery.Blocked = blocked
		lastQuery.Forwarded = forwarded
		setLastQueryEntry(lastQuery, c.config.PIHoleHostname)
	}

	log.Infof("Window Queries - Total: %.0f, Cached: %.0f, Blocked: %.0f, Forwarded: %.0f", total, cached, blocked, forwarded)

	// metrics.Queries.WithLabelValues(c.config.PIHoleHostname, "total").Set(float64(entry.Total))
	metrics.Queries.WithLabelValues(c.config.PIHoleHostname, "blocked").Set(float64(blocked))
	metrics.Queries.WithLabelValues(c.config.PIHoleHostname, "cached").Set(float64(cached))
	metrics.Queries.WithLabelValues(c.config.PIHoleHostname, "forwarded").Set(float64(total - cached - blocked))

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

func (c *Client) getStatistics() (*QueryHistoryResponse, *StatsSummary, *TopDomains, *TopDomains, *[]PiHoleClient, *Upstreams, *BlockingStatus, error) {
	var queryHistoryResponse QueryHistoryResponse
	var statsSummary StatsSummary
	var permittedDomains TopDomains
	var blockedDomains TopDomains
	var permittedClients TopClients
	var blockedClients TopClients
	var upstreams Upstreams
	var piHoleStatus BlockingStatus

	// Read LAST_QUERY_DATA env variable using helper
	// run one minute back in time
	now := time.Now().Unix() - 60
	lastQuery := getLastQueryEntry(c.config.PIHoleHostname)
	log.Infof("URL = /api/history/database?from=%d&until=%d", lastQuery.Timestamp, now)
	err := c.apiClient.FetchData(fmt.Sprintf("/api/history/database?from=%d&until=%d", lastQuery.Timestamp, now), &queryHistoryResponse)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching query history: %w", err)
	}

	err = c.apiClient.FetchData("/api/stats/summary", &statsSummary)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching stats summary: %w", err)
	}

	err = c.apiClient.FetchData("/api/stats/top_domains?blocked=true&count=10", &blockedDomains)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching blocked domains: %w", err)
	}
	err = c.apiClient.FetchData("/api/stats/top_domains?blocked=false&count=10", &permittedDomains)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching permitted domains: %w", err)
	}

	err = c.apiClient.FetchData("/api/stats/top_clients?blocked=true&count=10", &blockedClients)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching blocked clients: %w", err)
	}
	err = c.apiClient.FetchData("/api/stats/top_clients?blocked=false&count=10", &permittedClients)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching permitted clients: %w", err)
	}

	clients := MergeClients(permittedClients.Clients, blockedClients.Clients)

	err = c.apiClient.FetchData("/api/stats/upstreams", &upstreams)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching upstream stats: %w", err)
	}

	err = c.apiClient.FetchData("/api/dns/blocking", &piHoleStatus)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error fetching status: %w", err)
	}

	return &queryHistoryResponse, &statsSummary, &blockedDomains, &permittedDomains, &clients, &upstreams, &piHoleStatus, nil
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
