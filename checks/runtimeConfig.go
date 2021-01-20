package checks

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
	runtimeLookupPath     = "/var/lang/bin"
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
