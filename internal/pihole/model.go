package pihole

import "fmt"

type Upstreams struct {
	Upstreams []struct {
		IP         string `json:"ip"`
		Name       string `json:"name"`
		Port       int    `json:"port"`
		Count      int    `json:"count"`
		Statistics struct {
			Response float64 `json:"response"`
			Variance float64 `json:"variance"`
		} `json:"statistics"`
	} `json:"upstreams"`
	ForwardedQueries int     `json:"forwarded_queries"`
	TotalQueries     int     `json:"total_queries"`
	Took             float64 `json:"took"`
}

type TopDomains struct {
	Domains []struct {
		Domain string `json:"domain"`
		Count  int    `json:"count"`
	} `json:"domains"`
	TotalQueries   int     `json:"total_queries"`
	BlockedQueries int     `json:"blocked_queries"`
	Took           float64 `json:"took"`
}

type PiHoleClient struct {
	IP    string `json:"ip"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Response struct represents the full JSON response
type TopClients struct {
	Clients        []PiHoleClient `json:"clients"`
	TotalQueries   int            `json:"total_queries"`
	BlockedQueries int            `json:"blocked_queries"`
	Took           float64        `json:"took"`
}

type StatsSummary struct {
	Queries struct {
		Total          int                `json:"total"`
		Blocked        int                `json:"blocked"`
		PercentBlocked float64            `json:"percent_blocked"`
		UniqueDomains  int                `json:"unique_domains"`
		Forwarded      int                `json:"forwarded"`
		Cached         int                `json:"cached"`
		Frequency      float64            `json:"frequency"`
		Types          map[string]float64 `json:"types"`
		Status         struct {
			UNKNOWN              int `json:"UNKNOWN"`
			GRAVITY              int `json:"GRAVITY"`
			FORWARDED            int `json:"FORWARDED"`
			CACHE                int `json:"CACHE"`
			REGEX                int `json:"REGEX"`
			DENYLIST             int `json:"DENYLIST"`
			EXTERNALBLOCKEDIP    int `json:"EXTERNAL_BLOCKED_IP"`
			EXTERNALBLOCKEDNULL  int `json:"EXTERNAL_BLOCKED_NULL"`
			EXTERNALBLOCKEDNXRA  int `json:"EXTERNAL_BLOCKED_NXRA"`
			GRAVITYCNAME         int `json:"GRAVITY_CNAME"`
			REGEXCNAME           int `json:"REGEX_CNAME"`
			DENYLISTCNAME        int `json:"DENYLIST_CNAME"`
			RETRIED              int `json:"RETRIED"`
			RETRIEDDNSSEC        int `json:"RETRIED_DNSSEC"`
			INPROGRESS           int `json:"IN_PROGRESS"`
			DBBUSY               int `json:"DBBUSY"`
			SPECIALDOMAIN        int `json:"SPECIAL_DOMAIN"`
			CACHESTALE           int `json:"CACHE_STALE"`
			EXTERNALBLOCKEDEDE15 int `json:"EXTERNAL_BLOCKED_EDE15"`
		} `json:"status"`
		Replies struct {
			UNKNOWN  int `json:"UNKNOWN"`
			NODATA   int `json:"NODATA"`
			NXDOMAIN int `json:"NXDOMAIN"`
			CNAME    int `json:"CNAME"`
			IP       int `json:"IP"`
			DOMAIN   int `json:"DOMAIN"`
			RRNAME   int `json:"RRNAME"`
			SERVFAIL int `json:"SERVFAIL"`
			REFUSED  int `json:"REFUSED"`
			NOTIMP   int `json:"NOTIMP"`
			OTHER    int `json:"OTHER"`
			DNSSEC   int `json:"DNSSEC"`
			NONE     int `json:"NONE"`
			BLOB     int `json:"BLOB"`
		} `json:"replies"`
	} `json:"queries"`
	Clients struct {
		Active int `json:"active"`
		Total  int `json:"total"`
	} `json:"clients"`
	Gravity struct {
		DomainsBeingBlocked int `json:"domains_being_blocked"`
		LastUpdate          int `json:"last_update"`
	} `json:"gravity"`
	Took float64 `json:"took"`
}

// ToString method returns a string of the current statistics struct.
func (s *StatsSummary) String() string {
	return fmt.Sprintf("%d ads blocked / %d total DNS queries", s.Queries.Blocked, s.Queries.Total)
}

func MergeClients(clients1, clients2 []PiHoleClient) []PiHoleClient {
	clientMap := make(map[string]PiHoleClient)

	// Function to add clients to the map
	addClients := func(clients []PiHoleClient) {
		for _, client := range clients {
			key := client.IP // or client.Name if IPs are not unique
			if existing, found := clientMap[key]; found {
				existing.Count += client.Count
				clientMap[key] = existing
			} else {
				clientMap[key] = client
			}
		}
	}

	// Add both arrays
	addClients(clients1)
	addClients(clients2)

	// Convert map back to slice
	mergedClients := make([]PiHoleClient, 0, len(clientMap))
	for _, client := range clientMap {
		mergedClients = append(mergedClients, client)
	}

	return mergedClients
}
