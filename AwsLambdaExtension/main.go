// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

/**

Notes:
- 	Because of the asynchronous nature of the system, it is possible that telemetry for one invoke will be
	processed during the next invoke slice. Likewise, it is possible that telemetry for the last invoke will
	be processed during the SHUTDOWN event.

*/

package main

import (
	"context"
	"encoding/base64"
	"newrelic-lambda-extension/AwsLambdaExtension/agentTelemetry"
	"newrelic-lambda-extension/AwsLambdaExtension/extensionApi"
	"newrelic-lambda-extension/AwsLambdaExtension/telemetryApi"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

var l = log.WithFields(log.Fields{"pkg": "main"})

func main() {
	l.Info("[main] Starting the Telemetry API extension")
	conf := agentTelemetry.GetConfig()

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
	l.Info("[main] Registering extension")
	extensionApiClient := extensionApi.NewClient()
	extensionId, err := extensionApiClient.Register(ctx, conf.ExtensionName)
	if err != nil {
		l.Fatal(err)
	}
	l.Info("[main] Registation success with extensionId", extensionId)

	// Step 2 - Start the local http listener which will receive data from Telemetry API
	l.Info("[main] Starting the Telemetry listener")
	telemetryListener := telemetryApi.NewTelemetryApiListener()
	telemetryListenerUri, err := telemetryListener.Start()
	if err != nil {
		l.Fatal(err)
	}

	// Step 3 - Subscribe the listener to Telemetry API
	l.Info("[main] Subscribing to the Telemetry API")
	telemetryApiClient := telemetryApi.NewClient()
	_, err = telemetryApiClient.Subscribe(ctx, extensionId, telemetryListenerUri)
	if err != nil {
		l.Fatal(err)
	}
	l.Info("[main] Subscription success")
	dispatcher := telemetryApi.NewDispatcher(extensionApiClient.GetFunctionName(), &conf, ctx)

	// Optional - set 	up new relic telemetry client
	batch := agentTelemetry.NewBatch(agentTelemetry.DefaultBatchSize, true)
	telemetryClient := agentTelemetry.New(conf, batch, true)
	telemetryChan, err := agentTelemetry.InitTelemetryChannel()
	if err != nil {
		l.Panic("telemetry pipe init failed: ", err)
	}

	// Will block until invoke or shutdown event is received or cancelled via the context.
	for {
		select {
		case <-ctx.Done():
			return
		default:
			l.Info("[main] Waiting for next event...")

			// This is a blocking action
			res, err := extensionApiClient.NextEvent(ctx)
			if err != nil {
				l.Error("[main] Exiting. Error:", err)
				return
			}

			select {
			case telemetryBytes := <-telemetryChan:
				l.Debug("[main] Got Agent Telemetry %s", base64.URLEncoding.EncodeToString(telemetryBytes))
				batch.AddTelemetry(res.RequestID, telemetryBytes)
			default:
			}

			if batch.ReadyToHarvest() {
				harvestAgentTelemetry(ctx, batch.Harvest(false), telemetryClient, res.InvokedFunctionArn)
			}

			// Dispatching log events from previous invocations
			dispatcher.Dispatch(ctx, telemetryListener.LogEventsQueue, false)

			l.Info("[main] Received event")

			if res.EventType == extensionApi.Invoke {
				handleInvoke(res)
			} else if res.EventType == extensionApi.Shutdown {
				// Dispatch all remaining telemetry, handle shutdown
				dispatcher.Dispatch(ctx, telemetryListener.LogEventsQueue, true)

				// Close the batch and dump all remaining telemetry into harvest
				harvestAgentTelemetry(ctx, batch.Close(), telemetryClient, res.InvokedFunctionArn)

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

func harvestAgentTelemetry(ctx context.Context, harvested []*agentTelemetry.Invocation, telemetryClient *agentTelemetry.Client, functionARN string) {
	l.Debugf("[main] sending agent harvest with %d invocations", len(harvested))
	if len(harvested) > 0 {
		telemetrySlice := make([][]byte, 0, 2*len(harvested))
		for _, inv := range harvested {
			telemetrySlice = append(telemetrySlice, inv.Telemetry...)
		}

		err, numSuccessful := telemetryClient.SendTelemetry(ctx, functionARN, telemetrySlice)
		if err != nil {
			l.Infof("[main] failed to send harvested telemetry for %d invocations %v", len(harvested)-numSuccessful, err)
		}
	}
}
