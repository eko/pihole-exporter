package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/eko/pihole-exporter/config"
)

type mode int

const (
	modeServe mode = iota
	modeVersion
	modeHealthcheck
)

// takeMode returns the mode selected by the first command line argument and
// removes that argument, so that the remaining ones can be parsed as configuration flags.
func takeMode() mode {
	if len(os.Args) < 2 {
		return modeServe
	}

	var selected mode
	switch os.Args[1] {
	case "-version", "--version":
		selected = modeVersion
	case "-healthcheck", "--healthcheck":
		selected = modeHealthcheck
	default:
		return modeServe
	}

	os.Args = append(os.Args[:1], os.Args[2:]...)
	return selected
}

// healthcheck checks that the exporter configured by envConf answers on its
// liveness endpoint. The Docker image has no shell nor HTTP client, so the
// binary itself is used as the container health check.
func healthcheck(envConf *config.EnvConfig) error {
	host := envConf.BindAddr
	if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
		host = "127.0.0.1"
	}
	url := fmt.Sprintf("http://%s/liveness", net.JoinHostPort(host, strconv.Itoa(int(envConf.Port))))

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, url)
	}
	return nil
}
