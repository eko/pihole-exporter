package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// DomainsBlocked - The number of domains being blocked by PI-Hole.
	DomainsBlocked = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "domains_being_blocked",
			Namespace: "pihole",
			Help:      "This represent the number of domains being blocked",
		},
		[]string{"hostname"},
	)

	// DNSQueriesToday - The number of DNS requests made over PI-Hole over the current day.
	DNSQueriesToday = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "dns_queries_today",
			Namespace: "pihole",
			Help:      "This represent the number of DNS queries made over the current day",
		},
		[]string{"hostname"},
	)

	// AdsBlockedToday - The number of ads blocked by PI-Hole over the current day.
	AdsBlockedToday = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "ads_blocked_today",
			Namespace: "pihole",
			Help:      "This represent the number of ads blocked over the current day",
		},
		[]string{"hostname"},
	)

	// AdsPercentageToday - The percentage of ads blocked by PI-Hole over the current day.
	AdsPercentageToday = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "ads_percentage_today",
			Namespace: "pihole",
			Help:      "This represent the percentage of ads blocked over the current day",
		},
		[]string{"hostname"},
	)

	// UniqueDomains - The number of unique domains seen by PI-Hole.
	UniqueDomains = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "unique_domains",
			Namespace: "pihole",
			Help:      "This represent the number of unique domains seen",
		},
		[]string{"hostname"},
	)

	// QueriesForwarded - The number of queries forwarded by PI-Hole.
	QueriesForwarded = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "queries_forwarded",
			Namespace: "pihole",
			Help:      "This represent the number of queries forwarded",
		},
		[]string{"hostname"},
	)

	// QueriesCached - The number of queries cached by PI-Hole.
	QueriesCached = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "queries_cached",
			Namespace: "pihole",
			Help:      "This represent the number of queries cached",
		},
		[]string{"hostname"},
	)

	// ClientsEverSeen - The number of clients ever seen by PI-Hole.
	ClientsEverSeen = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "clients_ever_seen",
			Namespace: "pihole",
			Help:      "This represent the number of clients ever seen",
		},
		[]string{"hostname"},
	)

	// UniqueClients - The number of unique clients seen by PI-Hole.
	UniqueClients = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "unique_clients",
			Namespace: "pihole",
			Help:      "This represent the number of unique clients seen",
		},
		[]string{"hostname"},
	)

	// DNSQueriesAllTypes - The number of DNS queries made for all types by PI-Hole.
	DNSQueriesAllTypes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "dns_queries_all_types",
			Namespace: "pihole",
			Help:      "This represent the number of DNS queries made for all types",
		},
		[]string{"hostname"},
	)

	// Reply - The number of replies made for every types by PI-Hole.
	Reply = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "reply",
			Namespace: "pihole",
			Help:      "This represent the number of replies made for all types",
		},
		[]string{"hostname", "type"},
	)

	// TopQueries - The number of top queries made by PI-Hole by domain.
	TopQueries = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "top_queries",
			Namespace: "pihole",
			Help:      "This represent the number of top queries made by PI-Hole by domain",
		},
		[]string{"hostname", "domain"},
	)

	// TopAds - The number of top ads made by PI-Hole by domain.
	TopAds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "top_ads",
			Namespace: "pihole",
			Help:      "This represent the number of top ads made by PI-Hole by domain",
		},
		[]string{"hostname", "domain"},
	)

	// TopSources - The number of top sources requests made by PI-Hole by source host.
	TopSources = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "top_sources",
			Namespace: "pihole",
			Help:      "This represent the number of top sources requests made by PI-Hole by source host",
		},
		[]string{"hostname", "source"},
	)

	// ForwardDestinations - The number of forward destinations requests made by PI-Hole by destination.
	ForwardDestinations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "forward_destinations",
			Namespace: "pihole",
			Help:      "This represent the number of forward destinations requests made by PI-Hole by destination",
		},
		[]string{"hostname", "destination"},
	)

	// QueryTypes - The number of queries made by PI-Hole by type.
	QueryTypes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "querytypes",
			Namespace: "pihole",
			Help:      "This represent the number of queries made by PI-Hole by type",
		},
		[]string{"hostname", "type"},
	)

	// Status - Is PI-Hole enabled?
	Status = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "status",
			Namespace: "pihole",
			Help:      "This if PI-Hole is enabled",
		},
		[]string{"hostname"},
	)
)

// Init initializes all Prometheus metrics made available by PI-Hole exporter.
func Init() {
	initMetric("domains_blocked", DomainsBlocked)
	initMetric("dns_queries_today", DNSQueriesToday)
	initMetric("ads_blocked_today", AdsBlockedToday)
	initMetric("ads_percentag_today", AdsPercentageToday)
	initMetric("unique_domains", UniqueDomains)
	initMetric("queries_forwarded", QueriesForwarded)
	initMetric("queries_cached", QueriesCached)
	initMetric("clients_ever_seen", ClientsEverSeen)
	initMetric("unique_clients", UniqueClients)
	initMetric("dns_queries_all_types", DNSQueriesAllTypes)
	initMetric("reply", Reply)
	initMetric("top_queries", TopQueries)
	initMetric("top_ads", TopAds)
	initMetric("top_sources", TopSources)
	initMetric("forward_destinations", ForwardDestinations)
	initMetric("querytypes", QueryTypes)
	initMetric("status", Status)
}

func initMetric(name string, metric *prometheus.GaugeVec) {
	prometheus.MustRegister(metric)
	log.Printf("New Prometheus metric registered: %s", name)
}
