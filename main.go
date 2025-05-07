package main

import (
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/eko/pihole-exporter/config"
	"github.com/eko/pihole-exporter/internal/metrics"
	"github.com/eko/pihole-exporter/internal/pihole"
	"github.com/eko/pihole-exporter/internal/server"
	"github.com/xonvanetta/shutdown/pkg/shutdown"
)

func main() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	envConf, clientConfigs, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}
	if envConf.Debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Infof("starting pihole-exporter")

	metrics.Init()

	clients := buildClients(clientConfigs, envConf)
	defer closeClients(clients)

	srv := server.NewServer(envConf.BindAddr, envConf.Port, clients)

	// Context that is cancelled on SIGINT/SIGTERM.
	ctx := shutdown.Context()

	go func() {
		<-ctx.Done()
		srv.Stop()
	}()

	if err := srv.ListenAndServe(); err != nil {
		// Ignore the expected error when the server is closed gracefully.
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}

	log.Info("pihole-exporter HTTP server stopped")
}

// buildClients constructs a slice of Pi‑hole API clients from configuration.
func buildClients(clientConfigs []config.Config, envConfig *config.EnvConfig) []*pihole.Client {
	clients := make([]*pihole.Client, 0, len(clientConfigs))
	for i := range clientConfigs {
		// Use the index variable rather than the for‑range copy to avoid the pointer‑to‑loop‑variable pitfall.
		cfg := &clientConfigs[i]
		clients = append(clients, pihole.NewClient(cfg, envConfig))
	}
	return clients
}

// closeClients closes each client, logging progress.
func closeClients(clients []*pihole.Client) {
	log.Info("closing clients…")
	for _, c := range clients {
		c.Close()
	}
	log.Info("all clients closed")
}
