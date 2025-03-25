//go:build !race
// +build !race

package checks

//Testing123
import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuntimeCheck(t *testing.T) {
	dirname, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dirname)

	oldPath := runtimeLookupPath
	defer func() {
		runtimeLookupPath = oldPath
	}()
	runtimeLookupPath = filepath.Join(dirname, runtimeLookupPath)

	os.MkdirAll(filepath.Join(runtimeLookupPath, "node"), os.ModePerm)
	r, err := checkAndReturnRuntime()
	assert.Equal(t, runtimeConfigs[Node].language, r.language)
	assert.Nil(t, err)
}

func TestRuntimeCheckNil(t *testing.T) {
	r, err := checkAndReturnRuntime()
	assert.Equal(t, runtimeConfig{}, r)
	assert.Nil(t, err)
}

func TestLatestAgentTag(t *testing.T) {
	r := &runtimeConfig{agentVersionGitOrg: runtimeConfigs[Python].agentVersionGitOrg, agentVersionGitRepo: runtimeConfigs[Python].agentVersionGitRepo}
	err := latestAgentTag(r)
	assert.NotEmpty(t, r.AgentVersion)
	assert.Nil(t, err)
}

func TestLatestAgentTagError(t *testing.T) {
	r := &runtimeConfig{agentVersionGitOrg: "", agentVersionGitRepo: ""}
	err := latestAgentTag(r)
	assert.Empty(t, r.AgentVersion)
	assert.Nil(t, err)
}
