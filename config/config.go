package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultRipeMillis    = 7_000
	DefaultRotMillis     = 12_000
	DefaultLogLevel      = "INFO"
	DebugLogLevel        = "DEBUG"
	defaultLogServerHost = "sandbox.localdomain"
	DefaultClientTimeout = 10 * time.Second
)

var EmptyNRWrapper = "Undefined"

type Configuration struct {
	ExtensionEnabled    bool
	LogsEnabled         bool
	SendFunctionLogs    bool
	CollectTraceID      bool
	TelemetryAPIEnabled bool
	RipeMillis          uint32
	RotMillis           uint32
	LicenseKey          string
	LicenseKeySecretId  string
	NRHandler           string
	TelemetryEndpoint   string
	LogEndpoint         string
	LogLevel            string
	LogServerHost       string
	ClientTimeout       time.Duration
}

func ConfigurationFromEnvironment() *Configuration {
	enabledStr, extensionEnabledOverride := os.LookupEnv("NEW_RELIC_LAMBDA_EXTENSION_ENABLED")
	licenseKey, lkOverride := os.LookupEnv("NEW_RELIC_LICENSE_KEY")
	licenseKeySecretId, lkSecretOverride := os.LookupEnv("NEW_RELIC_LICENSE_KEY_SECRET")
	nrHandler, nrOverride := os.LookupEnv("NEW_RELIC_LAMBDA_HANDLER")
	telemetryEndpoint, teOverride := os.LookupEnv("NEW_RELIC_TELEMETRY_ENDPOINT")
	logEndpoint, leOverride := os.LookupEnv("NEW_RELIC_LOG_ENDPOINT")
	clientTimeout, ctOverride := os.LookupEnv("NEW_RELIC_DATA_COLLECTION_TIMEOUT")
	ripeMillisStr, ripeMillisOverride := os.LookupEnv("NEW_RELIC_HARVEST_RIPE_MILLIS")
	rotMillisStr, rotMillisOverride := os.LookupEnv("NEW_RELIC_HARVEST_ROT_MILLIS")
	logLevelStr, logLevelOverride := os.LookupEnv("NEW_RELIC_EXTENSION_LOG_LEVEL")
	logsEnabledStr, logsEnabledOverride := os.LookupEnv("NEW_RELIC_EXTENSION_LOGS_ENABLED")
	sendFunctionLogsStr, sendFunctionLogsOverride := os.LookupEnv("NEW_RELIC_EXTENSION_SEND_FUNCTION_LOGS")
	logServerHostStr, logServerHostOverride := os.LookupEnv("NEW_RELIC_LOG_SERVER_HOST")
	collectTraceIDStr, collectTraceIDOverride := os.LookupEnv("NEW_RELIC_COLLECT_TRACE_ID")
	telemetryAPIEnabledStr, telemetryAPIOverride := os.LookupEnv("NEW_RELIC_TELEMETRY_API_EXTENSION_ENABLED")

	extensionEnabled := true
	if extensionEnabledOverride && strings.ToLower(enabledStr) == "false" {
		extensionEnabled = false
	}

	logsEnabled := true
	if logsEnabledOverride && strings.ToLower(logsEnabledStr) == "false" {
		logsEnabled = false
	}

	ret := &Configuration{ExtensionEnabled: extensionEnabled, LogsEnabled: logsEnabled}

	ret.ClientTimeout = DefaultClientTimeout
	if ctOverride && clientTimeout != "" {
		clientTimeout, err := time.ParseDuration(clientTimeout)
		if err == nil {
			ret.ClientTimeout = clientTimeout
		}
	}

	if lkOverride {
		ret.LicenseKey = licenseKey
	} else if lkSecretOverride {
		ret.LicenseKeySecretId = licenseKeySecretId
	}

	if nrOverride {
		ret.NRHandler = nrHandler
	} else {
		ret.NRHandler = EmptyNRWrapper
	}

	if teOverride {
		ret.TelemetryEndpoint = telemetryEndpoint
	}

	if leOverride {
		ret.LogEndpoint = logEndpoint
	}

	if ripeMillisOverride {
		ripeMillis, err := strconv.ParseUint(ripeMillisStr, 10, 32)
		if err == nil {
			ret.RipeMillis = uint32(ripeMillis)
		}
	}

	if ret.RipeMillis == 0 {
		ret.RipeMillis = DefaultRipeMillis
	}

	if rotMillisOverride {
		rotMillis, err := strconv.ParseUint(rotMillisStr, 10, 32)
		if err == nil {
			ret.RotMillis = uint32(rotMillis)
		}
	}

	if ret.RotMillis == 0 {
		ret.RotMillis = DefaultRotMillis
	}

	if logLevelOverride && logLevelStr == DebugLogLevel {
		ret.LogLevel = DebugLogLevel
	} else {
		ret.LogLevel = DefaultLogLevel
	}

	if logServerHostOverride {
		ret.LogServerHost = logServerHostStr
	} else {
		ret.LogServerHost = defaultLogServerHost
	}

	if telemetryAPIOverride && strings.ToLower(telemetryAPIEnabledStr) == "true" {
		ret.TelemetryAPIEnabled = true
	}

	// if telemetry API is enabled, disable logAPI to avoid duplicating log data
	// telemetry API replaces and improves on logs api
	// https://aws.amazon.com/blogs/compute/introducing-the-aws-lambda-telemetry-api/
	if !telemetryAPIOverride && sendFunctionLogsOverride && strings.ToLower(sendFunctionLogsStr) == "true" {
		ret.SendFunctionLogs = true
	}

	if collectTraceIDOverride && strings.ToLower(collectTraceIDStr) == "true" {
		ret.CollectTraceID = true
	}

	return ret
}
