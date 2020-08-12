package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/newrelic/lambda-extension/api"
	"github.com/newrelic/lambda-extension/client"
	"github.com/newrelic/lambda-extension/credentials"
	"github.com/newrelic/lambda-extension/telemetry"
	"github.com/newrelic/lambda-extension/util"
)

const telemetryNamedPipePath = "/tmp/newrelic-telemetry"

func initTelemetryChannel() (chan []byte, error) {
	_ = os.Remove(telemetryNamedPipePath)

	err := syscall.Mkfifo(telemetryNamedPipePath, 0666)
	if err != nil {
		return nil, err
	}

	telemetryChan := make(chan []byte)

	go func() {
		for {
			telemetryChan <- pollForTelemetry()
		}
	}()

	return telemetryChan, nil
}

func pollForTelemetry() []byte {
	// Opening a pipe will block, until the write side has been opened as well
	telemetryPipe, err := os.OpenFile(telemetryNamedPipePath, os.O_RDONLY, 0)
	if err != nil {
		log.Panic("failed to open telemetry pipe", err)
	}

	defer util.Close(telemetryPipe)

	// When the write side closes, we get an EOF.
	bytes, err := ioutil.ReadAll(telemetryPipe)
	if err != nil {
		log.Panic("failed to read telemetry pipe", err)
	}

	return bytes
}

func main() {
	extensionStartup := time.Now()
	log.Println("New Relic Lambda Extension starting up")

	registrationClient := client.New(http.Client{})

	invocationClient, registrationResponse, err := registrationClient.RegisterDefault()
	if err != nil {
		log.Fatal(err)
	}

	util.LogAsJSON(registrationResponse)

	_, found := os.LookupEnv("NEW_RELIC_CLOUDWATCH_INGEST")
	if found {
		log.Println("Extension telemetry path disabled")
		noopLoop(invocationClient)
		return
	}

	licenseKey, err := credentials.GetNewRelicLicenseKey()
	if err != nil {
		log.Println("Failed to retrieve license key", err)
		// Don't create the telemetry named pipe, just silently pump events
		noopLoop(invocationClient)
		return
	}

	log.Println("Fetched license key from managed secret.")

	telemetryClient := telemetry.New(registrationResponse, *licenseKey)

	telemetryChan, err := initTelemetryChannel()
	if err != nil {
		log.Fatal("telemetry pipe init failed: ", err)
	}

	counter := 0
	for {
		event, err := invocationClient.NextEvent()
		if err != nil {
			// TODO: extension error API
			log.Fatal(err)
		}

		counter++

		util.LogAsJSON(event)

		if event.EventType == api.Shutdown {
			break
		}

		log.Printf("Awaiting telemetry channel...")
		telemetryBytes := <-telemetryChan
		log.Printf("Telemetry: %s", string(telemetryBytes))

		res, body, err := telemetryClient.Send(event, telemetryBytes)
		if err != nil {
			log.Printf("Telemetry client error: %s", err)
		} else {
			log.Printf("Telemetry client response: [%s] %s", res.Status, body)
		}
	}

	log.Printf("Shutting down after %v events\n", counter)

	shutdownAt := time.Now()
	ranFor := shutdownAt.Sub(extensionStartup)
	log.Printf("Extension shutdown after %vms", ranFor.Milliseconds())
}

func noopLoop(invocationClient *client.InvocationClient) {
	for {
		event, err := invocationClient.NextEvent()
		if err != nil {
			// TODO: extension error API
			log.Fatal(err)
		}

		if event.EventType == api.Shutdown {
			return
		}
	}
}
