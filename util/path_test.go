package util

import (
	"os"

	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPathExists(t *testing.T) {
	if _, err := os.Stat("/this/path/should/not/exist"); os.IsNotExist(err) {
		assert.False(t, PathExists("/this/path/should/not/exist"))
	}

	if _, err := os.Stat("/tmp"); !os.IsNotExist(err) {
		assert.True(t, PathExists("/tmp"))
	}
}

func TestAnyPathsExist(t *testing.T) {
	if _, err := os.Stat("/this/path/should/not/exist"); os.IsNotExist(err) {
		assert.False(t, AnyPathsExist([]string{"/this/path/should/not/exist"}))
	}

	if _, err := os.Stat("/tmp"); !os.IsNotExist(err) {
		assert.True(t, AnyPathsExist([]string{"/tmp"}))
	}
}

func TestAnyPathsExistString(t *testing.T) {
	if _, err := os.Stat("/this/path/should/not/exist"); os.IsNotExist(err) {
		assert.Equal(t, AnyPathsExistString([]string{"/this/path/should/not/exist"}), "")
	}

	if _, err := os.Stat("/tmp"); !os.IsNotExist(err) {
		assert.Equal(t, AnyPathsExistString([]string{"/tmp"}), "/tmp")
	}
}
