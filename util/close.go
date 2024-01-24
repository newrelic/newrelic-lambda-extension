package util

import (
	"io"
)

// Close closes things and logs errors if it fails
func Close(thing io.Closer) {
	err := thing.Close()
	if err != nil {
		Infoln(err)
	}
}
