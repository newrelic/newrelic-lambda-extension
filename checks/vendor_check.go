package checks

import (
	"context"
	"debug/buildinfo" // Import the package to read build info from Go binaries
	"fmt"
	"strings"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

// Constants for module paths to check in the build information.
const (
	nrLambdaModulePath = "github.com/newrelic/go-agent/v3/integrations/nrlambda"
	newrelicModulePath = "github.com/newrelic/go-agent/v3"
)

// runtimeConfig holds configuration details relevant to the checks.

// vendorCheck checks to see if the user included a vendored copy of the agent along
// with their function while also using a layer that includes the agent
func vendorCheck(ctx context.Context, _ *config.Configuration, _ *api.RegistrationResponse, r runtimeConfig) error {
	if r.vendorAgentPath != "" && len(r.layerAgentPaths) > 0 { // Basic check for non-empty paths
		if util.PathExists(r.vendorAgentPath) && util.AnyPathsExist(r.layerAgentPaths) {
			return fmt.Errorf("Vendored agent found at '%s', and a layer also includes this agent (e.g., at '%s'). Recommend using only the layer agent to avoid unexpected agent behavior.", r.vendorAgentPath, util.AnyPathsExistString(r.layerAgentPaths))
		}
	}
	return nil
}

// bootstrapCheck inspects the compiled Go application's module information using debug/buildinfo.ReadFile.
// It verifies that the necessary New Relic Go Agent modules are linked into the application.
func bootstrapCheck(ctx context.Context, _ *config.Configuration, _ *api.RegistrationResponse, r runtimeConfig) error {
	compiledAppPath := "/var/task/bootstrap"

	util.Logf("Starting bootstrap check for New Relic modules in '%s' using debug/buildinfo...", compiledAppPath)

	bi, err := buildinfo.ReadFile(compiledAppPath)
	if err != nil {
		util.Logf("Error reading build info from '%s': %v", compiledAppPath, err)
		return fmt.Errorf("failed to read Go module information from '%s'. Ensure it is a valid Go binary compiled with Go 1.18+ and includes build information. Original error: %v", compiledAppPath, err)
	}

	// --- Start of Diagnostic Logging ---
	util.Logf("Successfully read build info from '%s'. Main module path: '%s', Go version: '%s'", compiledAppPath, bi.Main.Path, bi.GoVersion)
	util.Logln("Dependencies found in build info:")
	if len(bi.Deps) == 0 {
		util.Logln("  <No dependencies listed>")
	}
	for i, dep := range bi.Deps {
		util.Logf("  Dep %d: Path='%s', Version='%s'", i+1, dep.Path, dep.Version)
		if dep.Replace != nil {
			util.Logf("    Replaced by: Path='%s', Version='%s'", dep.Replace.Path, dep.Replace.Version)
		}
	}
	// --- End of Diagnostic Logging ---

	var nrlambdaFound bool
	var newrelicFound bool

	// Check nrLambdaModulePath
	if bi.Main.Path == nrLambdaModulePath {
		nrlambdaFound = true
		util.Logf("Diagnostic: Found '%s' as the main module path.", nrLambdaModulePath) // Diagnostic log
	}
	if !nrlambdaFound { // Only check dependencies if not found in main module
		for _, dep := range bi.Deps {
			if dep.Path == nrLambdaModulePath {
				nrlambdaFound = true
				util.Logf("Diagnostic: Found '%s' as a dependency.", nrLambdaModulePath) // Diagnostic log
				break
			}
		}
	}

	// Check newrelicModulePath
	if bi.Main.Path == newrelicModulePath {
		newrelicFound = true
		util.Logf("Diagnostic: Found '%s' as the main module path.", newrelicModulePath) // Diagnostic log
	}
	if !newrelicFound { // Only check dependencies if not found in main module
		for _, dep := range bi.Deps {
			if dep.Path == newrelicModulePath {
				newrelicFound = true
				util.Logf("Diagnostic: Found '%s' as a dependency.", newrelicModulePath) // Diagnostic log
				break
			}
		}
	}

	util.Logf("Diagnostic: Module check results before final decision: nrlambdaFound=%t, newrelicFound=%t", nrlambdaFound, newrelicFound) // Diagnostic log

	if !nrlambdaFound || !newrelicFound {
		// Construct a more detailed message if specific modules are missing, for better debugging.
		// This will be part of the error returned, which is then logged by the calling `runCheck` function.
		var missingModulesDetail []string
		if !nrlambdaFound {
			missingModulesDetail = append(missingModulesDetail, nrLambdaModulePath)
		}
		if !newrelicFound {
			missingModulesDetail = append(missingModulesDetail, newrelicModulePath)
		}
		// The user wants to see "necessary imports are not available in bootstrap"
		// but the detailed string is better for debugging if that generic message is insufficient.
		// For now, sticking to the user's requested simpler error message for the return.
		util.Logf("Error: One or both New Relic modules are missing. nrlambdaFound: %t, newrelicFound: %t. Missing identified: %s", nrlambdaFound, newrelicFound, strings.Join(missingModulesDetail, ", "))
		return fmt.Errorf("necessary imports are not available in bootstrap")
	}

	util.Logf("Bootstrap check passed: New Relic modules ('%s', '%s') are present in '%s'.", nrLambdaModulePath, newrelicModulePath, compiledAppPath)
	return nil
}
