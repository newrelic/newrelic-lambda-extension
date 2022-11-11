package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigurationFromEnvironmentZero(t *testing.T) {
	conf := ConfigurationFromEnvironment()
	expected := &Configuration{
		ExtensionEnabled: true,
		RipeMillis:       DefaultRipeMillis,
		RotMillis:        DefaultRotMillis,
		LogLevel:         DefaultLogLevel,
		LogsEnabled:      true,
		NRHandler:        EmptyNRWrapper,
		LogServerHost:    defaultLogServerHost,
		ClientTimeout:    DefaultClientTimeout,
	}
	assert.Equal(t, expected, conf)
}

func TestConfigurationFromEnvironment(t *testing.T) {
	os.Unsetenv("NEW_RELIC_LAMBDA_EXTENSION_ENABLED")

	conf := ConfigurationFromEnvironment()

	assert.Equal(t, conf.ExtensionEnabled, true)
	assert.Equal(t, conf.LogsEnabled, true)

	os.Setenv("NEW_RELIC_LAMBDA_EXTENSION_ENABLED", "false")
	os.Setenv("NEW_RELIC_LAMBDA_HANDLER", "newrelic_lambda_wrapper.handler")
	os.Setenv("NEW_RELIC_LICENSE_KEY", "lk")
	os.Setenv("NEW_RELIC_LICENSE_KEY_SECRET", "secretId")
	os.Setenv("NEW_RELIC_LOG_ENDPOINT", "endpoint")
	os.Setenv("NEW_RELIC_TELEMETRY_ENDPOINT", "endpoint")
	os.Setenv("NEW_RELIC_HARVEST_RIPE_MILLIS", "0")
	os.Setenv("NEW_RELIC_HARVEST_ROT_MILLIS", "0")
	os.Setenv("NEW_RELIC_EXTENSION_LOG_LEVEL", "DEBUG")
	os.Setenv("NEW_RELIC_EXTENSION_SEND_FUNCTION_LOGS", "true")
	os.Setenv("NEW_RELIC_EXTENSION_LOGS_ENABLED", "false")
	os.Setenv("NEW_RELIC_DATA_COLLECTION_TIMEOUT", "5s")

	defer func() {
		os.Unsetenv("NEW_RELIC_LAMBDA_EXTENSION_ENABLED")
		os.Unsetenv("NEW_RELIC_LAMBDA_HANDLER")
		os.Unsetenv("NEW_RELIC_LICENSE_KEY")
		os.Unsetenv("NEW_RELIC_LICENSE_KEY_SECRET")
		os.Unsetenv("NEW_RELIC_LOG_ENDPOINT")
		os.Unsetenv("NEW_RELIC_TELEMETRY_ENDPOINT")
		os.Unsetenv("NEW_RELIC_HARVEST_RIPE_MILLIS")
		os.Unsetenv("NEW_RELIC_HARVEST_ROT_MILLIS")
		os.Unsetenv("NEW_RELIC_EXTENSION_LOG_LEVEL")
		os.Unsetenv("NEW_RELIC_EXTENSION_SEND_FUNCTION_LOGS")
		os.Unsetenv("NEW_RELIC_EXTENSION_LOGS_ENABLED")
		os.Unsetenv("NEW_RELIC_DATA_COLLECTION_TIMEOUT")
	}()

	conf = ConfigurationFromEnvironment()

	assert.Equal(t, conf.ExtensionEnabled, false)
	assert.Equal(t, "newrelic_lambda_wrapper.handler", conf.NRHandler)
	assert.Equal(t, "lk", conf.LicenseKey)
	assert.Empty(t, conf.LicenseKeySecretId)
	assert.Equal(t, "endpoint", conf.LogEndpoint)
	assert.Equal(t, "endpoint", conf.TelemetryEndpoint)
	assert.Equal(t, uint32(DefaultRipeMillis), conf.RipeMillis)
	assert.Equal(t, uint32(DefaultRotMillis), conf.RotMillis)
	assert.Equal(t, "DEBUG", conf.LogLevel)
	assert.Equal(t, true, conf.SendFunctionLogs)
	assert.Equal(t, false, conf.LogsEnabled)
}

func TestConfigurationFromEnvironmentSecretId(t *testing.T) {
	os.Setenv("NEW_RELIC_LICENSE_KEY_SECRET", "secretId")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY_SECRET")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, "secretId", conf.LicenseKeySecretId)
}

func TestConfigurationFromEnvironmentLogServerHost(t *testing.T) {
	os.Setenv("NEW_RELIC_LOG_SERVER_HOST", "foobar")
	defer os.Unsetenv("NEW_RELIC_LOG_SERVER_HOST")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, "foobar", conf.LogServerHost)
}
