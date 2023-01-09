// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

package telemetryApi

import (
	"context"
	"net/http"
	"os"
	"strconv"

	"github.com/golang-collections/go-datastructures/queue"
)

type Dispatcher struct {
	httpClient   *http.Client
	licenseKey   string
	minBatchSize int64
	functionName string
}

func NewDispatcher(functionName string, ctx context.Context) *Dispatcher {
	var licenseKey string
	licenseKey = os.Getenv("NEW_RELIC_LICENSE_KEY")
        if len(licenseKey) == 0 {
	        licenseKey, _ = getNewRelicLicenseKey(ctx)
	}
	if len(licenseKey) == 0 {
		panic("NEW_RELIC_LICENSE_KEY undefined")
	}

	dispatchMinBatchSize, err := strconv.ParseInt(os.Getenv("DISPATCH_MIN_BATCH_SIZE"), 0, 16)
	if err != nil {
		dispatchMinBatchSize = 1
	}

	return &Dispatcher{
		httpClient:   &http.Client{},
		licenseKey:   licenseKey,
		minBatchSize: dispatchMinBatchSize,
		functionName: functionName,
	}

}

func (d *Dispatcher) Dispatch(ctx context.Context, logEventsQueue *queue.Queue, force bool) {
	if !logEventsQueue.Empty() && (force || logEventsQueue.Len() >= d.minBatchSize) {
		l.Info("[dispatcher:Dispatch] Dispatching", logEventsQueue.Len(), "log events")
		logEntries, _ := logEventsQueue.Get(logEventsQueue.Len())

		err := sendDataToNR(ctx, logEntries, d)
		if err != nil {
			l.Error("[dispatcher:Dispatch] Failed to dispatch, returning to queue:", err)
			for logEntry := range logEntries {
				logEventsQueue.Put(logEntry)
			}
		}
	}
}
