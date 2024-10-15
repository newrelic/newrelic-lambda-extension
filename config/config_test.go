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
	os.Setenv("NEW_RELIC_LICENSE_KEY_SSM_PARAMETER_NAME", "parameterName")
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
	assert.Empty(t, conf.LicenseKeySSMParameterName)
	assert.Equal(t, "endpoint", conf.LogEndpoint)
	assert.Equal(t, "endpoint", conf.TelemetryEndpoint)
	assert.Equal(t, uint32(DefaultRipeMillis), conf.RipeMillis)
	assert.Equal(t, uint32(DefaultRotMillis), conf.RotMillis)
	assert.Equal(t, "DEBUG", conf.LogLevel)
	assert.Equal(t, true, conf.SendFunctionLogs)
	assert.Equal(t, false, conf.LogsEnabled)
}

func TestConfigurationFromEnvironmentNREnabled(t *testing.T) {
	os.Setenv("NEW_RELIC_ENABLED", "false")
	defer os.Unsetenv("NEW_RELIC_ENABLED")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, conf.ExtensionEnabled, false)
}

func TestConfigurationFromEnvironmentNREnabledBool(t *testing.T) {
	os.Setenv("NEW_RELIC_ENABLED", "0")
	defer os.Unsetenv("NEW_RELIC_ENABLED")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, conf.ExtensionEnabled, false)
}

func TestConfigurationFromEnvironmentNRAgentEnabled(t *testing.T) {
	os.Setenv("NEW_RELIC_AGENT_ENABLED", "false")
	defer os.Unsetenv("NEW_RELIC_AGENT_ENABLED")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, conf.ExtensionEnabled, false)
}

func TestConfigurationFromEnvironmentExtensionChecks(t *testing.T) {
	os.Setenv("NEW_RELIC_IGNORE_EXTENSION_CHECKS", "agent,handler,dummy")
	defer os.Unsetenv("NEW_RELIC_IGNORE_EXTENSION_CHECKS")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, conf.IgnoreExtensionChecks["agent"], true)
	assert.Equal(t, conf.IgnoreExtensionChecks["handler"], true)
	assert.Equal(t, len(conf.IgnoreExtensionChecks), 2)
}

func TestConfigurationFromEnvironmentExtensionChecksAll(t *testing.T) {
	os.Setenv("NEW_RELIC_IGNORE_EXTENSION_CHECKS", "ALL")
	defer os.Unsetenv("NEW_RELIC_IGNORE_EXTENSION_CHECKS")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, conf.IgnoreExtensionChecks["agent"], false)
	assert.Equal(t, conf.IgnoreExtensionChecks["handler"], false)
	assert.Equal(t, conf.IgnoreExtensionChecks["sanity"], false)
	assert.Equal(t, conf.IgnoreExtensionChecks["vendor"], false)
	assert.Equal(t, len(conf.IgnoreExtensionChecks), 1)
}

func TestConfigurationFromEnvironmentExtensionChecksIncorrectString(t *testing.T) {
	os.Setenv("NEW_RELIC_IGNORE_EXTENSION_CHECKS", "incorrect,valuess,...,,")
	defer os.Unsetenv("NEW_RELIC_IGNORE_EXTENSION_CHECKS")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, len(conf.IgnoreExtensionChecks), 0)
}

func TestConfigurationFromEnvironmentExtensionChecksIncorrectStringWithAll(t *testing.T) {
	os.Setenv("NEW_RELIC_IGNORE_EXTENSION_CHECKS", "incorrect,valuess,...,ALL,")
	defer os.Unsetenv("NEW_RELIC_IGNORE_EXTENSION_CHECKS")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, len(conf.IgnoreExtensionChecks), 0)
}

func TestConfigurationFromEnvironmentExtensionChecksIncorrectStringAll(t *testing.T) {
	os.Setenv("NEW_RELIC_IGNORE_EXTENSION_CHECKS", "All,ALL,...,ALL,")
	defer os.Unsetenv("NEW_RELIC_IGNORE_EXTENSION_CHECKS")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, len(conf.IgnoreExtensionChecks), 0)
}

func TestConfigurationFromEnvironmentExtensionChecksEmptyString(t *testing.T) {
	os.Setenv("NEW_RELIC_IGNORE_EXTENSION_CHECKS", "")
	defer os.Unsetenv("NEW_RELIC_IGNORE_EXTENSION_CHECKS")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, len(conf.IgnoreExtensionChecks), 0)
}

func TestConfigurationFromEnvironmentSecretId(t *testing.T) {
	os.Setenv("NEW_RELIC_LICENSE_KEY_SECRET", "secretId")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY_SECRET")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, "secretId", conf.LicenseKeySecretId)
}

func TestConfigurationFromEnvironmentParameterName(t *testing.T) {
	os.Setenv("NEW_RELIC_LICENSE_KEY_SSM_PARAMETER_NAME", "parameterName")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY_SSM_PARAMETER_NAME")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, "parameterName", conf.LicenseKeySSMParameterName)
}

func TestConfigurationFromEnvironmentLogServerHost(t *testing.T) {
	os.Setenv("NEW_RELIC_LOG_SERVER_HOST", "foobar")
	defer os.Unsetenv("NEW_RELIC_LOG_SERVER_HOST")

	conf := ConfigurationFromEnvironment()
	assert.Equal(t, "foobar", conf.LogServerHost)
}
