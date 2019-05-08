package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eko/pihole-exporter/config"
	"github.com/eko/pihole-exporter/internal/metrics"
	"github.com/eko/pihole-exporter/internal/pihole"
	"github.com/eko/pihole-exporter/internal/server"
)

const (
	name = "pihole-exporter"
)

var (
	s *server.Server
)

func main() {
	conf := config.Load()

	metrics.Init()

	initPiholeClient(conf.PIHoleHostname, conf.PIHolePassword, conf.Interval)
	initHttpServer(conf.Port)

	handleExitSignal()
}

func initPiholeClient(hostname, password string, interval time.Duration) {
	client := pihole.NewClient(hostname, password, interval)
	go client.Fetch()
}

func initHttpServer(port string) {
	s = server.NewServer(port)
	go s.ListenAndServe()
}

func handleExitSignal() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	s.Stop()
	fmt.Println(fmt.Sprintf("\n%s HTTP server stopped", name))
}
