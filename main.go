package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/checks"
	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"
	"github.com/newrelic/newrelic-lambda-extension/util"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/credentials"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/client"
	"github.com/newrelic/newrelic-lambda-extension/telemetry"
)

var rootCtx context.Context

func init() {
	rootCtx = context.Background()
}

func main() {
	extensionStartup := time.Now()

	ctx, cancel := context.WithCancel(rootCtx)
	defer cancel()

	// exit cleanly on SIGTERM or SIGINT
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		cancel()
		util.Logf("Received %v Exiting", s)
	}()

	// Allow extension to be interrupted with CTRL-C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			cancel()
			util.Fatal("Exiting...")
		}
	}()

	// Parse various env vars for our config
	conf := config.ConfigurationFromEnvironment()

	// Optionally enable debug logging, disabled by default
	util.ConfigLogger(conf.LogLevel == config.DebugLogLevel)

	// Extensions must register
	registrationClient := client.New(http.Client{})

	regReq := api.RegistrationRequest{
		Events: []api.LifecycleEvent{api.Invoke, api.Shutdown},
	}

	invocationClient, registrationResponse, err := registrationClient.Register(ctx, regReq)
	if err != nil {
		util.Panic(err)
	}

	// If extension disabled, go into no op mode
	if !conf.ExtensionEnabled {
		util.Logln("Extension telemetry processing disabled")
		noopLoop(ctx, invocationClient)
		return
	}

	// Attempt to find the license key for telemetry sending
	licenseKey, err := credentials.GetNewRelicLicenseKey(ctx, conf)
	if err != nil {
		util.Logln("Failed to retrieve New Relic license key", err)
		// We fail open; telemetry will go to CloudWatch instead
		noopLoop(ctx, invocationClient)
		return
	}

	// Set up the telemetry buffer
	batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis))

	// Start the Logs API server, and register it
	logServer, err := logserver.Start(conf)
	if err != nil {
		err2 := invocationClient.InitError(ctx, "logServer.start", err)
		if err2 != nil {
			util.Logln(err2)
		}
		util.Panic("Failed to start logs HTTP server", err)
	}

	eventTypes := []api.LogEventType{api.Platform}
	if conf.SendFunctionLogs {
		eventTypes = append(eventTypes, api.Function)
	}
	subscriptionRequest := api.DefaultLogSubscription(eventTypes, logServer.Port())
	err = invocationClient.LogRegister(ctx, subscriptionRequest)
	if err != nil {
		err2 := invocationClient.InitError(ctx, "logServer.register", err)
		if err2 != nil {
			util.Logln(err2)
		}
		util.Panic("Failed to register with Logs API", err)
	}

	// Init the telemetry sending client
	telemetryClient := telemetry.New(registrationResponse.FunctionName, licenseKey, conf.TelemetryEndpoint, conf.LogEndpoint)
	telemetryChan, err := telemetry.InitTelemetryChannel()
	if err != nil {
		err2 := invocationClient.InitError(ctx, "telemetryClient.init", err)
		if err2 != nil {
			util.Logln(err2)
		}
		util.Panic("telemetry pipe init failed: ", err)
	}

	// Run startup checks
	go func() {
		checks.RunChecks(ctx, conf, registrationResponse, telemetryClient)
	}()

	// Send function logs as they arrive. When disabled, function logs aren't delivered to the extension.
	backgroundTasks := &sync.WaitGroup{}
	backgroundTasks.Add(1)

	go func() {
		defer backgroundTasks.Done()
		logShipLoop(ctx, logServer, telemetryClient)
	}()

	// Call next, and process telemetry, until we're shut down
	eventCounter, invokedFunctionARN := mainLoop(ctx, invocationClient, batch, telemetryChan, logServer, telemetryClient)

	util.Logf("New Relic Extension shutting down after %v events\n", eventCounter)

	err = logServer.Close()
	if err != nil {
		util.Logln("Error shutting down Log API server", err)
	}

	pollLogServer(logServer, batch)
	finalHarvest := batch.Close()
	shipHarvest(ctx, finalHarvest, telemetryClient, invokedFunctionARN)

	util.Debugln("Waiting for background tasks to complete")
	backgroundTasks.Wait()

	shutdownAt := time.Now()
	ranFor := shutdownAt.Sub(extensionStartup)
	util.Logf("Extension shutdown after %vms", ranFor.Milliseconds())
}

