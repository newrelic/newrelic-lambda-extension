package config

import (
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	DataCollectionTimeout   time.Duration
	LogLevel                log.Level
	AgentTelemetryBatchSize int
	TelemetryAPIBatchSize   int64
	AgentTelemetryRegion    string
	LicenseKey              string
	AccountID               string
	ExtensionName           string
	CollectAgentData        bool
}

const (
	defaultCollectionTimeout      = 10 * time.Second
	minimumCollectionTimeout      = 600 * time.Millisecond
	defaultAgentTelemtryBatchSize = 1
	defaultTelemtryAPIBatchSize   = 1

	// Optional Environment variables that can be used to talior the user experience to your needs
	agentDataEnabledVariable     = "NEW_RELIC_EXTENSION_AGENT_DATA_COLLECTION_ENABLED"
	agentDataBatchSizeVariable   = "NEW_RELIC_EXTENSION_AGENT_DATA_BATCH_SIZE"
	clientRetryTimeoutVariable   = "NEW_RELIC_EXTENSION_DATA_COLLECTION_TIMEOUT"
	agentTelemetryRegionVariable = "NEW_RELIC_EXTENSION_COLLECTOR_OVERRIDE"
	extensionLogLevelVariable    = "NEW_RELIC_EXTENSION_LOG_LEVEL"
	telAPIBatchSizeVariable      = "NEW_RELIC_EXTENSION_TELEMETRY_API_BATCH_SIZE"

	// Required environment variable
	nrAccountIDVariable = "NEW_RELIC_ACCOUNT_ID"
)

var l = log.WithFields(log.Fields{"pkg": "config"})

// simplifies testing
func defaultConfig() Config {
	return Config{
		CollectAgentData:        true,
		DataCollectionTimeout:   defaultCollectionTimeout,
		AgentTelemetryBatchSize: defaultAgentTelemtryBatchSize,
		TelemetryAPIBatchSize:   defaultTelemtryAPIBatchSize,
		LogLevel:                log.InfoLevel,
		ExtensionName:           path.Base(os.Args[0]),
	}
}

func GetConfig() Config {
	conf := defaultConfig()
	conf.AccountID = os.Getenv(nrAccountIDVariable)
	if conf.AccountID == "" {
		l.Errorf("environment variable \"%s\" must be set to the ID of the New Relic account matching your license key", nrAccountIDVariable)
	}

	conf.AgentTelemetryRegion = os.Getenv(agentTelemetryRegionVariable)

	// Enable or disable collection of agent telemetry data
	enableAgent := os.Getenv(agentDataEnabledVariable)
	if strings.ToLower(enableAgent) == "false" {
		conf.CollectAgentData = false
	}
	// How long agent will try to resend
	clientTimeout := os.Getenv(clientRetryTimeoutVariable)
	if clientTimeout != "" {
		dur, err := time.ParseDuration(clientTimeout)
		if err != nil {
			environmentVariableError(clientRetryTimeoutVariable, err)
			l.Warnf("client retry timeout will be set to default value: %s", defaultCollectionTimeout.String())
		}
		if dur >= minimumCollectionTimeout {
			conf.DataCollectionTimeout = dur
		} else {
			l.Warnf("configured client retry duration is too short, setting it to minimum value: %s", minimumCollectionTimeout.String())
			conf.DataCollectionTimeout = minimumCollectionTimeout
		}
	}

	telApiBatchSizeStr := os.Getenv(telAPIBatchSizeVariable)
	if telApiBatchSizeStr != "" {
		telApiBatchSize, err := strconv.ParseInt(telApiBatchSizeStr, 0, 16)
		if err != nil {
			environmentVariableError(telAPIBatchSizeVariable, err)
			l.Warnf("telemetry api batch size will be set to default value: %d", defaultTelemtryAPIBatchSize)
		} else {
			conf.TelemetryAPIBatchSize = telApiBatchSize
		}
	}

	buffer := os.Getenv(agentDataBatchSizeVariable)
	if buffer != "" {
		val, err := strconv.Atoi(buffer)
		if err != nil {
			environmentVariableError(agentDataBatchSizeVariable, err)
			l.Warnf("agent data batch size will be set to default value: %d", defaultAgentTelemtryBatchSize)
		} else {
			conf.AgentTelemetryBatchSize = val
		}
	}

	logLevel := strings.ToLower(os.Getenv(extensionLogLevelVariable))
	switch logLevel {
	case "trace":
		conf.LogLevel = log.TraceLevel
	case "debug":
		conf.LogLevel = log.DebugLevel
	case "info":
		conf.LogLevel = log.InfoLevel
	case "warn":
		conf.LogLevel = log.WarnLevel
	}

	return conf
}

func environmentVariableError(variable string, err error) {
	l.Warnf("[config] error parsing environment variable \"%s\": %v", variable, err)
}
