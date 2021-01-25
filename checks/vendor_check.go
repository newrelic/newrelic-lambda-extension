package checks

import (
	"fmt"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

// vendorCheck checks to see if the user included a vendored copy of the agent along
// with their function while also using a layer that includes the agent
func vendorCheck(_ *config.Configuration, _ *api.RegistrationResponse, r runtimeConfig) error {

	if util.PathExists(r.vendorAgentPath) && util.AnyPathsExist(r.layerAgentPaths) {
		return fmt.Errorf("Vendored agent found at '%s', a layer already includes this agent at '%s'. Recommend using the layer agent to avoid unexpected agent behavior.", r.vendorAgentPath, util.AnyPathsExistString(r.layerAgentPaths))
	}

	return nil
}
