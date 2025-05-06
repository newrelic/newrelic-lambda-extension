package checks

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

const (
	nrlambdaImport = `"github.com/newrelic/go-agent/v3/integrations/nrlambda"`
	newrelicImport = `"github.com/newrelic/go-agent/v3/newrelic"`
)

// vendorCheck checks to see if the user included a vendored copy of the agent along
// with their function while also using a layer that includes the agent
func vendorCheck(ctx context.Context, _ *config.Configuration, _ *api.RegistrationResponse, r runtimeConfig) error {

	if util.PathExists(r.vendorAgentPath) && util.AnyPathsExist(r.layerAgentPaths) {
		return fmt.Errorf("Vendored agent found at '%s', a layer already includes this agent at '%s'. Recommend using the layer agent to avoid unexpected agent behavior.", r.vendorAgentPath, util.AnyPathsExistString(r.layerAgentPaths))
	}

	return nil
}

func bootstrapCheck(ctx context.Context, _ *config.Configuration, _ *api.RegistrationResponse, r runtimeConfig) error {
	time.Sleep(50 * time.Millisecond)
	bootstrapPath := "/var/task/bootstrap"

	contentBytes, _ := os.ReadFile(bootstrapPath)
	content := string(contentBytes)
	// Check for the presence of the New Relic nrlambda import
	nrlambdaMatch := strings.Contains(content, nrlambdaImport)

	// Check for the presence of the New Relic agent import
	newrelicMatch := strings.Contains(content, newrelicImport)

	if !nrlambdaMatch || !newrelicMatch {
		return fmt.Errorf("necessary imports are not available in bootstrap")
	}

	return nil
}
