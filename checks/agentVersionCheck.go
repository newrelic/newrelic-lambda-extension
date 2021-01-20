package checks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

type LayerAgentVersion struct {
	Version string `json:"version"`
}

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
				_ = json.Unmarshal([]byte(b), &v)
			}
			return r.semverValidation(v.Version)
		}
	}
	return nil
}

func (r runtimeConfig) semverValidation(v string) error {
	l := strings.Split(v, ".")
	a := strings.Split(r.AgentVersion, ".")

	for i := 0; i < len(l); i++ {
		if l[i] > a[i] {
			return nil
		} else if l[i] < a[i] {
			return fmt.Errorf("Agent version out of date: v%s, in order access up to date features please upgrade to the latest New Relic %s layer that includes agent version %s", v, r.language, r.AgentVersion)
		}
	}
	return nil
}
