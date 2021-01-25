package checks

import (
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
	"github.com/stretchr/testify/assert"
)

func TestSanityCheck(t *testing.T) {
	if util.AnyEnvVarsExist(awsLogIngestionEnvVars) {
		assert.Error(t, sanityCheck(&config.Configuration{}, &api.RegistrationResponse{}, runtimeConfig{}))
	} else {
		assert.Nil(t, sanityCheck(&config.Configuration{}, &api.RegistrationResponse{}, runtimeConfig{}))
	}
}
