package checks

import (
	"os"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/stretchr/testify/assert"
)

func TestAgentVersion(t *testing.T) {
	conf := config.Configuration{}
	reg := api.RegistrationResponse{}
	r := runtimeConfig{}

	// No version set
	err := agentVersionCheck(&conf, &reg, r)
	assert.Nil(t, err)

	// Error
	dirname, err := os.Getwd()
	testFile := dirname + "/opt/python/lib/python3.8/site-packages/newrelic/"

	r = runtimeConfigs[Python]
	r.AgentVersion = "10.1.2"
	r.layerAgentPaths = []string{testFile}
	err = os.MkdirAll(testFile, os.ModePerm)
	f, err := os.Create(testFile + r.agentVersionFile)
	f.WriteString("10.1.0")

	err = agentVersionCheck(&conf, &reg, r)
	assert.EqualError(t, err, "Agent version out of date: v10.1.0, in order access up to date features please upgrade to the latest New Relic python layer that includes agent version 10.1.2")

	// Success
	r.AgentVersion = "10.1.0"
	err = agentVersionCheck(&conf, &reg, r)
	assert.Nil(t, err)

	os.RemoveAll(dirname + "/opt")
}
