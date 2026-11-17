package pihole_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eko/pihole-exporter/config"
	"github.com/eko/pihole-exporter/internal/metrics"
	"github.com/eko/pihole-exporter/internal/pihole"
)

// statsPihole serves canned Pi-hole v6 API responses keyed by request URI.
type statsPihole struct {
	mu        sync.Mutex
	responses map[string]string
}

func (s *statsPihole) set(
	requestURI string,
	body string,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses[requestURI] = body
}

func (s *statsPihole) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r.URL.Path == "/api/auth" {
		_, _ = w.Write([]byte(`{"session":{"valid":true,"sid":"sid","validity":1800}}`))
		return
	}
	body, ok := s.responses[r.URL.RequestURI()]
	if !ok {
		body = `{}`
	}
	_, _ = w.Write([]byte(body))
}

func newStatsClient(
	t *testing.T,
	fake *statsPihole,
) *pihole.Client {
	t.Helper()
	return newStatsClientWithConfig(t, fake, config.EnvConfig{})
}

func newStatsClientWithConfig(
	t *testing.T,
	fake *statsPihole,
	envConfig config.EnvConfig,
) *pihole.Client {
	t.Helper()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)

	envConfig.Timeout = time.Second
	return pihole.NewClient(
		&config.Config{PIHoleProtocol: "http", PIHoleHostname: u.Hostname(), PIHolePort: uint16(port), PIHolePassword: "x"},
		&envConfig,
	)
}

func TestCollectMetricsStatus(t *testing.T) {
	testCases := []struct {
		name     string
		blocking string
		want     float64
	}{
		{
			name:     "enabled",
			blocking: `{"blocking":"enabled","timer":null,"took":0.001}`,
			want:     1,
		},
		{
			name:     "disabled",
			blocking: `{"blocking":"disabled","timer":null,"took":0.001}`,
			want:     0,
		},
		{
			name:     "disabled with a timer",
			blocking: `{"blocking":"disabled","timer":29.97,"took":0.001}`,
			want:     0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fake := &statsPihole{responses: map[string]string{"/api/dns/blocking": testCase.blocking}}
			client := newStatsClient(t, fake)

			require.NoError(t, client.CollectMetrics())
			assert.Equal(t, testCase.want, testutil.ToFloat64(metrics.Status.WithLabelValues(client.GetHostname())))
		})
	}
}

func TestCollectMetricsDropsStaleTopEntries(t *testing.T) {
	fake := &statsPihole{responses: map[string]string{}}
	client := newStatsClient(t, fake)

	fake.set("/api/stats/top_domains?blocked=true&count=10", `{"domains":[{"domain":"old.example","count":3}]}`)
	require.NoError(t, client.CollectMetrics())

	fake.set("/api/stats/top_domains?blocked=true&count=10", `{"domains":[{"domain":"new.example","count":5}]}`)
	require.NoError(t, client.CollectMetrics())

	assert.Equal(t, 1, testutil.CollectAndCount(metrics.TopAds))
	assert.Equal(t, 5.0, testutil.ToFloat64(metrics.TopAds.WithLabelValues(client.GetHostname(), "new.example")))
}

func TestCollectMetricsTopSources(t *testing.T) {
	testCases := []struct {
		name        string
		envConfig   config.EnvConfig
		count       string
		wantSources map[string]float64
		wantBlocked map[string]float64
	}{
		{
			name:  "default count",
			count: "10",
			// The laptop is in both lists, the phone only in the blocked one.
			wantSources: map[string]float64{"laptop": 2900, "tv": 50},
			wantBlocked: map[string]float64{"laptop": 426, "phone": 3},
		},
		{
			name:        "configured count",
			envConfig:   config.EnvConfig{TopClientsCount: 50},
			count:       "50",
			wantSources: map[string]float64{"laptop": 2900, "tv": 50},
			wantBlocked: map[string]float64{"laptop": 426, "phone": 3},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			metrics.TopSources.Reset()
			metrics.TopSourcesBlocked.Reset()

			fake := &statsPihole{responses: map[string]string{
				"/api/stats/top_clients?blocked=false&count=" + testCase.count: `{"clients":[` +
					`{"ip":"10.0.0.2","name":"laptop","count":2900},{"ip":"10.0.0.3","name":"tv","count":50}]}`,
				"/api/stats/top_clients?blocked=true&count=" + testCase.count: `{"clients":[` +
					`{"ip":"10.0.0.2","name":"laptop","count":426},{"ip":"10.0.0.4","name":"phone","count":3}]}`,
			}}
			client := newStatsClientWithConfig(t, fake, testCase.envConfig)

			require.NoError(t, client.CollectMetrics())

			ips := map[string]string{"laptop": "10.0.0.2", "tv": "10.0.0.3", "phone": "10.0.0.4"}
			for name, want := range testCase.wantSources {
				assert.Equal(t, want, testutil.ToFloat64(metrics.TopSources.WithLabelValues(client.GetHostname(), ips[name], name)), name)
			}
			for name, want := range testCase.wantBlocked {
				assert.Equal(t, want, testutil.ToFloat64(metrics.TopSourcesBlocked.WithLabelValues(client.GetHostname(), ips[name], name)), name)
			}
			assert.Equal(t, len(testCase.wantSources), testutil.CollectAndCount(metrics.TopSources))
		})
	}
}

func TestCollectMetricsLast10min(t *testing.T) {
	// Slots are 10 minutes long and timestamped in their middle. Pi-hole returns
	// the slot in progress and, right before its garbage collection, the next one.
	now := time.Now().Unix()
	current := now - now%600 + 300
	history := fmt.Sprintf(`{"history":[
		{"timestamp":%d,"total":10,"blocked":1},
		{"timestamp":%d,"total":20,"blocked":2},
		{"timestamp":%d,"total":5,"blocked":1},
		{"timestamp":%d,"total":0,"blocked":0}
	]}`, current-1200, current-600, current, current+600)

	fake := &statsPihole{responses: map[string]string{"/api/history": history}}
	client := newStatsClient(t, fake)

	require.NoError(t, client.CollectMetrics())

	assert.Equal(t, 20.0, testutil.ToFloat64(metrics.QueriesLast10min.WithLabelValues(client.GetHostname())))
	assert.Equal(t, 2.0, testutil.ToFloat64(metrics.AdsLast10min.WithLabelValues(client.GetHostname())))
}
