package main

import (
	"github.com/newrelic/newrelic-lambda-extension/logserver"
	"log"
	"net/http"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/credentials"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/client"
	"github.com/newrelic/newrelic-lambda-extension/telemetry"
)

func main() {
	extensionStartup := time.Now()
	log.Println("New Relic Lambda Extension starting up")

	registrationClient := client.New(http.Client{})
	regReq := api.RegistrationRequest{
		Events: []api.LifecycleEvent{api.Invoke, api.Shutdown},
	}

	invocationClient, registrationResponse, err := registrationClient.Register(regReq)
	if err != nil {
		log.Fatal(err)
	}

	conf := config.ConfigurationFromEnvironment()

	if !conf.ExtensionEnabled {
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

	// set up the telemetry buffer
	batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis))

	logs, err := logserver.Start()
	if err != nil {
		log.Println("Failed to start logs HTTP server", err)
		err = invocationClient.InitError("logs.start", err)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	endpoint := api.FormatLogsEndpoint(logs.Port())
	subscriptionRequest := api.DefaultLogSubscription([]api.LogEventType{api.Platform, api.Function}, endpoint)
	err = invocationClient.LogRegister(&subscriptionRequest)
	if err != nil {
		log.Println("Failed to register with Logs API", err)
		err = invocationClient.InitError("logs.register", err)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	telemetryClient := telemetry.New(registrationResponse.FunctionName, *licenseKey, conf.TelemetryEndpoint)

	telemetryChan, err := telemetry.InitTelemetryChannel()
	if err != nil {
		log.Fatal("telemetry pipe init failed: ", err)
	}

	counter := 0
	invokedFunctionARN := ""
	for {
		event, err := invocationClient.NextEvent()
		now := time.Now()
		if err != nil {
			errErr := invocationClient.ExitError("NextEventError.Main", err)
			if errErr != nil {
				log.Println(errErr)
			}
			log.Fatal(err)
		}

		counter++

		if event.EventType == api.Shutdown {
			break
		}

		invokedFunctionARN = event.InvokedFunctionARN

		batch.AddInvocation(event.RequestID, now)

		telemetryBytes := <-telemetryChan

		inv := batch.AddTelemetry(event.RequestID, telemetryBytes)
		if inv == nil {
			log.Printf("Failed to add telemetry for request %v", event.RequestID)
		}

		harvested := batch.Harvest(time.Now())
		shipHarvest(harvested, telemetryClient, invokedFunctionARN)
	}
	log.Printf("New Relic Extension shutting down after %v events\n", counter)

	err = logs.Close()
	if err != nil {
		log.Println("Error shutting down logs server", err)
	}

	finalHarvest := batch.Close()
	shipHarvest(finalHarvest, telemetryClient, invokedFunctionARN)

	shutdownAt := time.Now()
	ranFor := shutdownAt.Sub(extensionStartup)
	log.Printf("Extension shutdown after %vms", ranFor.Milliseconds())
}

func shipHarvest(harvested []*telemetry.Invocation, telemetryClient *telemetry.Client, invokedFunctionARN string) {
	if len(harvested) > 0 {
		telemetrySlice := make([][]byte, 0, 2*len(harvested))
		for _, inv := range harvested {
			telemetrySlice = append(telemetrySlice, inv.Telemetry...)
		}

		err := telemetryClient.SendTelemetry(invokedFunctionARN, telemetrySlice)
		if err != nil {
			log.Printf("Failed to send harvested telemetry for %d invocations %s", len(harvested), err)
		}
	}
}

func noopLoop(invocationClient *client.InvocationClient) {
	for {
		event, err := invocationClient.NextEvent()
		if err != nil {
			errErr := invocationClient.ExitError("NextEventError.Noop", err)
			if errErr != nil {
				log.Println(errErr)
			}
			log.Fatal(err)
		}

		if event.EventType == api.Shutdown {
			return
		}
	}
}
