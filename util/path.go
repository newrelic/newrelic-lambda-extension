package util

import (
	"os"
)

// PathExists returns true of a path exists, false if it does not
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// AnyPathsExists returns true if any of the provided paths exist, false if none exist
func AnyPathsExist(paths []string) bool {
	for _, path := range paths {
		if PathExists(path) {
			return true
		}
	}

	return false
}

// AnyPathsExistString is same as AnyPathsExist, only it returns the first path that
// exists or an empty string if none exist
func AnyPathsExistString(paths []string) string {
	for _, path := range paths {
		if PathExists(path) {
			return path
		}
	}

	return ""
}
