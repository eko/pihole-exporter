package main

import (
	"log"

	"github.com/eko/pihole-exporter/config"
	"github.com/eko/pihole-exporter/internal/metrics"
	"github.com/eko/pihole-exporter/internal/pihole"
	"github.com/eko/pihole-exporter/internal/server"
	"github.com/xonvanetta/shutdown/pkg/shutdown"
)

func main() {
	envConf, clientConfigs, err := config.Load()
	if err != nil {
		log.Fatal(err.Error())
	}

	metrics.Init()

	serverDead := make(chan struct{})

	clients := buildClients(clientConfigs)

	s := server.NewServer(envConf.Port, clients)
	go func() {
		s.ListenAndServe()
		close(serverDead)
	}()

	ctx := shutdown.Context()

	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	select {
	case <-ctx.Done():
	case <-serverDead:
	}

	log.Println("pihole-exporter HTTP server stopped")
}

func buildClients(clientConfigs []config.Config) []*pihole.Client {
	clients := make([]*pihole.Client, 0, len(clientConfigs))
	for i := range clientConfigs {
		clientConfig := &clientConfigs[i]

		client := pihole.NewClient(clientConfig)
		clients = append(clients, client)
	}
	return clients
}
