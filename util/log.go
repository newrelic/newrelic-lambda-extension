package util

import (
	"encoding/json"
	"log"
)

func LogAsJSON(v interface{}) {
	indent, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Panic(err)
	}

	log.Println(string(indent))
}
