package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	DefaultRipeMillis = 7_000
	DefaultRotMillis  = 12_000
)

type Configuration struct {
	ExtensionEnabled   bool
	LicenseKey         *string
	LicenseKeySecretId *string
	TelemetryEndpoint  *string
	RipeMillis         uint32
	RotMillis          uint32
}

func ConfigurationFromEnvironment() Configuration {
	enabledStr, extensionEnabledOverride := os.LookupEnv("NEW_RELIC_LAMBDA_EXTENSION_ENABLED")
	licenseKey, lkOverride := os.LookupEnv("NEW_RELIC_LICENSE_KEY")
	licenseKeySecretId, lkSecretOverride := os.LookupEnv("NEW_RELIC_LICENSE_KEY_SECRET")
	telemetryEndpoint, teOverride := os.LookupEnv("NEW_RELIC_TELEMETRY_ENDPOINT")
	ripeMillisStr, ripeMillisOverride := os.LookupEnv("NEW_RELIC_HARVEST_RIPE_MILLIS")
	rotMillisStr, rotMillisOverride := os.LookupEnv("NEW_RELIC_HARVEST_ROT_MILLIS")

	extensionEnabled := true
	if extensionEnabledOverride && "false" == strings.ToLower(enabledStr) {
		extensionEnabled = false
	}
	ret := Configuration{ExtensionEnabled: extensionEnabled}

	if lkOverride {
		ret.LicenseKey = &licenseKey
	} else if lkSecretOverride {
		ret.LicenseKeySecretId = &licenseKeySecretId
	}

	if teOverride {
		ret.TelemetryEndpoint = &telemetryEndpoint
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

	return ret
}
