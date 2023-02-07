package telemetryApi

import (
	"context"
	"net/http"
	"os"

	"github.com/golang-collections/go-datastructures/queue"

	"newrelic-lambda-extension/config"
	"newrelic-lambda-extension/extensionApi"
	"newrelic-lambda-extension/util"
)

type Dispatcher struct {
	httpClient   *http.Client
	compressTool *util.CompressTool
	licenseKey   string
	accountID    string
	arn          string
	functionName string
	minBatchSize int64
}

func GetNewRelicLicenseKey(ctx context.Context) string {
	licenseKey := os.Getenv("NEW_RELIC_LICENSE_KEY")

	var err error
	if len(licenseKey) == 0 {
		licenseKey, err = getNewRelicLicenseKey(ctx)
		if err != nil {
			l.Fatalf("failed to get New Relic license key: %v", err)
		}
	}
	if len(licenseKey) == 0 {
		l.Fatal("NEW_RELIC_LICENSE_KEY undefined or unavailable")
	}

	return licenseKey
}

func NewDispatcher(config *config.Config, ctx context.Context, batchSize int64) *Dispatcher {
	disp := &Dispatcher{
		httpClient:   &http.Client{},
		licenseKey:   config.LicenseKey,
		minBatchSize: batchSize,
		accountID:    config.AccountID,
		functionName: config.ExtensionName,
		compressTool: util.NewCompressTool(),
	}

	l.Tracef("Dispatcher: %+v", disp)
	return disp
}

func (d *Dispatcher) Dispatch(ctx context.Context, logEventsQueue *queue.Queue, lambdaEvent *extensionApi.NextEventResponse, force bool) {
	if !logEventsQueue.Empty() && (force || logEventsQueue.Len() >= d.minBatchSize) {
		l.Debug("[dispatcher:Dispatch] Dispatching ", logEventsQueue.Len(), " log events")
		logEntries, _ := logEventsQueue.Get(logEventsQueue.Len())

		if lambdaEvent.InvokedFunctionArn != "" && lambdaEvent.InvokedFunctionArn != d.arn {
			if len(lambdaEvent.InvokedFunctionArn) > MaxAttributeValueLen {
				d.arn = lambdaEvent.InvokedFunctionArn[:MaxAttributeValueLen]
			} else {
				d.arn = lambdaEvent.InvokedFunctionArn
			}
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
