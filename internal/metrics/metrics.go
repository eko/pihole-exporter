package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// DomainsBlocked - The number of domains being blocked by PI-Hole.
	DomainsBlocked = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "domains_being_blocked",
			Namespace: "pihole",
			Help:      "This represent the number of domains being blocked",
		},
	)

	// DNSQueriesToday - The number of DNS requests made over PI-Hole over the current day.
	DNSQueriesToday = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "dns_queries_today",
			Namespace: "pihole",
			Help:      "This represent the number of DNS queries made over the current day",
		},
	)

	// AdsBlockedToday - The number of ads blocked by PI-Hole over the current day.
	AdsBlockedToday = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "ads_blocked_today",
			Namespace: "pihole",
			Help:      "This represent the number of ads blocked over the current day",
		},
	)

	// AdsPercentageToday - The percentage of ads blocked by PI-Hole over the current day.
	AdsPercentageToday = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "ads_percentage_today",
			Namespace: "pihole",
			Help:      "This represent the percentage of ads blocked over the current day",
		},
	)

	// UniqueDomains - The number of unique domains seen by PI-Hole.
	UniqueDomains = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "unique_domains",
			Namespace: "pihole",
			Help:      "This represent the number of unique domains seen",
		},
	)

	// QueriesForwarded - The number of queries forwarded by PI-Hole.
	QueriesForwarded = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "queries_forwarded",
			Namespace: "pihole",
			Help:      "This represent the number of queries forwarded",
		},
	)

	// QueriesCached - The number of queries cached by PI-Hole.
	QueriesCached = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "queries_cached",
			Namespace: "pihole",
			Help:      "This represent the number of queries cached",
		},
	)

	// ClientsEverSeen - The number of clients ever seen by PI-Hole.
	ClientsEverSeen = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "clients_ever_seen",
			Namespace: "pihole",
			Help:      "This represent the number of clients ever seen",
		},
	)

	// UniqueClients - The number of unique clients seen by PI-Hole.
	UniqueClients = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "unique_clients",
			Namespace: "pihole",
			Help:      "This represent the number of unique clients seen",
		},
	)

	// DnsQueriesAllTypes - The number of DNS queries made for all types by PI-Hole.
	DnsQueriesAllTypes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "dns_queries_all_types",
			Namespace: "pihole",
			Help:      "This represent the number of DNS queries made for all types",
		},
	)
)

// Init initializes Prometheus metrics
func Init() {
	initMetric(DomainsBlocked)
	initMetric(DNSQueriesToday)
	initMetric(AdsBlockedToday)
	initMetric(AdsPercentageToday)
	initMetric(UniqueDomains)
	initMetric(QueriesForwarded)
	initMetric(QueriesCached)
	initMetric(ClientsEverSeen)
	initMetric(UniqueClients)
	initMetric(DnsQueriesAllTypes)
}

func initMetric(metric prometheus.Gauge) {
	prometheus.MustRegister(metric)
	log.Printf("New prometheus metric registered: %s", metric.Desc().String())
}
