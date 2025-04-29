package config

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend"
	"github.com/heetch/confita/backend/env"
	"github.com/heetch/confita/backend/flags"
)

// Config is the exporter CLI configuration.
type Config struct {
	PIHoleProtocol      string `config:"pihole_protocol"`
	PIHoleHostname      string `config:"pihole_hostname"`
	PIHolePort          uint16 `config:"pihole_port"`
	PIHolePassword      string `config:"pihole_password"`
	BindAddr            string `config:"bind_addr"`
	Port                uint16 `config:"port"`
	SkipTLSVerification bool `config:"skip_tls_verification"`
}

type EnvConfig struct {
	PIHoleProtocol      []string      `config:"pihole_protocol"`
	PIHoleHostname      []string      `config:"pihole_hostname"`
	PIHolePort          []uint16      `config:"pihole_port"`
	PIHolePassword      []string      `config:"pihole_password"`
	BindAddr            string        `config:"bind_addr"`
	Port                uint16        `config:"port"`
	Timeout             time.Duration `config:"timeout"`
	SkipTLSVerification bool `config:"skip_tls_verification"`
	}

func getDefaultEnvConfig() *EnvConfig {
	return &EnvConfig{
		PIHoleProtocol: []string{"http"},
		PIHoleHostname: []string{"127.0.0.1"},
		PIHolePort:     []uint16{80},
		PIHolePassword: []string{},
		BindAddr:       "0.0.0.0",
		Port:           9617,
		Timeout:        5 * time.Second,
		SkipTLSVerification: false,
	}
}

// Load method loads the configuration by using both flag or environment variables.
func Load() (*EnvConfig, []Config, error) {
	loaders := []backend.Backend{
		env.NewBackend(),
		flags.NewBackend(),
	}

	loader := confita.NewLoader(loaders...)

	cfg := getDefaultEnvConfig()
	err := loader.Load(context.Background(), cfg)
	if err != nil {
		log.Fatalf("error returned when passing config into loader.Load(): %v", err)
	}

	cfg.show()

	if clientsConfig, err := cfg.Split(); err != nil {
		return cfg, nil, err
	} else {
		return cfg, clientsConfig, nil
	}
}

func (c *Config) String() string {
	ref := reflect.ValueOf(c)
	fields := ref.Elem()

	buffer := make([]string, fields.NumField(), fields.NumField())
	for i := 0; i < fields.NumField(); i++ {
		valueField := fields.Field(i)
		typeField := fields.Type().Field(i)
		if typeField.Name != "PIHolePassword" {
			buffer[i] = fmt.Sprintf("%s=%v", typeField.Name, valueField.Interface())
		} else if valueField.Len() > 0 {
			buffer[i] = fmt.Sprintf("%s=%s", typeField.Name, "*****")
		}
	}

	buffer = removeEmptyString(buffer)
	return fmt.Sprintf("<Config@%X %s>", &c, strings.Join(buffer, ", "))
}

// Validate check if the config is valid
func (c Config) Validate() error {
	if c.PIHoleProtocol != "http" && c.PIHoleProtocol != "https" {
		return fmt.Errorf("protocol %s is invalid. Must be http or https", c.PIHoleProtocol)
	}
	return nil
}

func (c EnvConfig) Split() ([]Config, error) {
	hostsCount := len(c.PIHoleHostname)
	result := make([]Config, 0, hostsCount)

	for i, hostname := range c.PIHoleHostname {
		config := Config{
			PIHoleHostname: strings.TrimSpace(hostname),
		}

		if len(c.PIHolePort) == 1 {
			config.PIHolePort = c.PIHolePort[0]
		} else if len(c.PIHolePort) == hostsCount {
			config.PIHolePort = c.PIHolePort[i]
		} else if len(c.PIHolePort) != 0 {
			return nil, errors.New("Wrong number of ports. Port can be empty to use default, one value to use for all hosts, or match the number of hosts")
		}

		if hasData, data, isValid := extractStringConfig(c.PIHoleProtocol, i, hostsCount); hasData {
			config.PIHoleProtocol = data
		} else if !isValid {
			return nil, errors.New("Wrong number of PIHoleProtocol. PIHoleProtocol can be empty to use default, one value to use for all hosts, or match the number of hosts")
		}

		if hasData, data, isValid := extractStringConfig(c.PIHolePassword, i, hostsCount); hasData {
			config.PIHolePassword = data
		} else if !isValid {
			return nil, errors.New("Wrong number of PIHolePassword. PIHolePassword can be empty to use default, one value to use for all hosts, or match the number of hosts")
		}

		result = append(result, config)
	}

	return result, nil
}

func extractStringConfig(data []string, idx int, hostsCount int) (bool, string, bool) {
	if len(data) == 1 {
		v := strings.TrimSpace(data[0])
		if v != "" {
			return true, v, true
		}
	} else if len(data) == hostsCount {
		v := strings.TrimSpace(data[idx])
		if v != "" {
			return true, v, true
		}
	} else if len(data) != 0 { //Host count missmatch
		return false, "", false
	}

	// Empty
	return false, "", true
}

func removeEmptyString(source []string) []string {
	var result []string
	for _, s := range source {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func (c EnvConfig) show() {
	val := reflect.ValueOf(&c).Elem()
	log.Info("------------------------------------")
	log.Info("-  Pi-hole exporter configuration  -")
	log.Info("------------------------------------")
	log.Info("Go version: ", runtime.Version())
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)

		// Do not print password or api token but do print the authentication method
		if typeField.Name != "PIHolePassword" {
			log.Info(fmt.Sprintf("%s : %v", typeField.Name, valueField.Interface()))
		} else {
			showAuthenticationMethod(typeField.Name, valueField.Len())
		}
	}
	log.Info("------------------------------------")
}

func showAuthenticationMethod(name string, length int) {
	if length > 0 {
		log.Info(fmt.Sprintf("Pi-hole Authentication Method : %s", name))
	}
}
