/**

Notes:
- 	Because of the asynchronous nature of the system, it is possible that telemetry for one invoke will be
	processed during the next invoke slice. Likewise, it is possible that telemetry for the last invoke will
	be processed during the SHUTDOWN event.

*/

package main

import (
	"context"
	"newrelic-lambda-extension/AwsLambdaExtension/agentTelemetry"
	"newrelic-lambda-extension/AwsLambdaExtension/extensionApi"
	"newrelic-lambda-extension/AwsLambdaExtension/telemetryApi"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	collectAgentData bool

	l = log.WithFields(log.Fields{"pkg": "main"})
)

func main() {
	// Handle User Configured Settings
	conf := agentTelemetry.GetConfig()
	log.SetLevel(conf.LogLevel)
	collectAgentData = conf.CollectAgentData

	l.Info("[main] Starting the Telemetry API extension")
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		cancel()
		l.Info("[main] Received", s)
		l.Info("[main] Exiting")
	}()

	// Step 1 - Register the extension with Extensions API
	l.Debug("[main] Registering extension")
	extensionApiClient := extensionApi.NewClient(conf.LogLevel)
	extensionId, err := extensionApiClient.Register(ctx, conf.ExtensionName)
	if err != nil {
		l.Fatal(err)
	}
	l.Debug("[main] Registation success with extensionId", extensionId)

	// Step 2 - Start the local http listener which will receive data from Telemetry API
	l.Debug("[main] Starting the Telemetry listener")
	telemetryListener := telemetryApi.NewTelemetryApiListener()
	telemetryListenerUri, err := telemetryListener.Start()
	if err != nil {
		l.Fatal(err)
	}

	// Step 3 - Subscribe the listener to Telemetry API
	l.Debug("[main] Subscribing to the Telemetry API")
	telemetryApiClient := telemetryApi.NewClient(conf.LogLevel)
	_, err = telemetryApiClient.Subscribe(ctx, extensionId, telemetryListenerUri)
	if err != nil {
		l.Fatal(err)
	}
	l.Debug("[main] Subscription success")
	dispatcher := telemetryApi.NewDispatcher(extensionApiClient.GetFunctionName(), &conf, ctx, conf.TelemetryAPIBatchSize)

	// Optional - set 	up new relic telemetry client
	// Disable extract trace ID because it is bugged
	batch := agentTelemetry.NewBatch(conf.AgentTelemetryBatchSize, false, conf.LogLevel)
	telemetryClient := agentTelemetry.New(conf, batch, true)
	telemetryChan, err := agentTelemetry.InitTelemetryChannel()
	if err != nil {
		l.Fatalf("telemetry pipe init failed: %v", err)
	}

	l.Info("[main] New Relic Telemetry API Extension registered and subscribed for Lambda event streams succesfully")

	// Will block until invoke or shutdown event is received or cancelled via the context.
	for {
		select {
		case <-ctx.Done():
			return
		default:
			l.Debug("[main] Waiting for next event...")

			// This is a blocking action
			res, err := extensionApiClient.NextEvent(ctx)
			if err != nil {
				l.Errorf("[main] Exiting. Error: %v", err)
				return
			}

			// Fetches agent telemetry from channel then harvests only if ready
			batch.AddInvocation(res.RequestID, time.Now())
			dispatchAgentTelemetry(ctx, telemetryChan, batch, telemetryClient, res, false)
			// Dispatching log events from previous invocations
			dispatcher.Dispatch(ctx, telemetryListener.LogEventsQueue, false)

			l.Debug("[main] Received event")

			if res.EventType == extensionApi.Invoke {
				handleInvoke(res)
			} else if res.EventType == extensionApi.Shutdown {
				// Dispatch all remaining telemetry, handle shutdown
				dispatcher.Dispatch(ctx, telemetryListener.LogEventsQueue, true)

				// Close the batch and dump all remaining telemetry into harvest
				// this harvest will be forced, dumping telemtry that may not be ready
				dispatchAgentTelemetry(ctx, telemetryChan, batch, telemetryClient, res, true)
				handleShutdown(res)
				return
			}
		}
	}
}

func handleInvoke(r *extensionApi.NextEventResponse) {
	l.Info("[handleInvoke]")
}

func handleShutdown(r *extensionApi.NextEventResponse) {
	l.Info("[handleShutdown]")
}

// Collect agent data and attempt to send it if appropriate
// If force = true, collect and send data no matter what
func dispatchAgentTelemetry(ctx context.Context, telemetryChan chan []byte, batch *agentTelemetry.Batch, telemetryClient *agentTelemetry.Client, res *extensionApi.NextEventResponse, force bool) {
	if !collectAgentData {
		return
	}

	// Fetch and Batch latest agent telemetry if possible
	select {
	case telemetryBytes := <-telemetryChan:
		l.Debugf("[main] Got %d bytes of Agent Telemetry", len(telemetryBytes))
		batch.AddTelemetry(res.RequestID, telemetryBytes)
	default:
	}

	// Harvest and Send agent Data to New Relic
	if force {
		harvestAgentTelemetry(ctx, batch.Harvest(force), telemetryClient, res.InvokedFunctionArn)
	} else {
		if batch.ReadyToHarvest() {
			harvestData := batch.Harvest(false)
			harvestAgentTelemetry(ctx, harvestData, telemetryClient, res.InvokedFunctionArn)
		}
	}
}

// harvests and sends agent telemetry to New Relic
func harvestAgentTelemetry(ctx context.Context, harvested []*agentTelemetry.Invocation, telemetryClient *agentTelemetry.Client, functionARN string) {
	if len(harvested) > 0 {
		l.Debugf("[main] sending agent harvest with %d invocations", len(harvested))
		telemetrySlice := make([][]byte, 0, 2*len(harvested))
		for _, inv := range harvested {
			telemetrySlice = append(telemetrySlice, inv.Telemetry...)
		}

		numSuccessful, err := telemetryClient.SendTelemetry(ctx, functionARN, telemetrySlice)
		if err != nil {
			l.Errorf("[main] failed to send harvested telemetry for %d invocations %v", len(harvested)-numSuccessful, err)
		}
	}
}
