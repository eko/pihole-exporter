package config

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend"
	"github.com/heetch/confita/backend/env"
	"github.com/heetch/confita/backend/flags"
)

// Config is the exporter CLI configuration.
type Config struct {
	PIHoleProtocol string        `config:"pihole_protocol"`
	PIHoleHostname string        `config:"pihole_hostname"`
	PIHolePort     uint16        `config:"pihole_port"`
	PIHolePassword string        `config:"pihole_password"`
	PIHoleApiToken string        `config:"pihole_api_token"`
	Port           string        `config:"port"`
	Interval       time.Duration `config:"interval"`
}

func getDefaultConfig() *Config {
	return &Config{
		PIHoleProtocol: "http",
		PIHoleHostname: "127.0.0.1",
		PIHolePort:     80,
		PIHolePassword: "",
		PIHoleApiToken: "",
		Port:           "9617",
		Interval:       10 * time.Second,
	}
}

// Load method loads the configuration by using both flag or environment variables.
func Load() *Config {
	loaders := []backend.Backend{
		env.NewBackend(),
		flags.NewBackend(),
	}

	loader := confita.NewLoader(loaders...)

	cfg := getDefaultConfig()
	err := loader.Load(context.Background(), cfg)
	if err != nil {
		panic(err)
	}

	cfg.show()

	return cfg
}

func (c Config) show() {
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
			showAuthenticationMethod(typeField.Name, valueField.String())
		}
	}
	log.Println("------------------------------------")
}

func showAuthenticationMethod(name, value string) {
	if len(value) > 0 {
		log.Println(fmt.Sprintf("Pi-Hole Authentication Method : %s", name))
	}
}
