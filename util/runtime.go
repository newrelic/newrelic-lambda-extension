package util

import (
	"os"
)

// DetectRuntime detects which Lambda runtime we are running by checking for the existence of
// runtime specific directories. Returns values such as "nodejs" or "python", or
// "unknown if runtime is not known.
func DetectRuntime() string {
	if _, err := os.Stat("/opt/nodejs"); !os.IsNotExist(err) {
		return "nodejs"
	}

	if _, err := os.Stat("/opt/python"); !os.IsNotExist(err) {
		return "python"
	}

	return "unknown"
}
