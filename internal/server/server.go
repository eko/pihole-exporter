package server

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/context"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(port string) *Server {
	mux := http.NewServeMux()
	httpServer := &http.Server{Addr: ":" + port, Handler: mux}

	s := &Server{
		httpServer: httpServer,
	}

	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/readiness", s.readinessHandler())
	mux.Handle("/liveness", s.livenessHandler())

	return s
}

func (s *Server) ListenAndServe() {
	log.Println("Starting HTTP server")

	err := s.httpServer.ListenAndServe()
	if err != nil {
		log.Printf("Failed to start serving HTTP requests: %v", err)
	}
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	s.httpServer.Shutdown(ctx)
}

func (s *Server) readinessHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if s.isReady() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func (s *Server) livenessHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func (s *Server) isReady() bool {
	return s.httpServer != nil
}
