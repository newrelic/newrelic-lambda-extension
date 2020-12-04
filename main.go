package main

import (
	"encoding/base64"
	"fmt"
	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"
	"github.com/newrelic/newrelic-lambda-extension/util"
	"net/http"
	"sync"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/credentials"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/client"
	"github.com/newrelic/newrelic-lambda-extension/telemetry"
)

func main() {
	extensionStartup := time.Now()

	// Parse various env vars for our config
	conf := config.ConfigurationFromEnvironment()

	util.ConfigLogger(conf.LogLevel == config.DebugLogLevel)

	// Extensions must register
	registrationClient := client.New(http.Client{})
	regReq := api.RegistrationRequest{
		Events: []api.LifecycleEvent{api.Invoke, api.Shutdown},
	}

	invocationClient, registrationResponse, err := registrationClient.Register(regReq)
	if err != nil {
		util.Fatal(err)
	}

	if !conf.ExtensionEnabled {
		util.Logln("Extension telemetry processing disabled")
		noopLoop(invocationClient)
		return
	}

	// Attempt to find the license key for telemetry sending
	licenseKey, err := credentials.GetNewRelicLicenseKey(&conf)
	if err != nil {
		util.Logln("Failed to retrieve license key", err)
		// We fail open; telemetry will go to CloudWatch instead
		noopLoop(invocationClient)
		return
	}

	// Set up the telemetry buffer
	batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis))

	// Start the Logs API server, and register it
	logServer, err := logserver.Start()
	if err != nil {
		util.Logln("Failed to start logs HTTP server", err)
		err = invocationClient.InitError("logServer.start", err)
		if err != nil {
			util.Fatal(err)
		}
		return
	}
	eventTypes := []api.LogEventType{api.Platform}
	if conf.SendFunctionLogs {
		eventTypes = append(eventTypes, api.Function)
	}
	subscriptionRequest := api.DefaultLogSubscription(eventTypes, logServer.Port())
	err = invocationClient.LogRegister(&subscriptionRequest)
	if err != nil {
		util.Logln("Failed to register with Logs API", err)
		err = invocationClient.InitError("logServer.register", err)
		if err != nil {
			util.Fatal(err)
		}
		return
	}

	// Init the telemetry sending client
	telemetryClient := telemetry.New(registrationResponse.FunctionName, *licenseKey, conf.TelemetryEndpoint, conf.LogEndpoint)

	telemetryChan, err := telemetry.InitTelemetryChannel()
	if err != nil {
		util.Fatal("telemetry pipe init failed: ", err)
	}

	// Send function logs as they arrive. When disabled, function logs aren't delivered to the extension.
	var backgroundTasks sync.WaitGroup
	go func() {
		backgroundTasks.Add(1)
		defer backgroundTasks.Done()
		functionLogShipLoop(logServer, telemetryClient)
	}()

	// Call next, and process telemetry, until we're shut down
	mainLoop(invocationClient, &batch, telemetryChan, logServer, telemetryClient)

	util.Debugln("Waiting for background tasks to complete")
	backgroundTasks.Wait()

	shutdownAt := time.Now()
	ranFor := shutdownAt.Sub(extensionStartup)
	util.Logf("Extension shutdown after %vms", ranFor.Milliseconds())
}

// functionLogShipLoop ships function logs to New Relic as they arrive.
func functionLogShipLoop(logServer *logserver.LogServer, telemetryClient *telemetry.Client) {
	for {
		functionLogs, more := logServer.AwaitFunctionLogs()
		if !more {
			return
		}
		err := telemetryClient.SendFunctionLogs(functionLogs)
		if err != nil {
			util.Logf("Failed to send %d function logs", len(functionLogs))
		}
	}
}

