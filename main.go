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

	// Go Logging config
	log.SetPrefix("[NR_EXT] ")
	log.SetFlags(0)

	log.Println("New Relic Lambda Extension starting up")

	// Extensions must register
	registrationClient := client.New(http.Client{})
	regReq := api.RegistrationRequest{
		Events: []api.LifecycleEvent{api.Invoke, api.Shutdown},
	}

	invocationClient, registrationResponse, err := registrationClient.Register(regReq)
	if err != nil {
		log.Fatal(err)
	}

	// Parse various env vars for our config
	conf := config.ConfigurationFromEnvironment()

	if !conf.ExtensionEnabled {
		log.Println("Extension telemetry processing disabled")
		noopLoop(invocationClient)
		return
	}

	// Attempt to find the license key for telemetry sending
	licenseKey, err := credentials.GetNewRelicLicenseKey(&conf)
	if err != nil {
		log.Println("Failed to retrieve license key", err)
		// We fail open; telemetry will go to CloudWatch instead
		noopLoop(invocationClient)
		return
	}

	// Set up the telemetry buffer
	batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis))

	// Start the Logs API server, and register it
	logServer, err := logserver.Start()
	if err != nil {
		log.Println("Failed to start logs HTTP server", err)
		err = invocationClient.InitError("logServer.start", err)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	subscriptionRequest := api.DefaultLogSubscription([]api.LogEventType{api.Platform}, logServer.Port())
	err = invocationClient.LogRegister(&subscriptionRequest)
	if err != nil {
		log.Println("Failed to register with Logs API", err)
		err = invocationClient.InitError("logServer.register", err)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// Init the telemetry sending client
	telemetryClient := telemetry.New(registrationResponse.FunctionName, *licenseKey, conf.TelemetryEndpoint)

	telemetryChan, err := telemetry.InitTelemetryChannel()
	if err != nil {
		log.Fatal("telemetry pipe init failed: ", err)
	}

	// Call next, and process telemetry, until we're shut down
	mainLoop(invocationClient, &batch, telemetryChan, logServer, telemetryClient)

	shutdownAt := time.Now()
	ranFor := shutdownAt.Sub(extensionStartup)
	log.Printf("Extension shutdown after %vms", ranFor.Milliseconds())
}

func mainLoop(invocationClient *client.InvocationClient, batch *telemetry.Batch, telemetryChan chan []byte, logServer *logserver.LogServer, telemetryClient *telemetry.Client) {
	counter := 0
	var invokedFunctionARN string
	for {
		event, err := invocationClient.NextEvent()
		eventStart := time.Now()
		if err != nil {
			errErr := invocationClient.ExitError("NextEventError.Main", err)
			if errErr != nil {
				log.Println(errErr)
			}
			log.Fatal(err)
		}
		timeout := time.NewTimer(time.Duration(event.DeadlineMs) * time.Microsecond)

		counter++

		if event.EventType == api.Shutdown {
			break
		}

		invokedFunctionARN = event.InvokedFunctionARN

		batch.AddInvocation(event.RequestID, eventStart)

		// Await agent telemetry. This may time out.
		log.Printf(
			"Event %v started at %v times out in %.3fms",
			event.RequestID,
			eventStart,
			float64(event.DeadlineMs)/1000,
		)
		// Race the timeout against the telemetry channel
		select {
		case telemetryBytes := <-telemetryChan:
			// We received telemetry
			inv := batch.AddTelemetry(event.RequestID, telemetryBytes)
			if inv == nil {
				log.Printf("Failed to add telemetry for request %v", event.RequestID)
			}
			// Tear down the timer
			if !timeout.Stop() {
				<-timeout.C
			}

			pollLogServer(logServer, batch)

			harvested := batch.Harvest(time.Now())
			shipHarvest(harvested, telemetryClient, invokedFunctionARN)
		case <-timeout.C:
			// Function is timing out
		}
	}
	log.Printf("New Relic Extension shutting down after %v events\n", counter)

	err := logServer.Close()
	if err != nil {
		log.Println("Error shutting down Log API server", err)
	}

	pollLogServer(logServer, batch)
	finalHarvest := batch.Close()
	shipHarvest(finalHarvest, telemetryClient, invokedFunctionARN)
}

// pollLogServer polls for platform logs, and annotates telemetry
func pollLogServer(logServer *logserver.LogServer, batch *telemetry.Batch) {
	for _, platformLog := range logServer.PollPlatformChannel() {
		inv := batch.AddTelemetry(platformLog.RequestID, platformLog.Content)
		if inv == nil {
			log.Printf("Failed to add platform log for request %v", platformLog.RequestID)
		}
	}
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
