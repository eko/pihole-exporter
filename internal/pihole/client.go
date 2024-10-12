package pihole

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/eko/pihole-exporter/config"
	"github.com/eko/pihole-exporter/internal/metrics"
)

type ClientStatus byte

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
	httpClient http.Client
	interval   time.Duration
	config     *config.Config
	Status     chan *ClientChannel
}

// NewClient method initializes a new Pi-hole client.
func NewClient(config *config.Config, envConfig *config.EnvConfig) *Client {
	err := config.Validate()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	log.Printf("Creating client with config %s\n", config)

	return &Client{
		config: config,
		httpClient: http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Timeout: envConfig.Timeout,
		},
		Status: make(chan *ClientChannel, 1),
	}
}

func (c *Client) String() string {
	return c.config.PIHoleHostname
}

func (c *Client) CollectMetricsAsync(writer http.ResponseWriter, request *http.Request) {
	log.Printf("Collecting from %s", c.config.PIHoleHostname)
	if stats, err := c.getStatistics(); err == nil {
		c.setMetrics(stats)
		c.Status <- &ClientChannel{Status: MetricsCollectionSuccess, Err: nil}
		log.Printf("New tick of statistics from %s: %s", c.config.PIHoleHostname, stats)
	} else {
		c.Status <- &ClientChannel{Status: MetricsCollectionError, Err: err}
	}
}

func (c *Client) CollectMetrics(writer http.ResponseWriter, request *http.Request) error {
	stats, err := c.getStatistics()
	if err != nil {
		return err
	}
	c.setMetrics(stats)
	log.Printf("New tick of statistics from %s: %s", c.config.PIHoleHostname, stats)
	return nil
}

func (c *Client) GetHostname() string {
	return c.config.PIHoleHostname
}

// Pi-hole returns a map of unix epoch time with the number of stats in slots of 10 minutes.
// The last epoch is the current in-progress time slot, with stats still being added.
// We return the second latest epoch stats, which is definitive.
func latestEpochStats(statsOverTime map[int]int) float64 {
    var lastEpoch, secondLastEpoch int
    for timestamp := range statsOverTime {
        if timestamp > lastEpoch {
            secondLastEpoch = lastEpoch
            lastEpoch = timestamp
        } else if timestamp > secondLastEpoch && timestamp != lastEpoch {
            secondLastEpoch = timestamp
        }
    }
    return float64(statsOverTime[secondLastEpoch])
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

	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "unknown").Set(float64(stats.ReplyUnknown))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "no_data").Set(float64(stats.ReplyNoData))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "nx_domain").Set(float64(stats.ReplyNxDomain))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "cname").Set(float64(stats.ReplyCname))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "ip").Set(float64(stats.ReplyIP))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "domain").Set(float64(stats.ReplyDomain))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "rr_name").Set(float64(stats.ReplyRRName))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "serv_fail").Set(float64(stats.ReplyServFail))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "refused").Set(float64(stats.ReplyRefused))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "not_imp").Set(float64(stats.ReplyNotImp))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "other").Set(float64(stats.ReplyOther))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "dnssec").Set(float64(stats.ReplyDNSSEC))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "none").Set(float64(stats.ReplyNone))
	metrics.Reply.WithLabelValues(c.config.PIHoleHostname, "blob").Set(float64(stats.ReplyBlob))

	var isEnabled int = 0
	if stats.Status == enabledStatus {
		isEnabled = 1
	}
	metrics.Status.WithLabelValues(c.config.PIHoleHostname).Set(float64(isEnabled))

	// Pi-hole returns a subset of stats when Auth is missing or incorrect.
	// This provides a warning to users that metrics are not complete.
	if len(stats.TopQueries) == 0 {
		log.Warnf("Invalid Authentication - Some metrics may be missing. Please confirm your Pi-hole API token / Password for %s", c.config.PIHoleHostname)
	}

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

	metrics.QueriesLast10min.WithLabelValues(c.config.PIHoleHostname).Set(latestEpochStats(stats.DomainsOverTime))
	metrics.AdsLast10min.WithLabelValues(c.config.PIHoleHostname).Set(latestEpochStats(stats.AdsOverTime))
}

func (c *Client) getPHPSessionID() (string, error) {
	values := url.Values{"pw": []string{c.config.PIHolePassword}}

	req, err := http.NewRequest("POST", c.config.PIHoleLoginURL(), strings.NewReader(values.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating HTTP statistics request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(values.Encode())))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("loging in to Pi-hole: %w", err)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "PHPSESSID" {
			return cookie.Value, nil
		}
	}

	return "", fmt.Errorf("no PHPSESSID cookie found")
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
		err := c.authenticateRequest(req)
		if err != nil {
			return nil, fmt.Errorf("an error has occurred authenticating the request: %w", err)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("an error has occured during retrieving Pi-hole statistics: %w", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read Pi-hole statistics HTTP response: %w", err)
	}

	err = json.Unmarshal(body, stats)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal Pi-hole statistics to statistics struct model: %w", err)
	}

	return stats, nil
}

func (c *Client) isUsingPassword() bool {
	return len(c.config.PIHolePassword) > 0
}

func (c *Client) isUsingApiToken() bool {
	return len(c.config.PIHoleApiToken) > 0
}

func (c *Client) authenticateRequest(req *http.Request) error {
	sessionID, err := c.getPHPSessionID()
	if err != nil {
		return err
	}
	cookie := http.Cookie{Name: "PHPSESSID", Value: sessionID}
	req.AddCookie(&cookie)
	return nil
}
