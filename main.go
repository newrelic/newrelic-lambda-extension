package main

import (
	"encoding/json"
	"github.com/newrelic/lambda-extension/api"
	"github.com/newrelic/lambda-extension/client"
	"log"
	"net/http"
)

func logAsJson(v interface{}) {
	indent, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Panic(err)
	}
	log.Println(string(indent))
}

func main() {
	log.Println("Extension starting up")

	registrationClient := client.New(http.Client{})
	invocationClient, registrationResponse, err := registrationClient.RegisterDefault()
	if err != nil {
		log.Fatal(err)
	} else {
		logAsJson(registrationResponse)
		counter := 0
		for {
			counter++

			event, err := invocationClient.NextEvent()
			if err != nil {
				log.Fatal(err)
			}

			logAsJson(event)

			if event.EventType == api.Shutdown {
				break
			}
		}
		log.Printf("Shutting down after %v events\n", counter)
	}
}
