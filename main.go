package main

import (
	"encoding/json"
	"fmt"
	"github.com/newrelic/lambda-extension/api"
	"github.com/newrelic/lambda-extension/client"
	"log"
	"net/http"
)

func main() {
	fmt.Println("Extension starting up")

	registrationClient := client.New(http.Client{})
	invocationClient, err := registrationClient.RegisterDefault()
	if err != nil {
		log.Fatal(err)
	} else {
		counter := 0
		for {
			counter++

			event, err := invocationClient.NextEvent()
			if err != nil {
				log.Fatal(err)
			}

			jsonStr, _ := json.Marshal(event)
			fmt.Println(string(jsonStr))

			if event.EventType == api.Shutdown {
				break
			}
		}
		fmt.Printf("Shutting down after %v events\n", counter)
	}
}
