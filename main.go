package main

/*
Notes:
- 	Because of the asynchronous nature of the system, it is possible that telemetry for one invoke will be
	processed during the next invoke slice. Likewise, it is possible that telemetry for the last invoke will
	be processed during the SHUTDOWN event.
*/

import (
	"context"
	"newrelic-lambda-extension/agentTelemetry"
	"newrelic-lambda-extension/config"
	"newrelic-lambda-extension/extensionApi"
	"newrelic-lambda-extension/telemetryApi"
	"newrelic-lambda-extension/util"

	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

var (
	l = log.WithFields(log.Fields{"pkg": "main"})
)

func main() {
	// Handle User Configured Settings
	conf := config.GetConfig()
	log.SetLevel(conf.LogLevel)
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})

	l.Infof("[main] Starting the New Relic Telemetry API extension version %s", util.Version)
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		// ctrl + c escape
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
	dispatcher := telemetryApi.NewDispatcher(
		extensionApiClient.GetFunctionName(),
		&conf,
		ctx,
		conf.TelemetryAPIBatchSize,
	)

	// Set up new relic agent telemetry dispatcher
	agentDispatcher := agentTelemetry.NewDispatcher(conf)
	//	entityManager := util.NewLambdaEntityManager(conf.AccountID)

	l.Info("[main] New Relic Telemetry API Extension succesfully registered and subscribed")

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
			l.Debugf("[main] Received event %+v", res)
			//entityGUID := entityManager.GenerateGUID(res.InvokedFunctionArn)

			// Dispatching log events from previous invocations
			agentDispatcher.Dispatch(ctx, res, false)
			dispatcher.Dispatch(ctx, telemetryListener.LogEventsQueue, res, conf.AccountID, false)

			if res.EventType == extensionApi.Invoke {
				l.Debug("[handleInvoke]")
				// we no longer care about this but keep it here just in case
			} else if res.EventType == extensionApi.Shutdown {
				// force dispatch all remaining telemetry, handle shutdown
				l.Debug("[handleShutdown]")
				dispatcher.Dispatch(ctx, telemetryListener.LogEventsQueue, res, conf.AccountID, true)
				agentDispatcher.Dispatch(ctx, res, true)
				l.Info("[main] New Relic Telemetry API Extension successfully shut down")
				return
			}
		}
	}
}
