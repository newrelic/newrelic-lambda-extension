package telemetry

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testTelemetry       = "test_telemetry"
	moreTestTelemetry   = "more_test_telemetry"
	testRequestId       = "test_a"
	testRequestId2      = "test_b"
	testRequestId3      = "test_c"
	testNoSuchRequestId = "test_z"
	ripe                = 1000
	rot                 = 10000
)

var (
	requestStart = time.Unix(1603821157, 0)
)

func generateNLengthTelemetryString(length int) string {
	outStr := ""
	for i := 0; i < length; i++ {
		outStr += "a"
	}

	return outStr
}

func TestMissingInvocation(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	invocation := batch.AddTelemetry(testNoSuchRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.Nil(t, invocation)
}

func TestEmptyHarvest(t *testing.T) {
	batch := NewBatch(ripe, rot, false)
	res := batch.Harvest(requestStart)

	assert.Nil(t, res)
}

func TestEmptyRotHarvest(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	batch.AddInvocation("test", requestStart)

	res := batch.Harvest(requestStart)

	assert.Empty(t, res)
}

func TestEmptyRipeHarvest(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	batch.lastHarvest = requestStart.Add(-ripe)
	batch.AddInvocation("test", requestStart)

	res := batch.Harvest(requestStart)

	assert.Empty(t, res)
}

func TestWithInvocationRipeHarvest(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	batch.lastHarvest = requestStart

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))
	batch.AddInvocation(testRequestId3, requestStart.Add(200*time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, []byte(testTelemetry))
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes())

	harvested := batch.Harvest(requestStart.Add(ripe*time.Millisecond + time.Millisecond))
	assert.Equal(t, 1, len(harvested))
	assert.Equal(t, testRequestId, harvested[0].RequestId)
	assert.Equal(t, 2, len(harvested[0].Telemetry))
}

func TestWithInvocationAggressiveHarvest(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))
	batch.AddInvocation(testRequestId3, requestStart.Add(200*time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes())

	harvested := batch.Harvest(requestStart.Add(ripe*time.Millisecond + time.Millisecond))
	assert.Equal(t, 2, len(harvested))
}

func TestBatch_Close(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))
	batch.AddInvocation(testRequestId3, requestStart.Add(200*time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes())

	harvested := batch.Close()
	assert.Equal(t, 2, len(harvested))
}

func TestBatchAsync(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	batch.lastHarvest = requestStart

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		batch.AddInvocation(testRequestId, requestStart)
		wg.Done()
	}()
	go func() {
		batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))
		wg.Done()
	}()
	go func() {
		batch.AddInvocation(testRequestId3, requestStart.Add(200*time.Millisecond))
		wg.Done()
	}()

	// Doing this to try to trigger a panic
	go batch.RetrieveTraceID(testRequestId)

	wg.Wait()

	var invocation, invocation2 *Invocation
	wg.Add(2)

	go func() {
		invocation = batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
		wg.Done()
	}()
	go func() {
		invocation2 = batch.AddTelemetry(testRequestId, bytes.NewBufferString(moreTestTelemetry).Bytes())
		wg.Done()
	}()

	// Doing this to try to trigger a panic
	go batch.RetrieveTraceID(testRequestId)

	wg.Wait()
	assert.NotNil(t, invocation)
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes())

	harvested := batch.Harvest(requestStart.Add(ripe*time.Millisecond + time.Millisecond))
	go assert.Equal(t, 1, len(harvested))
	go assert.Equal(t, testRequestId, harvested[0].RequestId)
	go assert.Equal(t, 2, len(harvested[0].Telemetry))
}

func TestBatch_RetrieveTraceID(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	// Add a trace ID to the batch
	requestId := "testRequestId"
	expectedTraceID := "testTraceID"
	storeTraceID.SetTraceIDValue(requestId, expectedTraceID)

	// Retrieve the trace ID
	traceID := batch.RetrieveTraceID(requestId)
	assert.Equal(t, expectedTraceID, traceID)
	storeTraceID.SetTraceIDValue(requestId, "")
	traceID = batch.RetrieveTraceID(requestId)
	assert.Empty(t, traceID)
	// Test for a non-existent request ID
	nonExistentRequestId := "nonExistentRequestId"
	traceID = batch.RetrieveTraceID(nonExistentRequestId)
	assert.Equal(t, "", traceID)
}
