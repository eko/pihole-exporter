package pihole_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eko/pihole-exporter/internal/pihole"
)

// transport returns the transport of the client
func transport(t *testing.T, c *pihole.APIClient) *http.Transport {
	t.Helper()
	tr, ok := c.Client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Transport not *http.Transport")
	}
	if tr.TLSClientConfig == nil {
		t.Fatalf("nil TLSClientConfig")
	}
	return tr
}

// TestNewAPIClient_TLSVerification tests the TLS verification of the client
func TestNewAPIClient_TLSVerification(t *testing.T) {
	tests := []struct {
		name             string
		skipVerify       bool
		wantInsecureSkip bool
	}{
		{"disabled", true, true},
		{"enabled", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := pihole.NewAPIClient("https://cloudflare.com", "", time.Second, tc.skipVerify)
			if got := transport(t, c).TLSClientConfig.InsecureSkipVerify; got != tc.wantInsecureSkip {
				t.Errorf("InsecureSkipVerify = %v, want %v", got, tc.wantInsecureSkip)
			}
		})
	}
}

// fakePihole is a minimal Pi-hole v6 API that counts authentications and
// only accepts the most recently issued session ID.
type fakePihole struct {
	mu         sync.Mutex
	validity   int
	noPassword bool
	auths      int
	validSID   string
	logouts    []string
}

func (f *fakePihole) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case r.URL.Path == "/api/auth" && r.Method == http.MethodPost:
		f.auths++
		if f.noPassword {
			f.validSID = ""
			_, _ = w.Write([]byte(`{"session":{"valid":true,"sid":null,"validity":-1}}`))
			return
		}
		f.validSID = fmt.Sprintf("sid-%d", f.auths)
		_, _ = fmt.Fprintf(w, `{"session":{"valid":true,"sid":%q,"validity":%d}}`, f.validSID, f.validity)
	case r.URL.Path == "/api/auth" && r.Method == http.MethodDelete:
		f.logouts = append(f.logouts, r.Header.Get("X-FTL-SID"))
		w.WriteHeader(http.StatusNoContent)
	case r.Header.Get("X-FTL-SID") != f.validSID:
		w.WriteHeader(http.StatusUnauthorized)
	default:
		_, _ = w.Write([]byte(`{}`))
	}
}

// expireSession makes the fake reject the current session, as after a Pi-hole restart.
func (f *fakePihole) expireSession() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.validSID = "expired"
}

func (f *fakePihole) authCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.auths
}

func newFakeAPIClient(
	t *testing.T,
	fake *fakePihole,
) *pihole.APIClient {
	t.Helper()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	return pihole.NewAPIClient(srv.URL, "secret", time.Second, false)
}

func TestFetchDataReusesSession(t *testing.T) {
	testCases := []struct {
		name string
		fake *fakePihole
	}{
		{
			name: "validity announced",
			fake: &fakePihole{validity: 300},
		},
		{
			name: "zero validity",
			fake: &fakePihole{validity: 0},
		},
		{
			name: "no password set",
			fake: &fakePihole{noPassword: true},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client := newFakeAPIClient(t, testCase.fake)

			// 5 scrapes of 7 API calls each.
			for i := 0; i < 35; i++ {
				var result map[string]any
				require.NoError(t, client.FetchData("/api/stats/summary", &result))
			}

			assert.Equal(t, 1, testCase.fake.authCount())
		})
	}
}

func TestFetchDataRenewsRejectedSession(t *testing.T) {
	fake := &fakePihole{validity: 1800}
	client := newFakeAPIClient(t, fake)

	var result map[string]any
	require.NoError(t, client.FetchData("/api/stats/summary", &result))

	fake.expireSession()

	require.NoError(t, client.FetchData("/api/stats/summary", &result))
	assert.Equal(t, 2, fake.authCount())
}

func TestFetchDataRenewsSessionOnceForConcurrentCalls(t *testing.T) {
	fake := &fakePihole{validity: 1800}
	client := newFakeAPIClient(t, fake)

	var result map[string]any
	require.NoError(t, client.FetchData("/api/stats/summary", &result))

	fake.expireSession()

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var result map[string]any
			errs <- client.FetchData("/api/stats/summary", &result)
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		assert.NoError(t, err)
	}
	assert.Equal(t, 2, fake.authCount())
}

func TestCloseLogsOut(t *testing.T) {
	fake := &fakePihole{validity: 1800}
	client := newFakeAPIClient(t, fake)

	var result map[string]any
	require.NoError(t, client.FetchData("/api/stats/summary", &result))

	client.Close()

	fake.mu.Lock()
	defer fake.mu.Unlock()
	assert.Equal(t, []string{"sid-1"}, fake.logouts)
}
