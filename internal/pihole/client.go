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

	"github.com/eko/pihole-exporter/internal/metrics"
)

var (
	loginURLPattern = "http://%s/admin/index.php?login"
	statsURLPattern = "http://%s/admin/api.php?summaryRaw&overTimeData&topItems&recentItems&getQueryTypes&getForwardDestinations&getQuerySources&jsonForceObject"
)

// Client struct is a PI-Hole client to request an instance of a PI-Hole ad blocker.
type Client struct {
	hostname   string
	password   string
	interval   time.Duration
	httpClient http.Client
}

// NewClient method initializes a new PI-Hole client.
func NewClient(hostname, password string, interval time.Duration) *Client {
	return &Client{
		hostname: hostname,
		password: password,
		interval: interval,
		httpClient: http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Scrape method logins and retrieves statistics from PI-Hole JSON API
// and then pass them as Prometheus metrics.
func (c *Client) Scrape() {
	for range time.Tick(c.interval) {
		sessionID := c.getPHPSessionID()
		if sessionID == nil {
			log.Println("Unable to retrieve session identifier")
			return
		}

		stats := c.getStatistics(*sessionID)
		c.setMetrics(stats)

		log.Printf("New tick of statistics: %s", stats.ToString())
	}
}

func (c *Client) setMetrics(stats *Stats) {
	metrics.DomainsBlocked.WithLabelValues(c.hostname).Set(float64(stats.DomainsBeingBlocked))
	metrics.DNSQueriesToday.WithLabelValues(c.hostname).Set(float64(stats.DNSQueriesToday))
	metrics.AdsBlockedToday.WithLabelValues(c.hostname).Set(float64(stats.AdsBlockedToday))
	metrics.AdsPercentageToday.WithLabelValues(c.hostname).Set(float64(stats.AdsPercentageToday))
	metrics.UniqueDomains.WithLabelValues(c.hostname).Set(float64(stats.UniqueDomains))
	metrics.QueriesForwarded.WithLabelValues(c.hostname).Set(float64(stats.QueriesForwarded))
	metrics.QueriesCached.WithLabelValues(c.hostname).Set(float64(stats.QueriesCached))
	metrics.ClientsEverSeen.WithLabelValues(c.hostname).Set(float64(stats.ClientsEverSeen))
	metrics.UniqueClients.WithLabelValues(c.hostname).Set(float64(stats.UniqueClients))
	metrics.DNSQueriesAllTypes.WithLabelValues(c.hostname).Set(float64(stats.DNSQueriesAllTypes))

	metrics.Reply.WithLabelValues(c.hostname, "no_data").Set(float64(stats.ReplyNoData))
	metrics.Reply.WithLabelValues(c.hostname, "nx_domain").Set(float64(stats.ReplyNxDomain))
	metrics.Reply.WithLabelValues(c.hostname, "cname").Set(float64(stats.ReplyCname))
	metrics.Reply.WithLabelValues(c.hostname, "ip").Set(float64(stats.ReplyIP))

	var isEnabled int = 0
	if stats.Status == enabledStatus {
		isEnabled = 1
	}
	metrics.Status.WithLabelValues(c.hostname).Set(float64(isEnabled))

	for domain, value := range stats.TopQueries {
		metrics.TopQueries.WithLabelValues(c.hostname, domain).Set(float64(value))
	}

	for domain, value := range stats.TopAds {
		metrics.TopAds.WithLabelValues(c.hostname, domain).Set(float64(value))
	}

	for source, value := range stats.TopSources {
		metrics.TopSources.WithLabelValues(c.hostname, source).Set(float64(value))
	}

	for destination, value := range stats.ForwardDestinations {
		metrics.ForwardDestinations.WithLabelValues(c.hostname, destination).Set(value)
	}

	for queryType, value := range stats.QueryTypes {
		metrics.QueryTypes.WithLabelValues(c.hostname, queryType).Set(value)
	}
}

func (c *Client) getPHPSessionID() *string {
	var sessionID string

	loginURL := fmt.Sprintf(loginURLPattern, c.hostname)
	values := url.Values{"pw": []string{c.password}}

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(values.Encode()))
	if err != nil {
		log.Fatal("An error has occured when creating HTTP statistics request", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(values.Encode())))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("An error has occured during login to PI-Hole: %v", err)
	}

	if resp.StatusCode != http.StatusFound {
		log.Printf("Unable to login to PI-Hole, got a HTTP status code response '%d' instead of '%d'", resp.StatusCode, http.StatusFound)
		os.Exit(1)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "PHPSESSID" {
			sessionID = cookie.Value
			break
		}
	}

	return &sessionID
}

func (c *Client) getStatistics(sessionID string) *Stats {
	var stats Stats

	statsURL := fmt.Sprintf(statsURLPattern, c.hostname)

	req, err := http.NewRequest("GET", statsURL, nil)
	if err != nil {
		log.Fatal("An error has occured when creating HTTP statistics request", err)
	}

	cookie := http.Cookie{Name: "PHPSESSID", Value: sessionID}
	req.AddCookie(&cookie)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Println("An error has occured during retrieving PI-Hole statistics", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Unable to read PI-Hole statistics HTTP response", err)
	}

	err = json.Unmarshal(body, &stats)
	if err != nil {
		log.Println("Unable to unmarshal PI-Hole statistics to statistics struct model", err)
	}

	return &stats
}
