package util

import (
	"io"
	"log"
)

// Close closes things and logs errors if it fails
func Close(thing io.Closer) {
	err := thing.Close()
	if err != nil {
		log.Println(err)
	}
}
