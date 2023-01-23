package telemetryApi

import (
	"context"
	"net/http"
	"newrelic-lambda-extension/AwsLambdaExtension/agentTelemetry"
	"os"

	"github.com/golang-collections/go-datastructures/queue"
)

type Dispatcher struct {
	httpClient   *http.Client
	licenseKey   string
	accountID    string
	minBatchSize int64
	functionName string
}

func NewDispatcher(functionName string, config *agentTelemetry.Config, ctx context.Context, batchSize int64) *Dispatcher {
	var licenseKey string
	licenseKey = os.Getenv("NEW_RELIC_LICENSE_KEY")
	if len(licenseKey) == 0 {
		licenseKey, _ = getNewRelicLicenseKey(ctx)
	}
	if len(licenseKey) == 0 {
		l.Fatal("NEW_RELIC_LICENSE_KEY undefined")
	}

	config.LicenseKey = licenseKey
	config.ExtensionName = functionName

	return &Dispatcher{
		httpClient:   &http.Client{},
		licenseKey:   licenseKey,
		minBatchSize: batchSize,
		accountID:    config.AccountID,
		functionName: functionName,
	}

}

func (d *Dispatcher) Dispatch(ctx context.Context, logEventsQueue *queue.Queue, force bool) {
	if !logEventsQueue.Empty() && (force || logEventsQueue.Len() >= d.minBatchSize) {
		l.Debug("[dispatcher:Dispatch] Dispatching ", logEventsQueue.Len(), " log events")
		logEntries, _ := logEventsQueue.Get(logEventsQueue.Len())

		err := sendDataToNR(ctx, logEntries, d, d.accountID)
		if err != nil {
			l.Error("[dispatcher:Dispatch] Failed to dispatch, returning to queue:", err)
			for logEntry := range logEntries {
				logEventsQueue.Put(logEntry)
			}
		}
	}
}
