package checks

var (
	layerAgentPathNode    = []string{"/opt/nodejs/node_modules/newrelic"}
	layerAgentPathsPython = []string{
		"/opt/python/lib/python2.7/site-packages/newrelic",
		"/opt/python/lib/python3.6/site-packages/newrelic",
		"/opt/python/lib/python3.7/newrelic",
		"/opt/python/lib/python3.8/site-packages/newrelic",
		"/opt/python/lib/python3.9/site-packages/newrelic",
	}
	vendorAgentPathNode   = "/var/task/node_modules/newrelic"
	vendorAgentPathPython = "/var/task/newrelic"
	runtimeLookupPath     = "/var/lang/bin"
)

type runtimeConfig struct {
	AgentVersion     string
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
		agentVersionUrl:  "https://github.com/newrelic/node-newrelic/releases/latest",
	},
	Python: {
		language:         Python,
		wrapperName:      "newrelic_lambda_wrapper.handler",
		fileType:         "py",
		layerAgentPaths:  layerAgentPathsPython,
		vendorAgentPath:  vendorAgentPathPython,
		agentVersionFile: "version.txt",
		agentVersionUrl:  "https://github.com/newrelic/newrelic-python-agent/releases/latest",
	},
}
