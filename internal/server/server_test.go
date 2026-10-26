package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/eko/pihole-exporter/config"
	"github.com/eko/pihole-exporter/internal/pihole"
)

// fakePihole serves the Pi-hole v6 API endpoints the exporter calls,
// waiting delay before each answer so a collection takes a known time.
func fakePihole(t *testing.T, delay time.Duration) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth":
			_, _ = w.Write([]byte(`{"session":{"valid":true,"sid":"x","validity":1800}}`))
		case "/api/dns/blocking":
			_, _ = w.Write([]byte(`{"blocking":"enabled"}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// startExporter runs NewServer against the given Pi-hole and returns its base URL.
func startExporter(t *testing.T, piholeURL string) string {
	t.Helper()
	u, err := url.Parse(piholeURL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	client := pihole.NewClient(
		&config.Config{PIHoleProtocol: "http", PIHoleHostname: u.Hostname(), PIHolePort: uint16(port), PIHolePassword: "x"},
		&config.EnvConfig{Timeout: 5 * time.Second},
	)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := NewServer("127.0.0.1", 0, []*pihole.Client{client})
	go func() { _ = s.httpServer.Serve(ln) }()
	// Close, not Shutdown: a wedged handler must not keep the test from ending.
	t.Cleanup(func() { _ = s.httpServer.Close() })
	return "http://" + ln.Addr().String()
}

func scrape(baseURL string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/metrics", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

// TestMetricsAfterAbandonedScrape checks that a /metrics request abandoned in the
// middle of a collection (client timeout, or the handler's own 10 s deadline)
// does not make the requests after it hang.
func TestMetricsAfterAbandonedScrape(t *testing.T) {
	// 7 API calls x 40 ms: a collection takes ~0.3 s.
	exporter := startExporter(t, fakePihole(t, 40*time.Millisecond).URL)

	// The first scrape also authenticates.
	if err := scrape(exporter, 3*time.Second); err != nil {
		t.Fatalf("first scrape: %v", err)
	}

	// A client that gives up halfway through the collection.
	if err := scrape(exporter, 100*time.Millisecond); err == nil {
		t.Fatal("expected the 100 ms scrape to time out")
	}
	time.Sleep(500 * time.Millisecond) // let the abandoned collection finish

	for i := 1; i <= 3; i++ {
		start := time.Now()
		if err := scrape(exporter, 3*time.Second); err != nil {
			t.Fatalf("scrape %d after the abandoned one: %v", i, err)
		}
		t.Logf("scrape %d after the abandoned one: %v", i, time.Since(start).Round(time.Millisecond))
	}
}

// TestMetricsAfterConcurrentScrapes checks that overlapping /metrics requests,
// some of them abandoned, do not make the requests after them hang.
func TestMetricsAfterConcurrentScrapes(t *testing.T) {
	exporter := startExporter(t, fakePihole(t, 40*time.Millisecond).URL)

	if err := scrape(exporter, 3*time.Second); err != nil {
		t.Fatalf("first scrape: %v", err)
	}

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		timeout := 3 * time.Second
		if i%2 == 0 {
			timeout = 50 * time.Millisecond
		}
		go func() {
			_ = scrape(exporter, timeout)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	time.Sleep(500 * time.Millisecond) // let the abandoned collections finish

	for i := 1; i <= 3; i++ {
		if err := scrape(exporter, 3*time.Second); err != nil {
			t.Fatalf("scrape %d after the concurrent ones: %v", i, err)
		}
	}
}
