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
	postUri      string
	licenseKey   string
	minBatchSize int64
}

func NewDispatcher() *Dispatcher {
	dispatchPostUri := os.Getenv("DISPATCH_POST_URI")
	if len(dispatchPostUri) == 0 {
		panic("dispatchPostUri undefined")
	}

	licenseKey := os.Getenv("LICENSE_KEY")
	if len(licenseKey) == 0 {
		panic("licenseKey undefined")
	}

	dispatchMinBatchSize, err := strconv.ParseInt(os.Getenv("DISPATCH_MIN_BATCH_SIZE"), 0, 16)
	if err != nil {
		dispatchMinBatchSize = 1
	}

	return &Dispatcher{
		httpClient:   &http.Client{},
		postUri:      dispatchPostUri,
		licenseKey:   licenseKey,
		minBatchSize: dispatchMinBatchSize,
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
