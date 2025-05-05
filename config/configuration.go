package config

import (
	"context"
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
	SkipTLSVerification bool   `config:"skip_tls_verification"`
}

type EnvConfig struct {
	PIHoleProtocol      []string      `config:"pihole_protocol"`
	PIHoleHostname      []string      `config:"pihole_hostname"`
	PIHolePort          []uint16      `config:"pihole_port"`
	PIHolePassword      []string      `config:"pihole_password"`
	BindAddr            string        `config:"bind_addr"`
	Port                uint16        `config:"port"`
	Timeout             time.Duration `config:"timeout"`
	SkipTLSVerification bool          `config:"skip_tls_verification"`
}

const (
	DefaultTimeout = 5 * time.Second
)

func getDefaultEnvConfig() *EnvConfig {
	return &EnvConfig{
		PIHoleProtocol:      []string{"http"},
		PIHoleHostname:      []string{"127.0.0.1"},
		PIHolePort:          []uint16{80},
		PIHolePassword:      []string{},
		BindAddr:            "0.0.0.0",
		Port:                9617,
		Timeout:            DefaultTimeout,
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
		log.Fatalf("error returned when passing config into loader.Load(): %+v", err)
	}

	cfg.show()

	if clientsConfig, err := cfg.Split(); err != nil {
		return cfg, nil, err
	} else {
		return cfg, clientsConfig, nil
	}
}

// String implements fmt.Stringer with a modern strings.Builder implementation.
func (c *Config) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("<Config@%X ", c))
	ref := reflect.ValueOf(c).Elem()
	for i := 0; i < ref.NumField(); i++ {
		tf := ref.Type().Field(i)
		vf := ref.Field(i)
		if tf.Name == "PIHolePassword" {
			if vf.Kind() == reflect.String && vf.Len() > 0 {
				b.WriteString("PIHolePassword=*****,")
			}
			continue
		}
		fmt.Fprintf(&b, "%s=%v,", tf.Name, vf.Interface())
	}
	result := strings.TrimSuffix(b.String(), ",")
	result += ">"
	return result
}

// Validate checks if the config is valid.
func (c Config) Validate() error {
	if c.PIHoleProtocol != "http" && c.PIHoleProtocol != "https" {
		return fmt.Errorf("invalid protocol %s: must be http or https", c.PIHoleProtocol)
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
			return nil, fmt.Errorf("wrong number of ports: must be empty, single value or one per host")
		}

		if hasData, data, isValid := extractStringConfig(c.PIHoleProtocol, i, hostsCount); hasData {
			config.PIHoleProtocol = data
		} else if !isValid {
			return nil, fmt.Errorf("wrong number of PIHoleProtocol: must be empty, single value or one per host")
		}

		if hasData, data, isValid := extractStringConfig(c.PIHolePassword, i, hostsCount); hasData {
			config.PIHolePassword = data
		} else if !isValid {
			return nil, fmt.Errorf("wrong number of PIHolePassword: must be empty, single value or one per host")
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

func (c EnvConfig) show() {
	val := reflect.ValueOf(&c).Elem()
	log.Debug("------------------------------------")
	log.Debug("-  Pi-hole exporter configuration  -")
	log.Debug("------------------------------------")
	log.Debug("Go version: ", runtime.Version())

	for i := range make([]struct{}, val.NumField()) {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)

		// Do not print password but keep authentication method visibility
		if typeField.Name != "PIHolePassword" {
			log.Debugf("%s : %v", typeField.Name, valueField.Interface())
		} else {
			showAuthenticationMethod(typeField.Name, valueField.Len())
		}
	}
	log.Debug("------------------------------------")
}

func showAuthenticationMethod(name string, length int) {
	if length > 0 {
		log.Debugf("Pi-hole Authentication Method: %s", name)
	}
}
