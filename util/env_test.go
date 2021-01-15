package util

import (
	"os"

	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnvVarExists(t *testing.T) {
	if _, exists := os.LookupEnv("THIS_ENV_VAR_DOES_NOT_EXIST"); !exists {
		assert.False(t, EnvVarExists("THIS_ENV_VAR_DOES_NOT_EXIST"))
	} else {
		assert.True(t, EnvVarExists("THIS_ENV_VAR_DOES_NOT_EXIST"))
	}

	os.Unsetenv("FOO_BAR_BAZ")

	assert.False(t, EnvVarExists("FOO_BAR_BAZ"))

	os.Setenv("FOO_BAR_BAZ", "foo_bar_baz")

	defer func() {
		os.Unsetenv("FOO_BAR_BAZ")
	}()

	assert.True(t, EnvVarExists("FOO_BAR_BAZ"))
}

func TestAnyEnvVarsExist(t *testing.T) {
	if _, exists := os.LookupEnv("THIS_ENV_VAR_DOES_NOT_EXIST"); !exists {
		assert.False(t, AnyEnvVarsExist([]string{"THIS_ENV_VAR_DOES_NOT_EXIST"}))
	} else {
		assert.True(t, AnyEnvVarsExist([]string{"THIS_ENV_VAR_DOES_NOT_EXIST"}))
	}

	os.Unsetenv("FOO_BAR_BAZ")

	assert.False(t, AnyEnvVarsExist([]string{"FOO_BAR_BAZ"}))

	os.Setenv("FOO_BAR_BAZ", "foo_bar_baz")

	defer func() {
		os.Unsetenv("FOO_BAR_BAZ")
	}()

	assert.True(t, AnyEnvVarsExist([]string{"FOO_BAR_BAZ"}))
}

func TestAnyEnvVarsExistString(t *testing.T) {
	if _, exists := os.LookupEnv("THIS_ENV_VAR_DOES_NOT_EXIST"); !exists {
		assert.Equal(t, AnyEnvVarsExistString([]string{"THIS_ENV_VAR_DOES_NOT_EXIST"}), "")
	} else {
		assert.Equal(t, AnyEnvVarsExistString([]string{"THIS_ENV_VAR_DOES_NOT_EXIST"}), "THIS_ENV_VAR_DOES_NOT_EXIST")
	}

	os.Unsetenv("FOO_BAR_BAZ")

	assert.Equal(t, AnyEnvVarsExistString([]string{"FOO_BAR_BAZ"}), "")

	os.Setenv("FOO_BAR_BAZ", "foo_bar_baz")

	defer func() {
		os.Unsetenv("FOO_BAR_BAZ")
	}()

	assert.Equal(t, AnyEnvVarsExistString([]string{"FOO_BAR_BAZ"}), "FOO_BAR_BAZ")
}
