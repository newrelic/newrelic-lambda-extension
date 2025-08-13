package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
	"golang.org/x/mod/semver"
)

type LayerAgentVersion struct {
	Version string `json:"version"`
}

// We are only returning an error message when an out of date agent version is detected.
// All other errors will result in a nil return value.
func agentVersionCheck(ctx context.Context, conf *config.Configuration, reg *api.RegistrationResponse, r runtimeConfig) error {
	if r.AgentVersion == "" {
		return nil
	}

	v := LayerAgentVersion{}

	for i := range r.layerAgentPaths {
		f := filepath.Join(r.layerAgentPaths[i], r.agentVersionFile)
		if !util.PathExists(f) {
			continue
		}

		// #nosec G304 - File path is constructed from trusted layer paths and known version file names
		b, err := os.ReadFile(f)
		if err != nil {
			return nil
		}

		if r.language == Python {
			v.Version = string(b)
		} else {
			err = json.Unmarshal([]byte(b), &v)
			if err != nil {
				return nil
			}
		}
	}

	// semver requires a prepended v on version string
	if v.Version != "" && semver.Compare("v"+v.Version, r.AgentVersion) < 0 {
		return fmt.Errorf("Agent version out of date: v%s, in order access up to date features please upgrade to the latest New Relic %s layer that includes agent version %s", v.Version, r.language, r.AgentVersion)
	}

	return nil
}
