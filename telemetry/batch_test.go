package telemetry

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	testTelemetry = "test_telemetry"
	moreTestTelemetry = "more_test_telemetry"
	testRequestId = "test_a"
	testRequestId2 = "test_b"
	testRequestId3 = "test_c"
	testNoSuchRequestId = "test_z"
	ripe = 1000
	rot = 10000
)

var (
	requestStart = time.Unix(1603821157, 0)
)

func TestMissingInvocation(t *testing.T) {
	batch := NewBatch(ripe, rot)

	invocation := batch.AddTelemetry(testNoSuchRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.Nil(t, invocation)
}

func TestWithInvocationRipeHarvest(t *testing.T) {
	batch := NewBatch(ripe, rot)

	// To test a ripe harvest, we need to have done at least one harvest cycle before
	batch.Harvest(requestStart)

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100 * time.Millisecond))
	batch.AddInvocation(testRequestId3, requestStart.Add(200 * time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(moreTestTelemetry).Bytes())
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes())

	harvested := batch.Harvest(requestStart.Add(ripe*time.Millisecond + time.Millisecond))
	assert.Equal(t, 1, len(harvested))
	assert.Equal(t, testRequestId, harvested[0].RequestId)
}

func TestWithInvocationAggressiveHarvest(t *testing.T) {
	batch := NewBatch(ripe, rot)

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100 * time.Millisecond))
	batch.AddInvocation(testRequestId3, requestStart.Add(200 * time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(moreTestTelemetry).Bytes())
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes())

	harvested := batch.Harvest(requestStart.Add(ripe*time.Millisecond + time.Millisecond))
	assert.Equal(t, 2, len(harvested))
}

func TestBatch_Close(t *testing.T) {
	batch := NewBatch(ripe, rot)

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100 * time.Millisecond))
	batch.AddInvocation(testRequestId3, requestStart.Add(200 * time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(moreTestTelemetry).Bytes())
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes())

	harvested := batch.Close()
	assert.Equal(t, 2, len(harvested))
}
