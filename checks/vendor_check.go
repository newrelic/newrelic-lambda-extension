package checks

import (
	"fmt"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

var (
	layerAgentPathNode    = "/opt/nodejs/node_modules/newrelic"
	layerAgentPathsPython = []string{
		"/opt/python/lib/python2.7/site-packages/newrelic",
		"/opt/python/lib/python3.6/site-packages/newrelic",
		"/opt/python/lib/python3.7/newrelic",
		"/opt/python/lib/python3.8/site-packages/newrelic",
	}
	vendorAgentPathNode   = "/var/task/node_modules/newrelic"
	vendorAgentPathPython = "/var/task/newrelic"
)

// vendorCheck checks to see if the user included a vendored copy of the agent along
// with their function while also using a layer that includes the agent
func vendorCheck(*config.Configuration, *api.RegistrationResponse) error {
	if util.PathExists(vendorAgentPathNode) && util.PathExists(layerAgentPathNode) {
		return fmt.Errorf("Vendored agent found at '%s', a layer already includes this agent at '%s'. Recommend using one or the other to avoid unexpected agent behavior.", vendorAgentPathNode, layerAgentPathNode)
	}

	if util.PathExists(vendorAgentPathPython) && util.AnyPathsExist(layerAgentPathsPython) {
		return fmt.Errorf("Vendored agent found at '%s', a layer already includes this agent at '%s'. Recommend using one or the other to avoid unexpected agent behavior.", vendorAgentPathPython, util.AnyPathsExistString(layerAgentPathsPython))
	}

	return nil
}
