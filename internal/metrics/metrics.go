package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	// DomainsBlocked - The number of domains being blocked by Pi-hole.
	DomainsBlocked = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "domains_being_blocked",
			Namespace: "pihole",
			Help:      "This represent the number of domains being blocked",
		},
		[]string{"hostname"},
	)

	// DNSQueriesToday - The number of DNS requests made over Pi-hole over the current day.
	DNSQueriesToday = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "dns_queries_today",
			Namespace: "pihole",
			Help:      "This represent the number of DNS queries made over the current day",
		},
		[]string{"hostname"},
	)

	// AdsBlockedToday - The number of ads blocked by Pi-hole over the current day.
	AdsBlockedToday = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "ads_blocked_today",
			Namespace: "pihole",
			Help:      "This represent the number of ads blocked over the current day",
		},
		[]string{"hostname"},
	)

	// AdsPercentageToday - The percentage of ads blocked by Pi-hole over the current day.
	AdsPercentageToday = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "ads_percentage_today",
			Namespace: "pihole",
			Help:      "This represent the percentage of ads blocked over the current day",
		},
		[]string{"hostname"},
	)

	// UniqueDomains - The number of unique domains seen by Pi-hole.
	UniqueDomains = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "unique_domains",
			Namespace: "pihole",
			Help:      "This represent the number of unique domains seen",
		},
		[]string{"hostname"},
	)

	// QueriesForwarded - The number of queries forwarded by Pi-hole.
	QueriesForwarded = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "queries_forwarded",
			Namespace: "pihole",
			Help:      "This represent the number of queries forwarded",
		},
		[]string{"hostname"},
	)

	// QueriesCached - The number of queries cached by Pi-hole.
	QueriesCached = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "queries_cached",
			Namespace: "pihole",
			Help:      "This represent the number of queries cached",
		},
		[]string{"hostname"},
	)

	// ClientsEverSeen - The number of clients ever seen by Pi-hole.
	ClientsEverSeen = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "clients_ever_seen",
			Namespace: "pihole",
			Help:      "This represent the number of clients ever seen",
		},
		[]string{"hostname"},
	)

	// UniqueClients - The number of unique clients seen by Pi-hole.
	UniqueClients = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "unique_clients",
			Namespace: "pihole",
			Help:      "This represent the number of unique clients seen",
		},
		[]string{"hostname"},
	)

	// DNSQueriesAllTypes - The number of DNS queries made for all types by Pi-hole.
	DNSQueriesAllTypes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "dns_queries_all_types",
			Namespace: "pihole",
			Help:      "This represent the number of DNS queries made for all types",
		},
		[]string{"hostname"},
	)

	// Reply - The number of replies made for every types by Pi-hole.
	Reply = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "reply",
			Namespace: "pihole",
			Help:      "This represent the number of replies made for all types",
		},
		[]string{"hostname", "type"},
	)

	// TopQueries - The number of top queries made by Pi-hole by domain.
	TopQueries = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "top_queries",
			Namespace: "pihole",
			Help:      "This represent the number of top queries made by Pi-hole by domain",
		},
		[]string{"hostname", "domain"},
	)

	// TopAds - The number of top ads made by Pi-hole by domain.
	TopAds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "top_ads",
			Namespace: "pihole",
			Help:      "This represent the number of top ads made by Pi-hole by domain",
		},
		[]string{"hostname", "domain"},
	)

	// TopSources - The number of top sources requests made by Pi-hole by source host.
	TopSources = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "top_sources",
			Namespace: "pihole",
			Help:      "This represent the number of top sources requests made by Pi-hole by source host",
		},
		[]string{"hostname", "source", "source_name"},
	)

	// ForwardDestinations - The number of forward destinations requests made by Pi-hole by destination.
	ForwardDestinations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "forward_destinations",
			Namespace: "pihole",
			Help:      "This represent the number of forward destinations requests made by Pi-hole by destination",
		},
		[]string{"hostname", "destination", "destination_name"},
	)

	// QueryTypes - The number of queries made by Pi-hole by type.
	QueryTypes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "querytypes",
			Namespace: "pihole",
			Help:      "This represent the number of queries made by Pi-hole by type",
		},
		[]string{"hostname", "type"},
	)

	// Status - Is Pi-hole enabled?
	Status = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "status",
			Namespace: "pihole",
			Help:      "This if Pi-hole is enabled",
		},
		[]string{"hostname"},
	)
)

// Init initializes all Prometheus metrics made available by Pi-hole exporter.
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
	log.Info("New Prometheus metric registered: ", name)
}
