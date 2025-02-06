package telemetry

import (
	"bytes"
	"encoding/base64"
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
	isAPMTelemetry      = true
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

	invocation := batch.AddTelemetry(testNoSuchRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
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

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, []byte(testTelemetry), isAPMTelemetry)
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)

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

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)

	harvested := batch.Harvest(requestStart.Add(ripe*time.Millisecond + time.Millisecond))
	assert.Equal(t, 2, len(harvested))
}

func TestBatch_Close(t *testing.T) {
	batch := NewBatch(ripe, rot, false)

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))
	batch.AddInvocation(testRequestId3, requestStart.Add(200*time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)

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
		invocation = batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
		wg.Done()
	}()
	go func() {
		invocation2 = batch.AddTelemetry(testRequestId, bytes.NewBufferString(moreTestTelemetry).Bytes(), isAPMTelemetry)
		wg.Done()
	}()

	// Doing this to try to trigger a panic
	go batch.RetrieveTraceID(testRequestId)

	wg.Wait()
	assert.NotNil(t, invocation)
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)

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
	batch.SetTraceIDValue(requestId, expectedTraceID)

	// Retrieve the trace ID
	traceID := batch.RetrieveTraceID(requestId)
	assert.Equal(t, expectedTraceID, traceID)
	batch.SetTraceIDValue(requestId, "")
	traceID = batch.RetrieveTraceID(requestId)
	assert.Empty(t, traceID)
	// Test for a non-existent request ID
	nonExistentRequestId := "nonExistentRequestId"
	traceID = batch.RetrieveTraceID(nonExistentRequestId)
	assert.Equal(t, "", traceID)
}
func TestAddTelemetry(t *testing.T) {
	batch := NewBatch(ripe, rot, true)

	batch.AddInvocation(testRequestId, requestStart)
	inv := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.NotNil(t, inv)
	assert.Equal(t, 1, len(inv.Telemetry))
	assert.Equal(t, testTelemetry, string(inv.Telemetry[0]))

	inv2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(moreTestTelemetry).Bytes(), isAPMTelemetry)
	assert.NotNil(t, inv2)
	assert.Equal(t, 2, len(inv2.Telemetry))
	assert.Equal(t, moreTestTelemetry, string(inv2.Telemetry[1]))

	assert.Equal(t, requestStart, batch.eldest)

	traceId := "testTraceId"
	encodedTelemetry := base64.StdEncoding.EncodeToString([]byte(traceId))

	inv3 := batch.AddTelemetry(testRequestId, []byte(encodedTelemetry), isAPMTelemetry)
	assert.NotNil(t, inv3)
	assert.Equal(t, "", inv3.TraceId)
	assert.Equal(t, "", batch.RetrieveTraceID(testRequestId))

	inv4 := batch.AddTelemetry(testNoSuchRequestId, bytes.NewBufferString(testTelemetry).Bytes(), isAPMTelemetry)
	assert.Nil(t, inv4)

	inv5 := batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes(), false)
	assert.Nil(t, inv5)
}
