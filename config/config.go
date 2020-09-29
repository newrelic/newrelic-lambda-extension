package config

import "os"

type Configuration struct {
	ExtensionEnabled   bool
	LicenseKey         *string
	LicenseKeySecretId *string
	TelemetryEndpoint  *string
}

func ConfigurationFromEnvironment() Configuration {
	_, extensionEnabled := os.LookupEnv("NEW_RELIC_LAMBDA_EXTENSION_ENABLED")
	licenseKey, lkOverride := os.LookupEnv("NEW_RELIC_LICENSE_KEY")
	licenseKeySecretId, lkSecretOverride := os.LookupEnv("NEW_RELIC_LICENSE_KEY_SECRET")
	telemetryEndpoint, teOverride := os.LookupEnv("NEW_RELIC_TELEMETRY_ENDPOINT")

	ret := Configuration{ExtensionEnabled: extensionEnabled}

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
