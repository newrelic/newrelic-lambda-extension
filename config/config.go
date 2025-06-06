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
	TestingOverride            bool // ignores envioronment specific details when running unit tests
	ExtensionEnabled           bool
	IgnoreExtensionChecks      map[string]bool
	LogsEnabled                bool
	SendFunctionLogs           bool
	SendExtensionLogs		   bool
	CollectTraceID             bool
	RipeMillis                 uint32
	RotMillis                  uint32
	LicenseKey                 string
	LicenseKeySecretId         string
	LicenseKeySSMParameterName string
	NRHandler                  string
	TelemetryEndpoint          string
	MetricEndpoint 		       string
	LogEndpoint                string
	LogLevel                   string
	LogServerHost              string
	ClientTimeout              time.Duration
	NewRelicHost               string
	APMLambdaMode              bool
	PreconnectEnabled		   bool
}

func parseIgnoredExtensionChecks(nrIgnoreExtensionChecksOverride bool, nrIgnoreExtensionChecksStr string) map[string]bool {
	ignoredChecks := make(map[string]bool)

	if !nrIgnoreExtensionChecksOverride || nrIgnoreExtensionChecksStr == "" {
		return nil
	}

	validChecks := map[string]bool{
		"agent":   true,
		"handler": true,
		"sanity":  true,
		"vendor":  true,
	}

	ignoredChecksStr := strings.ToLower(nrIgnoreExtensionChecksStr)
	
	if ignoredChecksStr == "all" {
		ignoredChecks["all"] = true
		return ignoredChecks
	}

	checks := strings.Split(ignoredChecksStr, ",")
	for _, check := range checks {
		trimmedCheck := strings.TrimSpace(check)
		if trimmedCheck != "" && validChecks[trimmedCheck] {
			ignoredChecks[trimmedCheck] = true
		}
	}

	return ignoredChecks
}

func ConfigurationFromEnvironment() *Configuration {
	nrEnabledStr, nrEnabledOverride := os.LookupEnv("NEW_RELIC_ENABLED")
	nrEnabledRubyStr, nrEnabledRubyOverride := os.LookupEnv("NEW_RELIC_AGENT_ENABLED")
	nrIgnoreExtensionChecksStr, nrIgnoreExtensionChecksOverride := os.LookupEnv("NEW_RELIC_IGNORE_EXTENSION_CHECKS")
	enabledStr, extensionEnabledOverride := os.LookupEnv("NEW_RELIC_LAMBDA_EXTENSION_ENABLED")
	licenseKey, lkOverride := os.LookupEnv("NEW_RELIC_LICENSE_KEY")
	licenseKeySecretId, lkSecretOverride := os.LookupEnv("NEW_RELIC_LICENSE_KEY_SECRET")
	licenseKeySSMParameterName, lkSSMParameterOverride := os.LookupEnv("NEW_RELIC_LICENSE_KEY_SSM_PARAMETER_NAME")
	nrHandler, nrOverride := os.LookupEnv("NEW_RELIC_LAMBDA_HANDLER")
	telemetryEndpoint, teOverride := os.LookupEnv("NEW_RELIC_TELEMETRY_ENDPOINT")
	logEndpoint, leOverride := os.LookupEnv("NEW_RELIC_LOG_ENDPOINT")
	clientTimeout, ctOverride := os.LookupEnv("NEW_RELIC_DATA_COLLECTION_TIMEOUT")
	ripeMillisStr, ripeMillisOverride := os.LookupEnv("NEW_RELIC_HARVEST_RIPE_MILLIS")
	rotMillisStr, rotMillisOverride := os.LookupEnv("NEW_RELIC_HARVEST_ROT_MILLIS")
	logLevelStr, logLevelOverride := os.LookupEnv("NEW_RELIC_EXTENSION_LOG_LEVEL")
	logsEnabledStr, logsEnabledOverride := os.LookupEnv("NEW_RELIC_EXTENSION_LOGS_ENABLED")
	sendFunctionLogsStr, sendFunctionLogsOverride := os.LookupEnv("NEW_RELIC_EXTENSION_SEND_FUNCTION_LOGS")
	sendExtensionLogsStr, sendExtensionLogsOverride := os.LookupEnv("NEW_RELIC_EXTENSION_SEND_EXTENSION_LOGS")
	logServerHostStr, logServerHostOverride := os.LookupEnv("NEW_RELIC_LOG_SERVER_HOST")
	collectTraceIDStr, collectTraceIDOverride := os.LookupEnv("NEW_RELIC_COLLECT_TRACE_ID")
	nrHostStr, nrHostOverride := os.LookupEnv("NEW_RELIC_HOST")
	nrAPMModeStr, nrAPMModeOverride := os.LookupEnv("NEW_RELIC_APM_LAMBDA_MODE")
	metricEndpoint, meOverride := os.LookupEnv("NEW_RELIC_METRIC_ENDPOINT")


	extensionEnabled := true
	if nrEnabledOverride {
		b, err := strconv.ParseBool(nrEnabledStr)
		if err == nil && !b {
			extensionEnabled = false
		}
	}
	if nrEnabledRubyOverride && strings.ToLower(nrEnabledRubyStr) == "false" {
		extensionEnabled = false
	}
	if extensionEnabledOverride && strings.ToLower(enabledStr) == "false" {
		extensionEnabled = false
	}

	logsEnabled := true
	if logsEnabledOverride && strings.ToLower(logsEnabledStr) == "false" {
		logsEnabled = false
	}

	ret := &Configuration{ExtensionEnabled: extensionEnabled, LogsEnabled: logsEnabled}
	if nrAPMModeOverride && strings.ToLower(nrAPMModeStr) == "true" {
		ret.APMLambdaMode = true
		ret.PreconnectEnabled = true
	}
	if nrHostOverride {
		ret.NewRelicHost = nrHostStr
	}
	
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
	} else if lkSSMParameterOverride {
		ret.LicenseKeySSMParameterName = licenseKeySSMParameterName
	}

	ret.IgnoreExtensionChecks = parseIgnoredExtensionChecks(nrIgnoreExtensionChecksOverride, nrIgnoreExtensionChecksStr)

	if nrOverride {
		ret.NRHandler = nrHandler
	} else {
		ret.NRHandler = EmptyNRWrapper
	}

	if teOverride {
		ret.TelemetryEndpoint = telemetryEndpoint
	}

	if meOverride {
		ret.MetricEndpoint = metricEndpoint
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

	if sendFunctionLogsOverride && sendFunctionLogsStr == "true" {
		ret.SendFunctionLogs = true
	}

	if sendExtensionLogsOverride && strings.ToLower(sendExtensionLogsStr) == "true" {
		ret.SendExtensionLogs = true
	}

	if collectTraceIDOverride && collectTraceIDStr == "true" {
		ret.CollectTraceID = true
	}

	return ret
}
