package server

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/eko/pihole-exporter/internal/pihole"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// Server is the struct for the HTTP server.
type Server struct {
	httpServer *http.Server
}

// NewServer method initializes a new HTTP server instance and associates
// the different routes that will be used by Prometheus (metrics) or for monitoring (readiness, liveness).
func NewServer(addr string, port uint16, clients []*pihole.Client) *Server {
	mux := http.NewServeMux()
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", addr, port),
		Handler: mux,
	}

	s := &Server{
		httpServer: httpServer,
	}

	mux.HandleFunc("/metrics", func(writer http.ResponseWriter, request *http.Request) {
		log.Debugf("request.Header: %+v\n", request.Header)

		// Use a WaitGroup to track goroutines
		var wg sync.WaitGroup
		// Create a context with timeout for metrics collection
		ctx, cancel := context.WithTimeout(request.Context(), 10*time.Second)
		defer cancel()

		// Channel to collect results from goroutines
		resultChan := make(chan *pihole.ClientChannel, len(clients))

		for _, client := range clients {
			wg.Add(1)
			go func(c *pihole.Client) {
				defer wg.Done()

				// Create a channel for this goroutine
				doneChan := make(chan struct{})

				go func() {
					c.CollectMetricsAsync(writer, request)
					close(doneChan)
				}()

				// Wait for either completion or timeout
				select {
				case <-doneChan:
					// Normal completion, status will be read below
				case <-ctx.Done():
					// Timeout occurred
					resultChan <- &pihole.ClientChannel{
						Status: pihole.MetricsCollectionTimeout,
						Err:    fmt.Errorf("metrics collection from %s timed out", c.GetHostname()),
					}
					// We need to read from the Status channel to prevent blocking
					go func() {
						<-c.Status // Discard the result when it eventually comes
					}()
				}
			}(client)
		}

		// Start a goroutine to close resultChan when all goroutines are done
		go func() {
			wg.Wait()
			close(resultChan)
		}()

		// Read all results
		for _, client := range clients {
			status := <-client.Status
			if status.Status != pihole.MetricsCollectionSuccess {
				log.Warnf("An error occurred while contacting %s: %+v\n", client.GetHostname(), status.Err)
			}
		}

		promhttp.Handler().ServeHTTP(writer, request)
	})

	mux.Handle("/readiness", s.readinessHandler())
	mux.Handle("/liveness", s.livenessHandler())

	return s
}

// ListenAndServe method serves HTTP requests.
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

// Stop method stops the HTTP server (so the exporter become unavailable).
func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	s.httpServer.Shutdown(ctx)
}

// handleMetrics, helper function is unused
func (s *Server) handleMetrics(clients []*pihole.Client) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		errors := make([]string, 0)

		for _, client := range clients {
			if err := client.CollectMetrics(writer, request); err != nil {
				errors = append(errors, err.Error())
				log.Warnf("error collecting metrics from %s: %+v\n", client.GetHostname(), err)
			}
		}

		if len(errors) == len(clients) {
			writer.WriteHeader(http.StatusBadRequest)
			body := strings.Join(errors, "\n")
			_, _ = writer.Write([]byte(body))
		}

		promhttp.Handler().ServeHTTP(writer, request)
	}
}

func (s *Server) readinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		status := http.StatusNotFound
		if s.isReady() {
			status = http.StatusOK
		}
		w.WriteHeader(status)
	}
}

func (s *Server) livenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) isReady() bool {
	return s.httpServer != nil
}
