package agentTelemetry

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
	defaultAgentTelemtryBatchSize = 1
	defaultTelemtryAPIBatchSize   = 1

	// Optional Environment variables that can be used to talior the user experience to your needs
	telemetryApiEnabledVariable  = "NEW_RELIC_TELEMETRY_API_EXTENSION_ENABLED"
	agentDataEnabledVariable     = "NEW_RELIC_EXTENSION_AGENT_DATA_COLLECTION_ENABLED"
	agentDataBatchSizeVariable   = "NEW_RELIC_EXTENSION_AGENT_DATA_BATCH_SIZE"
	clientRetryTimeoutVariable   = "NEW_RELIC_EXTENSION_DATA_COLLECTION_TIMEOUT"
	agentTelemetryRegionVariable = "NEW_RELIC_EXTENSION_COLLECTOR_OVERRIDE"
	extensionLogLevelVariable    = "NEW_RELIC_EXTENSION_LOG_LEVEL"
	telAPIBatchSizeVariable      = "NEW_RELIC_EXTENSION_TELEMETRY_API_BATCH_SIZE"
	NrAccountIDVariable          = "NEW_RELIC_ACCOUNT_ID"
)

func GetConfig() Config {
	// Check if extension is disabled
	// Default: true
	enabled := os.Getenv(telemetryApiEnabledVariable)
	if strings.ToLower(enabled) == "false" {
		l.Warnf("[config] Lambda Extension is disabled")
		os.Exit(0) // exits if disabled in env
	}

	// Set Defaults
	conf := Config{
		CollectAgentData:        true,
		DataCollectionTimeout:   defaultCollectionTimeout,
		AgentTelemetryBatchSize: defaultAgentTelemtryBatchSize,
		TelemetryAPIBatchSize:   defaultTelemtryAPIBatchSize,
		LogLevel:                log.InfoLevel,
		ExtensionName:           path.Base(os.Args[0]),
	}

	conf.AccountID = os.Getenv("NEW_RELIC_ACCOUNT_ID")
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
		}
		if dur > time.Millisecond*400 {
			conf.DataCollectionTimeout = dur
		}
	}

	telApiBatchSize, err := strconv.ParseInt(os.Getenv(telAPIBatchSizeVariable), 0, 16)
	if err != nil {
		environmentVariableError(telAPIBatchSizeVariable, err)
	} else {
		conf.TelemetryAPIBatchSize = telApiBatchSize
	}

	buffer := os.Getenv(agentDataBatchSizeVariable)
	if buffer != "" {
		val, err := strconv.Atoi(buffer)
		if err != nil {
			environmentVariableError(agentDataBatchSizeVariable, err)
		} else {
			conf.AgentTelemetryBatchSize = val
		}
	}

	logLevel := os.Getenv(extensionLogLevelVariable)
	if strings.ToLower(logLevel) == "debug" {
		conf.LogLevel = log.DebugLevel
	}

	return conf
}

func environmentVariableError(variable string, err error) {
	l.Warnf("[config] error parsing environment variable \"%s\": %v", variable, err)
}
