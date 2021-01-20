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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(body), &r)
	if err != nil {
		return err
	}
	return nil
}
