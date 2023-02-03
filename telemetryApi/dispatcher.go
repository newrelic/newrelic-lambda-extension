package telemetryApi

import (
	"context"
	"net/http"
	"os"

	"github.com/golang-collections/go-datastructures/queue"

	"newrelic-lambda-extension/config"
	"newrelic-lambda-extension/extensionApi"
)

type Dispatcher struct {
	httpClient   *http.Client
	licenseKey   string
	accountID    string
	arn          string
	functionName string
	minBatchSize int64
}

func NewDispatcher(functionName string, config *config.Config, ctx context.Context, batchSize int64) *Dispatcher {
	var licenseKey string
	var err error
	licenseKey = os.Getenv("NEW_RELIC_LICENSE_KEY")
	if len(licenseKey) == 0 {
		licenseKey, err = getNewRelicLicenseKey(ctx)
		if err != nil {
			l.Fatalf("failed to get New Relic license key: %v", err)
		}
	}
	if len(licenseKey) == 0 {
		l.Fatal("NEW_RELIC_LICENSE_KEY undefined or unavailable")
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

func (d *Dispatcher) Dispatch(ctx context.Context, logEventsQueue *queue.Queue, lambdaEvent *extensionApi.NextEventResponse, accountID string, force bool) {
	if !logEventsQueue.Empty() && (force || logEventsQueue.Len() >= d.minBatchSize) {
		l.Debug("[dispatcher:Dispatch] Dispatching ", logEventsQueue.Len(), " log events")
		logEntries, _ := logEventsQueue.Get(logEventsQueue.Len())

		if lambdaEvent.InvokedFunctionArn != "" {
			d.arn = lambdaEvent.InvokedFunctionArn
		}

		err := sendDataToNR(ctx, logEntries, d, lambdaEvent.RequestID)
		if err != nil {
			l.Error("[dispatcher:Dispatch] Failed to dispatch, returning to queue:", err)
			for logEntry := range logEntries {
				logEventsQueue.Put(logEntry)
			}
		}
	}
}
