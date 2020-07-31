package util

import (
	"log"
)

type closeable interface {
	Close() error
}

// Close closes things and logs errors if it fails
func Close(thing closeable) {
	err := thing.Close()
	if err != nil {
		log.Println(err)
	}
}
