package agentTelemetry

import (
	"os"
	"path"
	"strings"
	"time"
)

type Config struct {
	DataCollectionTimeout time.Duration
	AgentTelemetryRegion  string
	LicenseKey            string
	ExtensionName         string
}

const (
	defaultCollectionTimeout    = 10 * time.Second
	telemetryApiEnabledVariable = "NEW_RELIC_TELEMETRY_API_EXTENSION_ENABLED"
)

func GetConfig() Config {
	conf := Config{}

	// Check if extension is disabled
	// Default: true
	enabled := os.Getenv(telemetryApiEnabledVariable)
	if strings.ToLower(enabled) == "false" {
		l.Debug("[main] Lambda Extension is disabled")
		os.Exit(0) // exits if disabled in env
	}

	// How long agent will try to resend
	clientTimeout := os.Getenv("NEW_RELIC_DATA_COLLECTION_TIMEOUT")
	conf.DataCollectionTimeout = defaultCollectionTimeout

	dur, err := time.ParseDuration(clientTimeout)
	if err != nil {
		l.Debug("[main] data collection timeout failed to parse: %v", err)
	}
	if dur > time.Millisecond*500 {
		conf.DataCollectionTimeout = dur
	}

	conf.AgentTelemetryRegion = getAwsRegion()

	conf.ExtensionName = path.Base(os.Args[0])
	return conf
}

func getAwsRegion() string {
	return os.Getenv("NEW_RELIC_COLLECTOR_OVERRIDE")

}
