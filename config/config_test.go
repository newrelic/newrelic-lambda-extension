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

func TestLogLevelConfiguration(t *testing.T) {
    tests := []struct {
        envValue string
        expected string
    }{
        {"", DefaultLogLevel},
        {"debug", DebugLogLevel},
        {"DEBUG", DebugLogLevel},
        {"info", InfoLogLevel},
        {"INFO", InfoLogLevel},
        {"error", DefaultLogLevel},
        {"ERROR", DefaultLogLevel},
        {"warn", DefaultLogLevel},
        {"invalid", DefaultLogLevel},
    }

    for _, tt := range tests {
        t.Run("LogLevel_"+tt.envValue, func(t *testing.T) {
            clearEnvVars()
            defer clearEnvVars()

            if tt.envValue != "" {
                os.Setenv("NEW_RELIC_EXTENSION_LOG_LEVEL", tt.envValue)
            }

            config := ConfigurationFromEnvironment()
            assert.Equal(t, tt.expected, config.LogLevel)
        })
    }
}

func TestBooleanFlags(t *testing.T) {
    tests := []struct {
        envVar   string
        envValue string
        getter   func(*Configuration) bool
    }{
        {"NEW_RELIC_EXTENSION_SEND_FUNCTION_LOGS", "true", func(c *Configuration) bool { return c.SendFunctionLogs }},
        {"NEW_RELIC_EXTENSION_SEND_EXTENSION_LOGS", "true", func(c *Configuration) bool { return c.SendExtensionLogs }},
        {"NEW_RELIC_COLLECT_TRACE_ID", "true", func(c *Configuration) bool { return c.CollectTraceID }},
        {"NEW_RELIC_APM_LAMBDA_MODE", "true", func(c *Configuration) bool { return c.APMLambdaMode }},
    }

    for _, tt := range tests {
        t.Run(tt.envVar, func(t *testing.T) {
            clearEnvVars()
            defer clearEnvVars()

            os.Setenv(tt.envVar, tt.envValue)
            config := ConfigurationFromEnvironment()
            assert.True(t, tt.getter(config))
        })
    }
}

func TestParseIgnoredExtensionChecks(t *testing.T) {
    tests := []struct {
        name      string
        override  bool
        checksStr string
        expected  map[string]bool
    }{
        {"No override", false, "", nil},
        {"Empty string", true, "", nil},
        {"All checks", true, "all", map[string]bool{"all": true}},
        {"Valid checks", true, "agent,handler", map[string]bool{"agent": true, "handler": true}},
        {"Invalid checks", true, "invalid,agent", map[string]bool{"agent": true}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := parseIgnoredExtensionChecks(tt.override, tt.checksStr)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func clearEnvVars() {
    envVars := []string{
        "NEW_RELIC_ENABLED",
        "NEW_RELIC_AGENT_ENABLED",
        "NEW_RELIC_IGNORE_EXTENSION_CHECKS",
        "NEW_RELIC_LAMBDA_EXTENSION_ENABLED",
        "NEW_RELIC_LICENSE_KEY",
        "NEW_RELIC_LICENSE_KEY_SECRET",
        "NEW_RELIC_LICENSE_KEY_SSM_PARAMETER_NAME",
        "NEW_RELIC_LAMBDA_HANDLER",
        "NEW_RELIC_TELEMETRY_ENDPOINT",
        "NEW_RELIC_LOG_ENDPOINT",
        "NEW_RELIC_METRIC_ENDPOINT",
        "NEW_RELIC_DATA_COLLECTION_TIMEOUT",
        "NEW_RELIC_HARVEST_RIPE_MILLIS",
        "NEW_RELIC_HARVEST_ROT_MILLIS",
        "NEW_RELIC_EXTENSION_LOG_LEVEL",
        "NEW_RELIC_EXTENSION_LOGS_ENABLED",
        "NEW_RELIC_EXTENSION_SEND_FUNCTION_LOGS",
        "NEW_RELIC_EXTENSION_SEND_EXTENSION_LOGS",
        "NEW_RELIC_LOG_SERVER_HOST",
        "NEW_RELIC_COLLECT_TRACE_ID",
        "NEW_RELIC_HOST",
        "NEW_RELIC_APM_LAMBDA_MODE",
    }

    for _, envVar := range envVars {
        os.Unsetenv(envVar)
    }
}
