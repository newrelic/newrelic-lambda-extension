package checks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/stretchr/testify/assert"
)

func TestAgentVersion(t *testing.T) {
	conf := config.Configuration{}
	reg := api.RegistrationResponse{}
	r := runtimeConfig{}
	ctx := context.Background()

	// No version set
	err := agentVersionCheck(ctx, &conf, &reg, r)
	assert.Nil(t, err)

	// Error
	dirname, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dirname)

	testFile := filepath.Join(dirname, "opt", "python", "lib", "python3.8", "site-packages", "newrelic")
	r = runtimeConfigs[Python]
	r.AgentVersion = "v10.1.2"
	r.layerAgentPaths = []string{testFile}

	os.MkdirAll(testFile, os.ModePerm)
	f, _ := os.Create(filepath.Join(testFile, r.agentVersionFile))
	f.WriteString("10.1.0")

	err = agentVersionCheck(ctx, &conf, &reg, r)
	assert.EqualError(t, err, "Agent version out of date: v10.1.0, in order access up to date features please upgrade to the latest New Relic python layer that includes agent version v10.1.2")

	// Success
	r.AgentVersion = "10.1.0"
	err = agentVersionCheck(ctx, &conf, &reg, r)
	assert.Nil(t, err)
}
