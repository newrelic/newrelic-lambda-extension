package checks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

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
func agentVersionCheck(conf *config.Configuration, reg *api.RegistrationResponse, r runtimeConfig) error {
	if r.AgentVersion == "" {
		return nil
	}

	for i := range r.layerAgentPaths {
		f := fmt.Sprintf("%s/%s", r.layerAgentPaths[i], r.agentVersionFile)
		if util.PathExists(f) {
			b, err := ioutil.ReadFile(f)
			if err != nil {
				return nil
			}
			v := LayerAgentVersion{}
			if r.language == Python {
				v.Version = string(b)
			} else {
				err = json.Unmarshal([]byte(b), &v)
				if err != nil {
					return nil
				}
			}
			// semver requires a prepended v on version string
			if semver.Compare("v"+v.Version, r.AgentVersion) < 0 {
				return fmt.Errorf("Agent version out of date: v%s, in order access up to date features please upgrade to the latest New Relic %s layer that includes agent version %s", v.Version, r.language, r.AgentVersion)
			}
			return nil
		}
	}
	return nil
}
