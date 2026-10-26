package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/eko/pihole-exporter/internal/pihole"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
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

		ctx, cancel := context.WithTimeout(request.Context(), 10*time.Second)
		defer cancel()

		collectMetrics(ctx, clients)

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

type collectResult struct {
	hostname string
	err      error
}

// collectMetrics refreshes the metrics of every client concurrently and waits
// for them to finish, or for ctx to be done.
func collectMetrics(ctx context.Context, clients []*pihole.Client) {
	// Each request gets its own channel, buffered so that collections still
	// running after ctx is done can complete without a reader.
	results := make(chan collectResult, len(clients))
	for _, client := range clients {
		go func() {
			results <- collectResult{hostname: client.GetHostname(), err: client.CollectMetrics()}
		}()
	}

	for range clients {
		select {
		case result := <-results:
			if result.err != nil {
				log.Warnf("An error occurred while contacting %s: %+v\n", result.hostname, result.err)
			}
		case <-ctx.Done():
			log.Warnf("Metrics collection did not complete: %v\n", ctx.Err())
			return
		}
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
