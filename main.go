package main

import (
	"fmt"

	"github.com/eko/pihole-exporter/config"
	"github.com/eko/pihole-exporter/internal/metrics"
	"github.com/eko/pihole-exporter/internal/pihole"
	"github.com/eko/pihole-exporter/internal/server"
	"github.com/xonvanetta/shutdown/pkg/shutdown"
)

func main() {
	envConf, clientConfigs := config.Load()

	metrics.Init()

	serverDead := make(chan struct{})
	clients := make([]*pihole.Client, 0, len(clientConfigs))
	for i, _ := range clientConfigs {
		client := pihole.NewClient(&clientConfigs[i])
		clients = append(clients, client)
		fmt.Printf("Append client %s\n", clients)
	}

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

	fmt.Println("pihole-exporter HTTP server stopped")
}
