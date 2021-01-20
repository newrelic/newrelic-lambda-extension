package checks

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuntimeCheck(t *testing.T) {
	dirname, _ := os.Getwd()
	runtimeLookupPath = fmt.Sprintf("%s/%s", dirname, runtimeLookupPath)
	os.MkdirAll(runtimeLookupPath+"/node", os.ModePerm)
	defer os.RemoveAll(dirname + "/var")
	r, err := checkAndReturnRuntime()
	assert.Equal(t, runtimeConfigs[Node].language, r.language)
	assert.Nil(t, err)
}

func TestRuntimeCheckNil(t *testing.T) {
	r, err := checkAndReturnRuntime()
	assert.Equal(t, runtimeConfig{}, r)
	assert.Nil(t, err)
}
