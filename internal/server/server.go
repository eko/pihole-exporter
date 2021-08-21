package server

import (
	"log"
	"net/http"
	"time"

	"github.com/eko/pihole-exporter/internal/pihole"
	"golang.org/x/net/context"
)

// Server is the struct for the HTTP server.
type Server struct {
	httpServer *http.Server
}

// NewServer method initializes a new HTTP server instance and associates
// the different routes that will be used by Prometheus (metrics) or for monitoring (readiness, liveness).
func NewServer(port string, client *pihole.Client) *Server {
	mux := http.NewServeMux()
	httpServer := &http.Server{Addr: ":" + port, Handler: mux}

	s := &Server{
		httpServer: httpServer,
	}

	mux.Handle("/metrics", client.Metrics())
	mux.Handle("/readiness", s.readinessHandler())
	mux.Handle("/liveness", s.livenessHandler())

	return s
}

// ListenAndServe method serves HTTP requests.
func (s *Server) ListenAndServe() {
	log.Println("Starting HTTP server")

	err := s.httpServer.ListenAndServe()
	if err != nil {
		log.Printf("Failed to start serving HTTP requests: %v", err)
	}
}

// Stop method stops the HTTP server (so the exporter become unavailable).
func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	s.httpServer.Shutdown(ctx)
}

func (s *Server) readinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if s.isReady() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
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
