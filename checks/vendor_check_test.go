package checks

import (
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
	"github.com/stretchr/testify/assert"
)

func TestVendorCheck(t *testing.T) {
	if !util.PathExists(layerAgentPathNode) && !util.AnyPathsExist(layerAgentPathsPython) && !util.PathExists(vendorAgentPathNode) && !util.PathExists(vendorAgentPathPython) {
		assert.Nil(t, vendorCheck(&config.Configuration{}, &api.RegistrationResponse{}))
	}

	if util.PathExists(layerAgentPathNode) && util.PathExists(vendorAgentPathNode) {
		assert.Error(t, vendorCheck(&config.Configuration{}, &api.RegistrationResponse{}))
	}

	if util.AnyPathsExist(layerAgentPathsPython) && util.PathExists(vendorAgentPathPython) {
		assert.Error(t, vendorCheck(&config.Configuration{}, &api.RegistrationResponse{}))
	}
}
