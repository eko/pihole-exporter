package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitDefault(t *testing.T) {
	assert := assert.New(t)

	env := getDefaultEnvConfig()

	clientConfigs, err := env.Split()
	assert.NoError(err)

	clientConfig := clientConfigs[0]
	assert.Equal("127.0.0.1", clientConfig.PIHoleHostname)
	assert.Equal("http", clientConfig.PIHoleProtocol)
	assert.Equal(uint16(80), clientConfig.PIHolePort)
	assert.Empty(clientConfig.PIHolePassword)
}

func TestSplitMultipleHostWithSameConfig(t *testing.T) {
	assert := assert.New(t)

	env := getDefaultEnvConfig()
	env.PIHoleHostname = []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}
	env.PIHolePort = []uint16{8080}

	clientConfigs, err := env.Split()
	assert.NoError(err)
	assert.Len(clientConfigs, 3)

	testCases := []struct {
		Host         string
		Port         uint16
		AdminContext string
		Protocol     string
	}{
		{
			Host:         "127.0.0.1",
			Port:         8080,
			AdminContext: "foo",
			Protocol:     "http",
		},
		{
			Host:         "127.0.0.2",
			Port:         8080,
			AdminContext: "foo",
			Protocol:     "http",
		},
		{
			Host:         "127.0.0.3",
			Port:         8080,
			AdminContext: "foo",
			Protocol:     "http",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Test %s", tc.Host), func(t *testing.T) {
			clientConfig := clientConfigs[i]

			assert.Equal(tc.Host, clientConfig.PIHoleHostname)
			assert.Equal(tc.Protocol, clientConfig.PIHoleProtocol)
			assert.Equal(tc.Port, clientConfig.PIHolePort)
			assert.Empty(clientConfig.PIHolePassword)
		})
	}
}

func TestSplitMultipleHostWithMultipleConfigs(t *testing.T) {
	assert := assert.New(t)

	env := getDefaultEnvConfig()
	env.PIHoleHostname = []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}
	env.PIHolePassword = []string{"", "password2", ""}
	env.PIHolePort = []uint16{8081, 8082, 8083}

	clientConfigs, err := env.Split()
	assert.NoError(err)
	assert.Len(clientConfigs, 3)

	testCases := []struct {
		Host         string
		Port         uint16
		AdminContext string
		Protocol     string
		Password     string
	}{
		{
			Host:         "127.0.0.1",
			Port:         8081,
			AdminContext: "",
			Protocol:     "http",
			Password:     "",
		},
		{
			Host:         "127.0.0.2",
			Port:         8082,
			AdminContext: "foo",
			Protocol:     "http",
			Password:     "password2",
		},
		{
			Host:         "127.0.0.3",
			Port:         8083,
			AdminContext: "bar",
			Protocol:     "http",
			Password:     "",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Test %s", tc.Host), func(t *testing.T) {
			clientConfig := clientConfigs[i]

			assert.Equal(tc.Host, clientConfig.PIHoleHostname)
			assert.Equal(tc.Protocol, clientConfig.PIHoleProtocol)
			assert.Equal(tc.Port, clientConfig.PIHolePort)
			assert.Equal(tc.Password, clientConfig.PIHolePassword)
		})
	}
}
