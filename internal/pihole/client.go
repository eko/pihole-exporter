package pihole

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/eko/pihole-exporter/config"
	"github.com/eko/pihole-exporter/internal/metrics"
)

// Client struct is a PI-Hole client to request an instance of a PI-Hole ad blocker.
type Client struct {
	httpClient      http.Client
	interval        time.Duration
	config          *config.Config
	MetricRetrieved chan bool
}

// NewClient method initializes a new PI-Hole client.
func NewClient(config *config.Config) *Client {
	err := config.Validate()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	fmt.Printf("Creating client with config %s\n", config)

	return &Client{
		config: config,
		httpClient: http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *Client) String() string {
	return c.config.PIHoleHostname
}

/*
// Metrics scrapes pihole and sets them
func (c *Client) Metrics() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		stats, err := c.getStatistics()
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte(err.Error()))
			return
		}
		c.setMetrics(stats)

		log.Printf("New tick of statistics: %s", stats.ToString())
		promhttp.Handler().ServeHTTP(writer, request)
	}
}*/

func (c *Client) CollectMetrics(writer http.ResponseWriter, request *http.Request) error {

	stats, err := c.getStatistics()
	if err != nil {
		return err
	}
	c.setMetrics(stats)

	log.Printf("New tick of statistics from %s: %s", c, stats)
	return nil
}

func (c *Client) GetHostname() string {
	return c.config.PIHoleHostname
}

func (c *Client) setMetrics(stats *Stats) {
	metrics.DomainsBlocked.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.DomainsBeingBlocked))
	metrics.DNSQueriesToday.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.DNSQueriesToday))
	metrics.AdsBlockedToday.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.AdsBlockedToday))
	metrics.AdsPercentageToday.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.AdsPercentageToday))
	metrics.UniqueDomains.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.UniqueDomains))
	metrics.QueriesForwarded.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.QueriesForwarded))
	metrics.QueriesCached.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.QueriesCached))
	metrics.ClientsEverSeen.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.ClientsEverSeen))
	metrics.UniqueClients.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.UniqueClients))
	metrics.DNSQueriesAllTypes.WithLabelValues(c.config.PIHoleHostname).Set(float64(stats.DNSQueriesAllTypes))

	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "no_data").Set(float64(stats.ReplyNoData))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "nx_domain").Set(float64(stats.ReplyNxDomain))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "cname").Set(float64(stats.ReplyCname))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "ip").Set(float64(stats.ReplyIP))

	var isEnabled int = 0
	if stats.Status == enabledStatus {
		isEnabled = 1
	}
	metrics.Status.WithLabelValues(c.config.PIHoleHostname).Set(float64(isEnabled))

	for domain, value := range stats.TopQueries {
		metrics.TopQueries.WithLabelValues(c.config.PIHoleHostname, domain).Set(float64(value))
	}

	for domain, value := range stats.TopAds {
		metrics.TopAds.WithLabelValues(c.config.PIHoleHostname, domain).Set(float64(value))
	}

	for source, value := range stats.TopSources {
		metrics.TopSources.WithLabelValues(c.config.PIHoleHostname, source).Set(float64(value))
	}

	for destination, value := range stats.ForwardDestinations {
		metrics.ForwardDestinations.WithLabelValues(c.config.PIHoleHostname, destination).Set(value)
	}

	for queryType, value := range stats.QueryTypes {
		metrics.QueryTypes.WithLabelValues(c.config.PIHoleHostname, queryType).Set(value)
	}
}

func (c *Client) getPHPSessionID() (sessionID string) {
	values := url.Values{"pw": []string{c.config.PIHolePassword}}

	req, err := http.NewRequest("POST", c.config.PIHoleLoginURL(), strings.NewReader(values.Encode()))
	if err != nil {
		log.Fatal("An error has occured when creating HTTP statistics request", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(values.Encode())))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("An error has occured during login to PI-Hole: %v", err)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "PHPSESSID" {
			sessionID = cookie.Value
			break
		}
	}

	return
}

func (c *Client) getStatistics() (*Stats, error) {
	stats := new(Stats)

	statsURL := c.config.PIHoleStatsURL()

	if c.isUsingApiToken() {
		statsURL = fmt.Sprintf("%s&auth=%s", statsURL, c.config.PIHoleApiToken)
	}

	req, err := http.NewRequest("GET", statsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("an error has occured when creating HTTP statistics request: %w", err)
	}

	if c.isUsingPassword() {
		c.authenticateRequest(req)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("an error has occured during retrieving PI-Hole statistics: %w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read PI-Hole statistics HTTP response: %w", err)
	}

	err = json.Unmarshal(body, stats)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal PI-Hole statistics to statistics struct model: %w", err)
	}

	return stats, nil
}

func (c *Client) isUsingPassword() bool {
	return len(c.config.PIHolePassword) > 0
}

func (c *Client) isUsingApiToken() bool {
	return len(c.config.PIHoleApiToken) > 0
}

func (c *Client) authenticateRequest(req *http.Request) {
	cookie := http.Cookie{Name: "PHPSESSID", Value: c.getPHPSessionID()}
	req.AddCookie(&cookie)
}
