package agentTelemetry

import (
	"context"
	"encoding/base64"
	"newrelic-lambda-extension/config"
	"newrelic-lambda-extension/extensionApi"
	"time"
)

type AgentTelemetryDispatcher struct {
	collectData     bool
	telemetryChan   chan []byte
	batch           *Batch
	telemetryClient *Client
}

// NewDispatcher creates a new dispatcher object to manage the collection and sending of New Relic Agent data
func NewDispatcher(conf config.Config) *AgentTelemetryDispatcher {
	batch := NewBatch(conf.AgentTelemetryBatchSize, false, conf.LogLevel)
	telemetryClient := New(conf, batch, true)
	telemetryChan, err := InitTelemetryChannel()
	if err != nil {
		l.Fatalf("[agentTelemetry] agent telemetry dispatcher failed to create telemetry channel: %v", err)
	}

	l.Tracef("[agentTelemtry] client: %+v", telemetryClient)

	return &AgentTelemetryDispatcher{
		collectData:     conf.CollectAgentData,
		telemetryChan:   telemetryChan,
		telemetryClient: telemetryClient,
		batch:           batch,
	}
}

// Dispatch collects agent data and attempts to send it if appropriate
// If force = true, collect and send data no matter what
func (disp *AgentTelemetryDispatcher) Dispatch(ctx context.Context, res *extensionApi.NextEventResponse, force bool) {
	// Fetch and Batch latest agent telemetry if possible
	select {
	case telemetryBytes := <-disp.telemetryChan:
		if !disp.collectData {
			return
		}

		l.Tracef("[agentTelemetry] Agent telemetry bytes: %s", base64.URLEncoding.EncodeToString(telemetryBytes))
		if !disp.batch.HasInvocation(res.RequestID) {
			disp.batch.AddInvocation(res.RequestID, time.Now())
		}
		disp.batch.AddTelemetry(res.RequestID, telemetryBytes)
	default:
		if !disp.collectData {
			return
		}
	}

	// Harvest and Send agent Data to New Relic
	if force {
		harvestAgentTelemetry(ctx, disp.batch.Harvest(force), disp.telemetryClient, res.InvokedFunctionArn)
	} else {
		if disp.batch.ReadyToHarvest() {
			harvestData := disp.batch.Harvest(false)
			harvestAgentTelemetry(ctx, harvestData, disp.telemetryClient, res.InvokedFunctionArn)
		}
	}
}

// harvests and sends agent telemetry to New Relic
func harvestAgentTelemetry(ctx context.Context, harvested []*Invocation, telemetryClient *Client, functionARN string) {
	if len(harvested) > 0 {
		l.Debugf("[agentTelemetry] sending agent harvest with %d invocations", len(harvested))
		telemetrySlice := make([][]byte, 0, 2*len(harvested))
		for _, inv := range harvested {
			telemetrySlice = append(telemetrySlice, inv.Telemetry...)
		}

		numSuccessful, err := telemetryClient.SendTelemetry(ctx, functionARN, telemetrySlice)
		if err != nil {
			l.Errorf("[agentTelemetry] failed to send harvested telemetry for %d invocations %v", len(harvested)-numSuccessful, err)
		}
	}
}
