package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	PIHOLE_HOSTNAME      = "PIHOLE_HOSTNAME"
	PIHOLE_PORT          = "PIHOLE_PORT"
	PIHOLE_ADMIN_CONTEXT = "PIHOLE_ADMIN_CONTEXT"
	PIHOLE_API_TOKEN     = "PIHOLE_API_TOKEN"
	PIHOLE_PROTOCOL      = "PIHOLE_PROTOCOL"
)

type EnvInitiazlier func(*testing.T)

type TestCase struct {
	Name        string
	Initializer EnvInitiazlier
}

func TestSplitDefault(t *testing.T) {
	assert := assert.New(t)

	env := getDefaultEnvConfig()

	clientConfigs, err := env.Split()
	assert.NoError(err)

	clientConfig := clientConfigs[0]
	assert.Equal("127.0.0.1", clientConfig.PIHoleHostname)
	assert.Equal("http", clientConfig.PIHoleProtocol)
	assert.Equal("admin", clientConfig.PIHoleAdminContext)
	assert.Equal(uint16(80), clientConfig.PIHolePort)
	assert.Empty(clientConfig.PIHoleApiToken)
	assert.Empty(clientConfig.PIHolePassword)
}

func TestSplitMultipleHostWithSameConfig(t *testing.T) {
	assert := assert.New(t)

	env := getDefaultEnvConfig()
	env.PIHoleHostname = []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}
	env.PIHoleApiToken = []string{"api-token"}
	env.PIHoleAdminContext = []string{"foo"}
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
			assert.Equal("api-token", clientConfig.PIHoleApiToken)
			assert.Empty(clientConfig.PIHolePassword)
		})
	}
}

func TestSplitMultipleHostWithMultipleConfigs(t *testing.T) {
	assert := assert.New(t)

	env := getDefaultEnvConfig()
	env.PIHoleHostname = []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}
	env.PIHoleApiToken = []string{"api-token1", "", "api-token3"}
	env.PIHoleAdminContext = []string{"", "foo", "bar"}
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
		ApiToken     string
		Password     string
	}{
		{
			Host:         "127.0.0.1",
			Port:         8081,
			AdminContext: "",
			Protocol:     "http",
			ApiToken:     "api-token1",
			Password:     "",
		},
		{
			Host:         "127.0.0.2",
			Port:         8082,
			AdminContext: "foo",
			Protocol:     "http",
			ApiToken:     "",
			Password:     "password2",
		},
		{
			Host:         "127.0.0.3",
			Port:         8083,
			AdminContext: "bar",
			Protocol:     "http",
			ApiToken:     "api-token3",
			Password:     "",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Test %s", tc.Host), func(t *testing.T) {
			clientConfig := clientConfigs[i]

			assert.Equal(tc.Host, clientConfig.PIHoleHostname)
			assert.Equal(tc.Protocol, clientConfig.PIHoleProtocol)
			assert.Equal(tc.AdminContext, clientConfig.PIHoleAdminContext)
			assert.Equal(tc.Port, clientConfig.PIHolePort)
			assert.Equal(tc.ApiToken, clientConfig.PIHoleApiToken)
			assert.Equal(tc.Password, clientConfig.PIHolePassword)
		})
	}
}

func TestWrongNumberOfApiTokenParams(t *testing.T) {
	assert := assert.New(t)

	env := getDefaultEnvConfig()
	env.PIHoleHostname = []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}
	env.PIHoleApiToken = []string{"api-token1", "api-token2"}
	env.PIHolePort = []uint16{808}

	clientConfigs, err := env.Split()
	assert.Errorf(err, "Wrong number of PIHoleApiToken. PIHoleApiToken can be empty to use default, one value to use for all hosts, or match the number of hosts")
	assert.Nil(clientConfigs)
}

func TestWrongNumberOfAdminContextParams(t *testing.T) {
	assert := assert.New(t)

	env := getDefaultEnvConfig()
	env.PIHoleHostname = []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}
	env.PIHoleApiToken = []string{"api-token"}
	env.PIHoleAdminContext = []string{"admin1", "admin2"}
	env.PIHolePort = []uint16{808}

	clientConfigs, err := env.Split()
	assert.Errorf(err, "Wrong number of PIHoleAdminContext. PIHoleAdminContext can be empty to use default, one value to use for all hosts, or match the number of hosts")
	assert.Nil(clientConfigs)
}
