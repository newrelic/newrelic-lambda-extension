package checks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

func checkAndReturnRuntime() (runtimeConfig, error) {
	for k, v := range runtimeConfigs {
		p := fmt.Sprintf("%s/%s", runtimeLookupPath, k)
		if util.PathExists(p) {
			err := latestAgentTag(&v)
			return v, err
		}
	}
	// If we make it here that means the runtime is not one we
	// currently validate so we don't want to warn against anything
	return runtimeConfig{}, nil
}

func latestAgentTag(r *runtimeConfig) error {
	resp, err := http.Get(r.agentVersionUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		// The version check HTTP request failed; this doesn't tell us anything
		util.Debugf("Can't query latest agent version. Request to %v returned status %v", r.agentVersionUrl, resp.StatusCode)
		return nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, &r)
	if err != nil {
		return err
	}
	return nil
}
