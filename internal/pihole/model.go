package pihole

// Stats is the PI-Hole statistics JSON API corresponding model
type Stats struct {
	DomainsBeingBlocked int     `json:"domains_being_blocked"`
	DNSQueriesToday     int     `json:"dns_queries_today"`
	AdsBlockedToday     int     `json:"ads_blocked_today"`
	AdsPercentageToday  float64 `json:"ads_percentage_today"`
	UniqueDomains       int     `json:"unique_domains"`
	QueriesForwarded    int     `json:"queries_forwarded"`
	QueriesCached       int     `json:"queries_cached"`
	ClientsEverSeen     int     `json:"clients_ever_seen"`
	UniqueClients       int     `json:"unique_clients"`
	DnsQueriesAllTypes  int     `json:"dns_queries_all_types"`
}
