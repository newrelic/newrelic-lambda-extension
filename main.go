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

	"github.com/newrelic/newrelic-lambda-extension/apm"
	"github.com/newrelic/newrelic-lambda-extension/checks"
	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"
	"github.com/newrelic/newrelic-lambda-extension/util"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/credentials"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/client"
	"github.com/newrelic/newrelic-lambda-extension/telemetry"
)

var (
	invokedFunctionARN string
	lastEventStart     time.Time
	lastRequestId      string
	rootCtx            context.Context
	LambdaFunctionName string
	LambdaAccountId    string
	LambdaFunctionVersion string
)

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
	ctrlCChan := make(chan os.Signal, 1)
	signal.Notify(ctrlCChan, os.Interrupt)
	go func() {
		for range ctrlCChan {
			cancel()
			util.Fatal("Exiting...")
		}
	}()

	// Parse various env vars for our config
	conf := config.ConfigurationFromEnvironment()

	// Optionally enable debug logging, disabled by default
	util.ConfigLogger(conf.LogsEnabled, conf.LogLevel == config.DebugLogLevel)

	util.Logf("Initializing version %s of the New Relic Lambda Extension...", util.Version)

	// Extensions must register
	registrationClient := client.New(http.Client{})

	regReq := api.RegistrationRequest{
		Events: []api.LifecycleEvent{api.Invoke, api.Shutdown},
	}

	invocationClient, registrationResponse, err := registrationClient.Register(ctx, regReq)
	LambdaFunctionName = registrationResponse.FunctionName
	LambdaAccountId = registrationResponse.AccountId
	LambdaFunctionVersion = registrationResponse.FunctionVersion
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
	var timeout = 1 * time.Second
	ctxLicenseKey, cancelLicenseKey := context.WithTimeout(ctx, timeout)
	defer cancelLicenseKey()
	licenseKey, err := credentials.GetNewRelicLicenseKey(ctxLicenseKey, conf)
	if err != nil {
		util.Logln("Failed to retrieve New Relic license key", err)
		// We fail open; telemetry will go to CloudWatch instead
		noopLoop(ctx, invocationClient)
		return
	}

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
	if conf.SendExtensionLogs {
		eventTypes = append(eventTypes, api.Extension)
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
	telemetryChan, err := telemetry.InitTelemetryChannel()
	if err != nil {
		err2 := invocationClient.InitError(ctx, "telemetryClient.init", err)
		if err2 != nil {
			util.Logln(err2)
		}
		util.Panic("telemetry pipe init failed: ", err)
	}
	// Set up the telemetry buffer
	batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis), conf.CollectTraceID)
	// In APM Lambda mode, we don't send telemetry
	telemetryClient := telemetry.New(registrationResponse.FunctionName, licenseKey, conf.TelemetryEndpoint, conf.LogEndpoint, batch, conf.CollectTraceID, conf.ClientTimeout)
	

	// Run startup checks
	go func() {
		if conf.IgnoreExtensionChecks["all"] && conf.APMLambdaMode{
			// Ignore extension checks in APM Mode
			util.Debugf("Ignoring all extension checks")
			return
		}
		checks.RunChecks(ctx, conf, registrationResponse, telemetryClient)
	}()

	// Send function logs as they arrive. When disabled, function logs aren't delivered to the extension.
	backgroundTasks := &sync.WaitGroup{}
	backgroundTasks.Add(1)

	go func() {
		defer backgroundTasks.Done()
		logShipLoop(ctx, logServer, telemetryClient, conf.APMLambdaMode)
	}()

	// Call next, and process telemetry, until we're shut down
	eventCounter := mainLoop(ctx, invocationClient, batch, telemetryChan, logServer, conf, telemetryClient)

	util.Logf("New Relic Extension shutting down after %v events\n", eventCounter)
	if conf.APMLambdaMode {
		pollLogAPMServer(logServer, conf)
	} else {
		pollLogServer(logServer, batch)
	}
	err = logServer.Close()
	if err != nil {
		util.Logln("Error shutting down Log API server", err)
	}
	if !conf.APMLambdaMode {
		finalHarvest := batch.Close()
		shipHarvest(ctx, finalHarvest, telemetryClient)
	}
	util.Debugln("Waiting for background tasks to complete")
	backgroundTasks.Wait()

	shutdownAt := time.Now()
	ranFor := shutdownAt.Sub(extensionStartup)
	util.Logf("Extension shutdown after %vms", ranFor.Milliseconds())
}

