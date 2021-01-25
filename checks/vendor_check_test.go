package checks

import (
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
	"github.com/stretchr/testify/assert"
)

func TestVendorCheck(t *testing.T) {

	n := runtimeConfigs[Node]

	if !util.AnyPathsExist(n.layerAgentPaths) && !util.PathExists(n.vendorAgentPath) {
		assert.Nil(t, vendorCheck(&config.Configuration{}, &api.RegistrationResponse{}, n))
	}

	if util.PathExists(n.layerAgentPaths[0]) && util.PathExists(n.vendorAgentPath) {
		assert.Error(t, vendorCheck(&config.Configuration{}, &api.RegistrationResponse{}, n))
	}

	p := runtimeConfigs[Python]

	if !util.AnyPathsExist(p.layerAgentPaths) && !util.PathExists(p.vendorAgentPath) {
		assert.Nil(t, vendorCheck(&config.Configuration{}, &api.RegistrationResponse{}, n))
	}

	if util.AnyPathsExist(p.layerAgentPaths) && util.PathExists(p.vendorAgentPath) {
		assert.Error(t, vendorCheck(&config.Configuration{}, &api.RegistrationResponse{}, n))
	}
}
