package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/eko/pihole-exporter/config"
	"github.com/eko/pihole-exporter/internal/metrics"
	"github.com/eko/pihole-exporter/internal/pihole"
	"github.com/eko/pihole-exporter/internal/server"
	"github.com/xonvanetta/shutdown/pkg/shutdown"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	mode := takeMode()
	if mode == modeVersion {
		fmt.Printf("pihole-exporter %s\n", version)
		return
	}

	envConf, clientConfigs, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}
	if err := setupLogging(envConf.LogLevel, envConf.Debug); err != nil {
		log.Fatalf("failed to set up logging: %v", err)
	}

	if mode == modeHealthcheck {
		if err := healthcheck(envConf); err != nil {
			log.Fatalf("healthcheck failed: %v", err)
		}
		return
	}

	envConf.Show()
	log.Infof("starting pihole-exporter %s", version)

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

	log.Infof("listening on %s", net.JoinHostPort(envConf.BindAddr, strconv.Itoa(int(envConf.Port))))
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
