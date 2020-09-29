package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestConfigurationFromEnvironmentZero(t *testing.T) {
	conf := ConfigurationFromEnvironment()
	assert.Equal(t, Configuration{}, conf)
}

func TestConfigurationFromEnvironment(t *testing.T) {
	os.Unsetenv("NEW_RELIC_LAMBDA_EXTENSION_ENABLED")

	conf := ConfigurationFromEnvironment()

	assert.Equal(t, conf.ExtensionEnabled, false)

	os.Setenv("NEW_RELIC_LAMBDA_EXTENSION_ENABLED", "set")
	os.Setenv("NEW_RELIC_LICENSE_KEY", "lk")
	os.Setenv("NEW_RELIC_LICENSE_KEY_SECRET", "secretId")
	os.Setenv("NEW_RELIC_TELEMETRY_ENDPOINT", "endpoint")

	defer func() {
		os.Unsetenv("NEW_RELIC_LAMBDA_EXTENSION_ENABLED")
		os.Unsetenv("NEW_RELIC_LICENSE_KEY")
		os.Unsetenv("NEW_RELIC_LICENSE_KEY_SECRET")
		os.Unsetenv("NEW_RELIC_TELEMETRY_ENDPOINT")
	}()

	conf = ConfigurationFromEnvironment()

	assert.Equal(t, conf.ExtensionEnabled, true)
	assert.Equal(t, "lk", *conf.LicenseKey)
	assert.Nil(t, conf.LicenseKeySecretId)
	assert.Equal(t, "endpoint", *conf.TelemetryEndpoint)
}

func TestConfigurationFromEnvironmentSecretId(t *testing.T) {
	os.Setenv("NEW_RELIC_LICENSE_KEY_SECRET", "secretId")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY_SECRET")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, "secretId", *conf.LicenseKeySecretId)
}
