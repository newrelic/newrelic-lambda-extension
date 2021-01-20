package util

import (
	"os"
)

// EnvVarExists returns true if environment variable exists, false if not
func EnvVarExists(key string) bool {
	_, exists := os.LookupEnv(key)
	return exists
}

// AnyEnvVarsExist returns true if any of the environment variables exist, false if none
// exist
func AnyEnvVarsExist(keys []string) bool {
	for _, key := range keys {
		if EnvVarExists(key) {
			return true
		}
	}

	return false
}

// AnyEnvVarsExistString is the same as AnyEnvVarsExist only it returns the environment
// variable key of the first environment variable that exists, or an empty string if
// none exist
func AnyEnvVarsExistString(keys []string) string {
	for _, key := range keys {
		if EnvVarExists(key) {
			return key
		}
	}

	return ""
}
