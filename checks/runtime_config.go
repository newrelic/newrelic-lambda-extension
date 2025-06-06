package checks

var (
	LayerAgentPathNode    = []string{"/opt/nodejs/node_modules/newrelic"}
	LayerAgentPathsPython = []string{
		"/opt/python/lib/python3.9/site-packages/newrelic",
		"/opt/python/lib/python3.10/site-packages/newrelic",
		"/opt/python/lib/python3.11/site-packages/newrelic",
		"/opt/python/lib/python3.12/site-packages/newrelic",
		"/opt/python/lib/python3.13/site-packages/newrelic",
	}
	layerAgentPathsRuby    = []string{
    "/opt/ruby/gems/3.2.0/gems/newrelic_rpm",
    "/opt/ruby/gems/3.3.0/gems/newrelic_rpm",
  }
	vendorAgentPathNode   = "/var/task/node_modules/newrelic"
	vendorAgentPathPython = "/var/task/newrelic"
	vendorAgentPathRuby   = "/var/task/vendor/bundle/ruby/3.3.0/gems/newrelic_rpm"
	runtimeLookupPath     = "/var/lang/bin"
)

type runtimeConfig struct {
	AgentVersion        string
	agentVersionGitOrg  string
	agentVersionGitRepo string
	agentVersionFile    string
	fileType            string
	language            Runtime
	layerAgentPaths     []string
	vendorAgentPath     string
	wrapperName         string
}

type Runtime string

const (
	Python Runtime = "python"
	Node   Runtime = "node"
	Ruby   Runtime = "ruby"
)

// Runtime static values
var runtimeConfigs = map[Runtime]runtimeConfig{
	Node: {
		language:            Node,
		wrapperName:         "newrelic-lambda-wrapper.handler",
		fileType:            "js",
		layerAgentPaths:     LayerAgentPathNode,
		vendorAgentPath:     vendorAgentPathNode,
		agentVersionFile:    "package.json",
		agentVersionGitOrg:  "newrelic",
		agentVersionGitRepo: "node-newrelic",
	},
	Python: {
		language:            Python,
		wrapperName:         "newrelic_lambda_wrapper.handler",
		fileType:            "py",
		layerAgentPaths:     LayerAgentPathsPython,
		vendorAgentPath:     vendorAgentPathPython,
		agentVersionFile:    "version.txt",
		agentVersionGitOrg:  "newrelic",
		agentVersionGitRepo: "newrelic-python-agent",
	},
	Ruby: {
		language:        Ruby,
		wrapperName:     "newrelic_lambda_wrapper.handler",
		fileType:        "rb",
		layerAgentPaths: layerAgentPathsRuby,
		vendorAgentPath: vendorAgentPathRuby,
		// TODO: requires Ruby to parse out the version
		agentVersionFile:    "lib/new_relic/version.rb",
		agentVersionGitOrg:  "newrelic",
		agentVersionGitRepo: "newrelic-ruby-agent",
	},
}
