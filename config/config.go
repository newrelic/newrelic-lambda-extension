package config

type Configuration struct {
	UseCloudWatchIngest bool
	LicenseKey          *string
	LicenseKeySecretId  *string
	TelemetryEndpoint   *string
}

func ParseRegistration(conf map[string]string) Configuration {
	_, useCW := conf["NEW_RELIC_CLOUDWATCH_INGEST"]
	licenseKey, lkOverride := conf["NEW_RELIC_LICENSE_KEY"]
	licenseKeySecretId, lkSecretOverride := conf["NEW_RELIC_LICENSE_KEY_SECRET"]
	telemetryEndpoint, teOverride := conf["NEW_RELIC_TELEMETRY_ENDPOINT"]

	ret := Configuration{UseCloudWatchIngest: useCW}

	if lkOverride {
		ret.LicenseKey = &licenseKey
	} else if lkSecretOverride {
		ret.LicenseKeySecretId = &licenseKeySecretId
	}

	if teOverride {
		ret.TelemetryEndpoint = &telemetryEndpoint
	}

	return ret
}

var ConfigurationKeys = []string{
	"NEW_RELIC_CLOUDWATCH_INGEST",
	"NEW_RELIC_LICENSE_KEY",
	"NEW_RELIC_LICENSE_KEY_SECRET",
	"NEW_RELIC_TELEMETRY_ENDPOINT",
}