// mainLoop repeatedly calls the /next api, and processes telemetry and platform logs. The timing is rather complicated.
func mainLoop(invocationClient *client.InvocationClient, batch *telemetry.Batch, telemetryChan chan []byte, logServer *logserver.LogServer, telemetryClient *telemetry.Client) {
	counter := 0
	var invokedFunctionARN string
	var lastRequestId string
	var lastEventStart time.Time
	probablyTimeout := false
	for {
		// Our call to next blocks. It is likely that the container is frozen immediately after we call NextEvent.
		event, err := invocationClient.NextEvent()
		// We've thawed.
		eventStart := time.Now()
		if err != nil {
			errErr := invocationClient.ExitError("NextEventError.Main", err)
			if errErr != nil {
				util.Logln(errErr)
			}
			util.Fatal(err)
		}

		counter++

		if probablyTimeout {
			// We suspect a timeout. Either way, we've gotten to the next event, so telemetry will
			// have arrived for the last request if it's going to. Non-blocking poll for telemetry.
			// If we have indeed timed out, there's a chance we got telemetry out anyway. If we haven't
			// timed out, this will catch us up to the current state of telemetry, allowing us to resume.
			select {
			case telemetryBytes := <-telemetryChan:
				// We received telemetry
				batch.AddTelemetry(lastRequestId, telemetryBytes)
				util.Logf("We suspected a timeout for request %s but got telemetry anyway", lastRequestId)
			default:
			}
		}

		if event.EventType == api.Shutdown {
			if event.ShutdownReason == api.Timeout && lastRequestId != "" {
				// Synthesize the timeout error message that the platform produces, and LLC parses
				timestamp := eventStart.UTC()
				timeoutSecs := eventStart.Sub(lastEventStart).Seconds()
				timeoutMessage := fmt.Sprintf(
					"%s %s Task timed out after %.2f seconds",
					timestamp.Format(time.RFC3339),
					lastRequestId,
					timeoutSecs,
				)
				batch.AddTelemetry(lastRequestId, []byte(timeoutMessage))
			} else if event.ShutdownReason == api.Failure && lastRequestId != "" {
				// Synthesize a generic platform error. Probably an OOM, though it could be any runtime crash.
				errorMessage := fmt.Sprintf("RequestId: %s A platform error caused a shutdown", lastRequestId)
				batch.AddTelemetry(lastRequestId, []byte(errorMessage))
			}

			break
		}

		invokedFunctionARN = event.InvokedFunctionARN
		lastRequestId = event.RequestID
		// Create an invocation record to hold telemetry
		batch.AddInvocation(lastRequestId, eventStart)

		// Await agent telemetry. This may time out, so we race the timeout against the telemetry channel
		// timeoutInstant is when the invocation will time out
		timeoutInstant := time.Unix(0, event.DeadlineMs*int64(time.Millisecond))

		// Set the timeout timer for a smidge before the actual timeout; we can recover from early.
		timeout := time.NewTimer(timeoutInstant.Sub(time.Now()) - time.Millisecond)
		select {
		case telemetryBytes := <-telemetryChan:
			// We received telemetry
			util.Debugf("Agent telemetry bytes: %s", base64.URLEncoding.EncodeToString(telemetryBytes))
			inv := batch.AddTelemetry(lastRequestId, telemetryBytes)
			if inv == nil {
				util.Logf("Failed to add telemetry for request %v", lastRequestId)
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
			util.Debugln("Timeout suspected")
			probablyTimeout = true
		}
		lastEventStart = eventStart
	}
	util.Logf("New Relic Extension shutting down after %v events\n", counter)

	err := logServer.Close()
	if err != nil {
		util.Logln("Error shutting down Log API server", err)
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
			util.Debugf("Failed to add platform log for request %v", platformLog.RequestID)
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
			util.Logf("Failed to send harvested telemetry for %d invocations %s", len(harvested), err)
		}
	}
}

func noopLoop(invocationClient *client.InvocationClient) {
	for {
		event, err := invocationClient.NextEvent()
		if err != nil {
			errErr := invocationClient.ExitError("NextEventError.Noop", err)
			if errErr != nil {
				util.Logln(errErr)
			}
			util.Fatal(err)
		}

		if event.EventType == api.Shutdown {
			return
		}
	}
}
