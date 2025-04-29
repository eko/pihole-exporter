package pihole_test

import (
	"net/http"
	"testing"
	"time"

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