// logShipLoop ships function logs to New Relic as they arrive.
func logShipLoop(ctx context.Context, logServer *logserver.LogServer, telemetryClient *telemetry.Client) {
	for {
		functionLogs, more := logServer.AwaitFunctionLogs()
		if !more {
			return
		}

		err := telemetryClient.SendFunctionLogs(ctx, functionLogs)
		if err != nil {
			util.Logf("Failed to send %d function logs", len(functionLogs))
		}
	}
}

// mainLoop repeatedly calls the /next api, and processes telemetry and platform logs. The timing is rather complicated.
func mainLoop(ctx context.Context, invocationClient *client.InvocationClient, batch *telemetry.Batch, telemetryChan chan []byte, logServer *logserver.LogServer, telemetryClient *telemetry.Client) (int, string) {
	var (
		invokedFunctionARN string
		lastEventStart     time.Time
		lastRequestId      string
	)

	eventCounter := 0
	probablyTimeout := false

	for {
		select {
		case <-ctx.Done():
			// We're already done
			util.Logln(ctx.Err())
			return eventCounter, ""
		default:
			// Our call to next blocks. It is likely that the container is frozen immediately after we call NextEvent.
			event, err := invocationClient.NextEvent(ctx)

			// We've thawed.
			eventStart := time.Now()

			if err != nil {

				util.Logln(err)
				err = invocationClient.ExitError(ctx, "NextEventError.Main", err)
				if err != nil {
					util.Logln(err)
				}
				continue
			}

			eventCounter++

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

			invokedFunctionARN = event.InvokedFunctionARN
			lastRequestId = event.RequestID

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

				return eventCounter, invokedFunctionARN
			}

			// Create an invocation record to hold telemetry
			batch.AddInvocation(lastRequestId, eventStart)

			// Await agent telemetry. This may time out
			// timeoutInstant is when the invocation will time out
			timeoutInstant := time.Unix(0, event.DeadlineMs*int64(time.Millisecond))

			// Set the timeout timer for a smidge before the actual timeout;
			// we can recover from early.
			timeoutWatchBegins := time.Millisecond * 100
			timeout := timeoutInstant.Sub(time.Now()) - timeoutWatchBegins

			invCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			select {
			case <-invCtx.Done():
				// We are about to timeout
				util.Debugln("Timeout suspected: ", invCtx.Err())
				probablyTimeout = true
				continue
			case telemetryBytes := <-telemetryChan:
				// We received telemetry
				util.Debugf("Agent telemetry bytes: %s", base64.URLEncoding.EncodeToString(telemetryBytes))
				inv := batch.AddTelemetry(lastRequestId, telemetryBytes)
				if inv == nil {
					util.Logf("Failed to add telemetry for request %v", lastRequestId)
				}

				pollLogServer(logServer, batch)
				harvested := batch.Harvest(time.Now())
				shipHarvest(ctx, harvested, telemetryClient, invokedFunctionARN)
			}

			lastEventStart = eventStart
		}
	}
}

// pollLogServer polls for platform logs, and annotates telemetry
func pollLogServer(logServer *logserver.LogServer, batch *telemetry.Batch) {
	for _, platformLog := range logServer.PollPlatformChannel() {
		inv := batch.AddTelemetry(platformLog.RequestID, platformLog.Content)
		if inv == nil {
			util.Debugf("Skipping platform log for request %v", platformLog.RequestID)
		}
	}
}

func shipHarvest(ctx context.Context, harvested []*telemetry.Invocation, telemetryClient *telemetry.Client, invokedFunctionARN string) {
	if len(harvested) > 0 {
		telemetrySlice := make([][]byte, 0, 2*len(harvested))
		for _, inv := range harvested {
			telemetrySlice = append(telemetrySlice, inv.Telemetry...)
		}

		err, _ := telemetryClient.SendTelemetry(ctx, invokedFunctionARN, telemetrySlice)
		if err != nil {
			util.Logf("Failed to send harvested telemetry for %d invocations %s", len(harvested), err)
		}
	}
}

func noopLoop(ctx context.Context, invocationClient *client.InvocationClient) {
	util.Logln("Starting no-op mode, no telemetry will be sent")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			event, err := invocationClient.NextEvent(ctx)
			if err != nil {
				util.Logln(err)
				errErr := invocationClient.ExitError(ctx, "NextEventError.Noop", err)
				if errErr != nil {
					util.Logln(errErr)
				}
				continue
			}

			if event.EventType == api.Shutdown {
				return
			}
		}
	}
}
