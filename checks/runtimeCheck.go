package checks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

var (
	layerAgentPathNode    = []string{"/opt/nodejs/node_modules/newrelic"}
	layerAgentPathsPython = []string{
		"/opt/python/lib/python2.7/site-packages/newrelic",
		"/opt/python/lib/python3.6/site-packages/newrelic",
		"/opt/python/lib/python3.7/newrelic",
		"/opt/python/lib/python3.8/site-packages/newrelic",
	}
	vendorAgentPathNode   = "/var/task/node_modules/newrelic"
	vendorAgentPathPython = "/var/task/newrelic"
)

type runtimeConfig struct {
	AgentVersion     string `json:"latest_release_number"`
	agentVersionUrl  string
	agentVersionFile string
	fileType         string
	language         Runtime
	layerAgentPaths  []string
	vendorAgentPath  string
	wrapperName      string
}

type Runtime string

const (
	Python Runtime = "python"
	Node   Runtime = "node"
)

// Runtime static values
var runtimeConfigs = map[Runtime]runtimeConfig{
	Node: {
		language:         Node,
		wrapperName:      "newrelic-lambda-wrapper.handler",
		fileType:         "js",
		layerAgentPaths:  layerAgentPathNode,
		vendorAgentPath:  vendorAgentPathNode,
		agentVersionFile: "package.json",
		agentVersionUrl:  "https://libraries.io/api/npm/newrelic/",
	},
	Python: {
		language:         Python,
		wrapperName:      "newrelic_lambda_wrapper.handler",
		fileType:         "py",
		layerAgentPaths:  layerAgentPathsPython,
		vendorAgentPath:  vendorAgentPathPython,
		agentVersionFile: "version.txt",
		agentVersionUrl:  "https://libraries.io/api/pypi/newrelic/",
	},
}

func checkAndReturnRuntime() (runtimeConfig, error) {
	for k, v := range runtimeConfigs {
		p := fmt.Sprintf("/var/lang/bin/%s", k)
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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	_ = json.Unmarshal([]byte(body), &r)
	defer resp.Body.Close()
	return nil
}
