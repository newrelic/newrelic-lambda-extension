package util

import (
	"io"

	log "github.com/sirupsen/logrus"
)

var l = log.WithFields(log.Fields{"pkg": "util"})

// Close closes things and logs errors if it fails
func Close(thing io.Closer) {
	err := thing.Close()
	if err != nil {
		l.Error(err)
	}
}
