package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseRegistrationZero(t *testing.T) {
	conf := ParseRegistration(make(map[string]string))
	assert.Equal(t, Configuration{}, conf)
}

func TestParseRegistration(t *testing.T) {
	conf := ParseRegistration(map[string]string{
		"NEW_RELIC_CLOUDWATCH_INGEST":  "set",
		"NEW_RELIC_LICENSE_KEY":        "lk",
		"NEW_RELIC_LICENSE_KEY_SECRET": "secretId",
		"NEW_RELIC_TELEMETRY_ENDPOINT": "endpoint",
	})

	assert.Equal(t, true, conf.UseCloudWatchIngest)
	assert.Equal(t, "lk", *conf.LicenseKey)
	assert.Nil(t, conf.LicenseKeySecretId)
	assert.Equal(t, "endpoint", *conf.TelemetryEndpoint)
}

func TestParseRegistrationSecretId(t *testing.T) {
	conf := ParseRegistration(map[string]string{
		"NEW_RELIC_LICENSE_KEY_SECRET": "secretId",
	})
	assert.Equal(t, "secretId", *conf.LicenseKeySecretId)
}
