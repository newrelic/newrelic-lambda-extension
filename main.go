package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/newrelic/lambda-extension/client"
)

func main() {
	fmt.Println("Extension starting up")

	httpClient := http.Client{}
	counter := 0
	ic, err := client.RegisterDefault(httpClient)
	if err != nil {
		log.Fatal(err)
	}

	for {
		counter++
		err := client.NextEvent(ic)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("event number: %v\n", counter)
	}
}