// logShipLoop ships function logs to New Relic as they arrive.
func logShipLoop(ctx context.Context, logServer *logserver.LogServer, telemetryClient *telemetry.Client, isAPMLambdaMode bool) {
	for {
		functionLogs, more := logServer.AwaitFunctionLogs()
		if !more {
			return
		}
		err := telemetryClient.SendFunctionLogs(ctx, invokedFunctionARN, functionLogs, isAPMLambdaMode)
		if err != nil {
			util.Logf("Failed to send %d function logs", len(functionLogs))
		}
	}
}

// mainLoop repeatedly calls the /next api, and processes telemetry and platform logs. The timing is rather complicated.
func mainLoop(ctx context.Context, invocationClient *client.InvocationClient, batch *telemetry.Batch, telemetryChan chan []byte, logServer *logserver.LogServer, conf *config.Configuration, telemetryClient *telemetry.Client) int {
	eventCounter := 0
	probablyTimeout := false
	apmCmd := apm.RpmCmd{}
	apmControls := &apm.RpmControls{}
	for {
		select {
		case <-ctx.Done():
			// We're already done
			return eventCounter
		default:
			// Our call to next blocks. It is likely that the container is frozen immediately after we call NextEvent.
			util.Debugln("mainLoop: waiting for next lambda invocation event...")
			event, err := invocationClient.NextEvent(ctx)
			if conf.APMLambdaMode {
				apm.Once.Do(func() {
					apmCmd, apmControls, err = apm.NewAPMClient(conf, LambdaFunctionName, LambdaAccountId, LambdaFunctionVersion)
					if err != nil {
						util.Logln("mainLoop: failed to initialize APM client:", err)
					}
				})
			}
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
					if conf.APMLambdaMode {
						shipAPMHarvest(ctx, telemetryBytes, conf, apmCmd, apmControls)
					} else {
						// We received telemetry
						util.Debugf("Agent telemetry bytes: %s", base64.URLEncoding.EncodeToString(telemetryBytes))
						batch.AddTelemetry(lastRequestId, telemetryBytes)
						util.Logf("We suspected a timeout for request %s but got telemetry anyway", lastRequestId)
					}
				default:
				}
			}

			if event.EventType == api.Shutdown {
				if conf.APMLambdaMode {
					if event.ShutdownReason == api.Timeout {
						timeoutSecs := eventStart.Sub(lastEventStart).Seconds()
						errorMessage := fmt.Sprintf(
							"Task timed out after %.2f seconds",
							timeoutSecs,
						)
						apm.SendErrorEvent(apmCmd, apmControls, []interface{}{"Lambda.Timedout", 
																			fmt.Sprintf("%f", timeoutSecs), 
																			lastRequestId, 
																			errorMessage,
																			LambdaFunctionName, 
																			LambdaAccountId, 
																			LambdaFunctionVersion})
					} else if event.ShutdownReason == api.Failure {
						errorMessage := fmt.Sprintf("RequestId: %s AWS Lambda platform fault caused a shutdown", lastRequestId)
						timeoutSecs := eventStart.Sub(lastEventStart).Seconds()
						apm.SendErrorEvent(apmCmd, apmControls, []interface{}{"Lambda.PlatformFault", 
																			fmt.Sprintf("%f", timeoutSecs), 
																			lastRequestId, 
																			errorMessage,
																			LambdaFunctionName, 
																			LambdaAccountId, 
																			LambdaFunctionVersion})
					}
				} else {
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
						errorMessage := fmt.Sprintf("RequestId: %s AWS Lambda platform fault caused a shutdown", lastRequestId)
						batch.AddTelemetry(lastRequestId, []byte(errorMessage))
					}
				}

				return eventCounter
			} else {
				// Reset probablyTimeout if the event after the suspected timeout wasn't a timeout shutdown.
				probablyTimeout = false
			}
			if !conf.APMLambdaMode {
				// Note: shutdown events do not have these properties; we now know this is an invocation event.
				invokedFunctionARN = event.InvokedFunctionARN
				lastRequestId = event.RequestID

				// Create an invocation record to hold telemetry
				batch.AddInvocation(lastRequestId, eventStart)

				// Await agent telemetry, which may time out.
			}

			// timeoutInstant is when the invocation will time out
			timeoutInstant := time.Unix(0, event.DeadlineMs*int64(time.Millisecond))

			// Set the timeout timer for a smidge before the actual timeout; we can recover from false timeouts.
			timeoutWatchBegins := 200 * time.Millisecond
			timeLimitContext, timeLimitCancel := context.WithDeadline(ctx, timeoutInstant.Add(-timeoutWatchBegins))
			if conf.APMLambdaMode {
				pollLogAPMServer(logServer, conf)
			} else {
				// Before we begin to await telemetry, harvest and ship. Ripe telemetry will mostly be handled here. Even that is a
				// minority of invocations. Putting this here lets us run the HTTP request to send to NR in parallel with the Lambda
				// handler, reducing or eliminating our latency impact.
				pollLogServer(logServer, batch)
				shipHarvest(ctx, batch.Harvest(time.Now()), telemetryClient)
			}

			select {
			case <-timeLimitContext.Done():
				timeLimitCancel()

				// We are about to timeout
				probablyTimeout = true
				continue
			case telemetryBytes := <-telemetryChan:
				timeLimitCancel()
				if conf.APMLambdaMode {
					pollLogAPMServer(logServer, conf)
					shipAPMHarvest(ctx, telemetryBytes, conf, apmCmd, apmControls)
				} else {
					// We received telemetry
					util.Debugf("Agent telemetry bytes: %s", base64.URLEncoding.EncodeToString(telemetryBytes))
					inv := batch.AddTelemetry(lastRequestId, telemetryBytes)
					if inv == nil {
						util.Logf("Failed to add telemetry for request %v", lastRequestId)
					}
					// Opportunity for an aggressive harvest, in which case, we definitely want to wait for the HTTP POST
					// to complete. Mostly, nothing really happens here.
					pollLogServer(logServer, batch)
					shipHarvest(ctx, batch.Harvest(time.Now()), telemetryClient)
				}
				
			}

			lastEventStart = eventStart
		}
	}
}

