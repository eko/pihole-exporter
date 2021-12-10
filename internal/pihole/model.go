package pihole

import "fmt"

const (
	enabledStatus = "enabled"
)

// Stats struct is the PI-Hole statistics JSON API corresponding model.
type Stats struct {
	DomainsBeingBlocked int                `json:"domains_being_blocked"`
	DNSQueriesToday     int                `json:"dns_queries_today"`
	AdsBlockedToday     int                `json:"ads_blocked_today"`
	AdsPercentageToday  float64            `json:"ads_percentage_today"`
	UniqueDomains       int                `json:"unique_domains"`
	QueriesForwarded    int                `json:"queries_forwarded"`
	QueriesCached       int                `json:"queries_cached"`
	ClientsEverSeen     int                `json:"clients_ever_seen"`
	UniqueClients       int                `json:"unique_clients"`
	DNSQueriesAllTypes  int                `json:"dns_queries_all_types"`
	ReplyNoData         int                `json:"reply_NODATA"`
	ReplyNxDomain       int                `json:"reply_NXDOMAIN"`
	ReplyCname          int                `json:"reply_CNAME"`
	ReplyIP             int                `json:"reply_IP"`
	TopQueries          map[string]int     `json:"top_queries"`
	TopAds              map[string]int     `json:"top_ads"`
	TopSources          map[string]int     `json:"top_sources"`
	ForwardDestinations map[string]float64 `json:"forward_destinations"`
	QueryTypes          map[string]float64 `json:"querytypes"`
	Status              string             `json:"status"`
}

// ToString method returns a string of the current statistics struct.
func (s *Stats) String() string {
	return fmt.Sprintf("%d ads blocked / %d total DNS queries", s.AdsBlockedToday, s.DNSQueriesAllTypes)
}
