package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {

	client := http.Client{}
	counter := 0
	ic, err := registerDefault(client)
	if err != nil {
		log.Fatal(err)
	}

	for {
		counter++
		err := nextEvent(ic)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("event number: %v\n", counter)
	}

}
