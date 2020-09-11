package main

import (
	"log"
	"net/http"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/credentials"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/client"
	"github.com/newrelic/newrelic-lambda-extension/telemetry"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

func main() {
	extensionStartup := time.Now()
	log.Println("New Relic Lambda Extension starting up")

	registrationClient := client.New(http.Client{})
	regReq := api.RegistrationRequest{
		Events:            []api.LifecycleEvent{api.Invoke, api.Shutdown},
	}
	invocationClient, registrationResponse, err := registrationClient.Register(regReq)
	if err != nil {
		log.Fatal(err)
	}
	conf := config.ConfigurationFromEnvironment()
	util.LogAsJSON(registrationResponse)

	if conf.UseCloudWatchIngest {
		log.Println("Extension telemetry processing disabled")
		noopLoop(invocationClient)
		return
	}

	licenseKey, err := credentials.GetNewRelicLicenseKey(&conf)
	if err != nil {
		log.Println("Failed to retrieve license key", err)
		// Don't create the telemetry named pipe, just silently pump events
		noopLoop(invocationClient)
		return
	}

	telemetryClient := telemetry.New(registrationResponse.FunctionName, *licenseKey, conf.TelemetryEndpoint)

	telemetryChan, err := telemetry.InitTelemetryChannel()
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

		telemetryBytes := <-telemetryChan
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
