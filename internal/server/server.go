package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eko/pihole-exporter/internal/pihole"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/context"
)

// Server is the struct for the HTTP server.
type Server struct {
	httpServer *http.Server
}

// NewServer method initializes a new HTTP server instance and associates
// the different routes that will be used by Prometheus (metrics) or for monitoring (readiness, liveness).
func NewServer(port uint16, clients []*pihole.Client) *Server {
	mux := http.NewServeMux()
	httpServer := &http.Server{Addr: ":" + strconv.Itoa(int(port)), Handler: mux}

	s := &Server{
		httpServer: httpServer,
	}

	/*fmt.Printf("Server received clients -> %s\n", clients)
	for i, client := range clients {
		fmt.Printf("Server received clients -> idx: %d, Hostname: %s\n", i, &client)
	}*/

	mux.HandleFunc("/metrics",
		func(writer http.ResponseWriter, request *http.Request) {
			errors := make([]string, 0)

			for _, client := range clients {
				if err := client.CollectMetrics(writer, request); err != nil {
					errors = append(errors, err.Error())
					fmt.Printf("Error %s\n", err)
				}
			}

			if len(errors) == len(clients) {
				writer.WriteHeader(http.StatusBadRequest)
				body := strings.Join(errors, "\n")
				_, _ = writer.Write([]byte(body))
			}

			promhttp.Handler().ServeHTTP(writer, request)
		},
	)

	//mux.Handle("/metrics", client.Metrics())
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