// pollLogAPMServer polls for platform logs, and send as APM telemetry
func pollLogAPMServer(logServer *logserver.LogServer, conf *config.Configuration) {
	<-apm.ConnectDone
	entityGuid := apm.GetEntityGuid()
	for _, platformLog := range logServer.PollPlatformChannel() {
		lambdaMetrics, _ := apm.ParseLambdaReportLog(string(platformLog.Content))
		metrics := lambdaMetrics.ConvertToMetrics("apm.lambda.transaction", entityGuid, LambdaFunctionName)
		statusCode, responseBody, err := apm.SendMetrics(conf.LicenseKey, conf.MetricEndpoint, metrics, true)
		if err != nil {
			util.Logf("Error sending metric: %v", err)
		}
		util.Logf("Response Status: %d\n", statusCode)
		util.Logf("Response Body: %s\n", responseBody)
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

func shipAPMHarvest(ctx context.Context, payload []byte, conf *config.Configuration, cmd apm.RpmCmd, cs *apm.RpmControls) {
	<-apm.ConnectDone
	apmErr, _ := apm.SendAPMTelemetry(ctx, invokedFunctionARN, payload, conf, cmd, cs)
	if apmErr != nil {
		util.Logf("Failed to send APM telemetry for invocations %s", apmErr)
	}
}

func shipHarvest(ctx context.Context, harvested []*telemetry.Invocation, telemetryClient *telemetry.Client) {
	if len(harvested) > 0 {
		util.Debugf("shipHarvest: harvesting agent telemetry")
		telemetrySlice := make([][]byte, 0, 2*len(harvested))
		for _, inv := range harvested {
			telemetrySlice = append(telemetrySlice, inv.Telemetry...)
		}
		util.Debugf("shipHarveset: %d telemetry payloads harvested", len(telemetrySlice))

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
