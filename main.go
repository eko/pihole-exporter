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
	conf := config.Load()

	metrics.Init()

	serverDead := make(chan struct{})
	s := server.NewServer(conf.Port, pihole.NewClient(conf))
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
