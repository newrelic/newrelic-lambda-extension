package util

import (
	"os"

	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDetectRuntime(t *testing.T) {
	nodeDirExists := false
	_, err := os.Stat("/opt/nodejs")
	nodeDirExists = !os.IsNotExist(err)

	pythonDirExists := false
	_, err = os.Stat("/opt/python")
	pythonDirExists = !os.IsNotExist(err)

	if !nodeDirExists && !pythonDirExists {
		assert.Equal(t, DetectRuntime(), "unknown")
	}

	if nodeDirExists {
		assert.Equal(t, DetectRuntime(), "nodejs")
	}

	if !nodeDirExists && pythonDirExists {
		assert.Equal(t, DetectRuntime(), "python")
	}
}
