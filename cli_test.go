package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eko/pihole-exporter/config"
)

func TestTakeMode(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		wantMode mode
		wantArgs []string
	}{
		{
			name:     "no arguments",
			args:     []string{"pihole-exporter"},
			wantMode: modeServe,
			wantArgs: []string{"pihole-exporter"},
		},
		{
			name:     "configuration flags",
			args:     []string{"pihole-exporter", "-port", "9000"},
			wantMode: modeServe,
			wantArgs: []string{"pihole-exporter", "-port", "9000"},
		},
		{
			name:     "version",
			args:     []string{"pihole-exporter", "--version"},
			wantMode: modeVersion,
			wantArgs: []string{"pihole-exporter"},
		},
		{
			name:     "healthcheck with flags",
			args:     []string{"pihole-exporter", "-healthcheck", "-port", "9000"},
			wantMode: modeHealthcheck,
			wantArgs: []string{"pihole-exporter", "-port", "9000"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			originalArgs := os.Args
			t.Cleanup(func() { os.Args = originalArgs })
			os.Args = testCase.args

			assert.Equal(t, testCase.wantMode, takeMode())
			assert.Equal(t, testCase.wantArgs, os.Args)
		})
	}
}

func TestHealthcheck(t *testing.T) {
	testCases := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{
			name:   "live",
			status: http.StatusOK,
		},
		{
			name:    "unexpected status",
			status:  http.StatusServiceUnavailable,
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/liveness", r.URL.Path)
				w.WriteHeader(testCase.status)
			}))
			t.Cleanup(srv.Close)

			_, portStr, err := net.SplitHostPort(srv.Listener.Addr().String())
			require.NoError(t, err)
			port, err := strconv.Atoi(portStr)
			require.NoError(t, err)

			err = healthcheck(&config.EnvConfig{BindAddr: "0.0.0.0", Port: uint16(port)})
			if testCase.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
