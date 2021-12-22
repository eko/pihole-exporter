package config

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend"
	"github.com/heetch/confita/backend/env"
	"github.com/heetch/confita/backend/flags"
)

// Config is the exporter CLI configuration.
type Config struct {
	PIHoleProtocol string `config:"pihole_protocol"`
	PIHoleHostname string `config:"pihole_hostname"`
	PIHolePort     uint16 `config:"pihole_port"`
	PIHolePassword string `config:"pihole_password"`
	PIHoleApiToken string `config:"pihole_api_token"`
}

type EnvConfig struct {
	PIHoleProtocol []string      `config:"pihole_protocol"`
	PIHoleHostname []string      `config:"pihole_hostname"`
	PIHolePort     []uint16      `config:"pihole_port"`
	PIHolePassword []string      `config:"pihole_password"`
	PIHoleApiToken []string      `config:"pihole_api_token"`
	Port           uint16        `config:"port"`
	Interval       time.Duration `config:"interval"`
}

func getDefaultEnvConfig() *EnvConfig {
	return &EnvConfig{
		PIHoleProtocol: []string{"http"},
		PIHoleHostname: []string{"127.0.0.1"},
		PIHolePort:     []uint16{80},
		PIHolePassword: []string{},
		PIHoleApiToken: []string{},
		Port:           9617,
		Interval:       10 * time.Second,
	}
}

// Load method loads the configuration by using both flag or environment variables.
func Load() (*EnvConfig, []Config) {
	loaders := []backend.Backend{
		env.NewBackend(),
		flags.NewBackend(),
	}

	loader := confita.NewLoader(loaders...)

	cfg := getDefaultEnvConfig()
	err := loader.Load(context.Background(), cfg)
	if err != nil {
		panic(err)
	}

	cfg.show()

	return cfg, cfg.Split()
}

func (c *Config) String() string {
	ref := reflect.ValueOf(c)
	fields := ref.Elem()

	buffer := make([]string, fields.NumField(), fields.NumField())
	for i := 0; i < fields.NumField(); i++ {
		valueField := fields.Field(i)
		typeField := fields.Type().Field(i)
		buffer[i] = fmt.Sprintf("%s=%v", typeField.Name, valueField.Interface())
	}

	return fmt.Sprintf("<Config@%X %s>", &c, strings.Join(buffer, ", "))
}

//Validate check if the config is valid
func (c Config) Validate() error {
	if c.PIHoleProtocol != "http" && c.PIHoleProtocol != "https" {
		return fmt.Errorf("protocol %s is invalid. Must be http or https", c.PIHoleProtocol)
	}
	return nil
}

func (c EnvConfig) Split() []Config {
	result := make([]Config, 0, len(c.PIHoleHostname))

	for i, hostname := range c.PIHoleHostname {
		config := Config{
			PIHoleHostname: hostname,
			PIHoleProtocol: c.PIHoleProtocol[i],
			PIHolePort:     c.PIHolePort[i],
		}

		if c.PIHoleApiToken != nil {
			if len(c.PIHoleApiToken) == 1 {
				if c.PIHoleApiToken[0] != "" {
					config.PIHoleApiToken = c.PIHoleApiToken[0]
				}
			} else if len(c.PIHoleApiToken) > 1 {
				if c.PIHoleApiToken[i] != "" {
					config.PIHoleApiToken = c.PIHoleApiToken[i]
				}
			}
		}

		if c.PIHolePassword != nil {
			if len(c.PIHolePassword) == 1 {
				if c.PIHolePassword[0] != "" {
					config.PIHolePassword = c.PIHolePassword[0]
				}
			} else if len(c.PIHolePassword) > 1 {
				if c.PIHolePassword[i] != "" {
					config.PIHolePassword = c.PIHolePassword[i]
				}
			}
		}

		result = append(result, config)
	}
	return result
}

func (c Config) hostnameURL() string {
	s := fmt.Sprintf("%s://%s", c.PIHoleProtocol, c.PIHoleHostname)
	if c.PIHolePort != 0 {
		s += fmt.Sprintf(":%d", c.PIHolePort)
	}
	return s
}

//PIHoleStatsURL returns the stats url
func (c Config) PIHoleStatsURL() string {
	return c.hostnameURL() + "/admin/api.php?summaryRaw&overTimeData&topItems&recentItems&getQueryTypes&getForwardDestinations&getQuerySources&jsonForceObject"
}

//PIHoleLoginURL returns the login url
func (c Config) PIHoleLoginURL() string {
	return c.hostnameURL() + "/admin/index.php?login"
}

func (c EnvConfig) show() {
	val := reflect.ValueOf(&c).Elem()
	log.Println("------------------------------------")
	log.Println("-  PI-Hole exporter configuration  -")
	log.Println("------------------------------------")
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)

		// Do not print password or api token but do print the authentication method
		if typeField.Name != "PIHolePassword" && typeField.Name != "PIHoleApiToken" {
			log.Println(fmt.Sprintf("%s : %v", typeField.Name, valueField.Interface()))
		} else {
			showAuthenticationMethod(typeField.Name, valueField.Len())
		}
	}
	log.Println("------------------------------------")
}

func showAuthenticationMethod(name string, length int) {
	if length > 0 {
		log.Println(fmt.Sprintf("Pi-Hole Authentication Method : %s", name))
	}
}
