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

type Client struct {
	hostname   string
	password   string
	interval   time.Duration
	httpClient http.Client
}

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

func (c *Client) Fetch() {
	for range time.Tick(c.interval) {
		sessionID := c.getPHPSessionID()
		if sessionID == nil {
			log.Println("Unable to retrieve session identifier")
			return
		}

		stats := c.getStatistics(*sessionID)

		log.Println("New tick of statistics", stats)

		metrics.DomainsBlocked.Set(float64(stats.DomainsBeingBlocked))
		metrics.DNSQueriesToday.Set(float64(stats.DNSQueriesToday))
		metrics.AdsBlockedToday.Set(float64(stats.AdsBlockedToday))
		metrics.AdsPercentageToday.Set(float64(stats.AdsPercentageToday))
		metrics.UniqueDomains.Set(float64(stats.UniqueDomains))
		metrics.QueriesForwarded.Set(float64(stats.QueriesForwarded))
		metrics.QueriesCached.Set(float64(stats.QueriesCached))
		metrics.ClientsEverSeen.Set(float64(stats.ClientsEverSeen))
		metrics.UniqueClients.Set(float64(stats.UniqueClients))
		metrics.DnsQueriesAllTypes.Set(float64(stats.DnsQueriesAllTypes))
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
