package agentTelemetry

import (
	"bytes"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	testTelemetry       = "test_telemetry"
	moreTestTelemetry   = "more_test_telemetry"
	testRequestId       = "test_a"
	testRequestId2      = "test_b"
	testRequestId3      = "test_c"
	testNoSuchRequestId = "test_z"
)

var DefaultBatchSize = defaultAgentTelemtryBatchSize

func TestMissingInvocation(t *testing.T) {
	batch := NewBatch(DefaultBatchSize, false, log.InfoLevel)

	invocation := batch.AddTelemetry(testNoSuchRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.Nil(t, invocation)
}

func TestEmptyHarvestForce(t *testing.T) {
	batch := NewBatch(DefaultBatchSize, false, log.InfoLevel)
	res := batch.Harvest(true)

	assert.Equal(t, 0, len(res))
}

func TestEmptyHarvest(t *testing.T) {
	batch := NewBatch(DefaultBatchSize, false, log.InfoLevel)
	res := batch.Harvest(false)

	assert.Equal(t, 0, len(res))
}

func TestFullHarvest(t *testing.T) {
	batch := NewBatch(DefaultBatchSize, false, log.InfoLevel)
	requestStart := time.Now()

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))
	batch.AddInvocation(testRequestId3, requestStart.Add(200*time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(moreTestTelemetry).Bytes())
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes())

	harvested := batch.Harvest(false)
	assert.Equal(t, 2, len(harvested))
	assert.Equal(t, testRequestId, harvested[0].RequestId)
	assert.Equal(t, 2, len(harvested[0].Telemetry))
}

func TestHarvestWithTraceID(t *testing.T) {
	batch := NewBatch(DefaultBatchSize, true, log.InfoLevel)
	requestStart := time.Now()

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))
	batch.AddInvocation(testRequestId3, requestStart.Add(200*time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(moreTestTelemetry).Bytes())
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes())

	harvested := batch.Harvest(false)
	assert.Equal(t, 2, len(harvested))

	for _, harvest := range harvested {
		assert.GreaterOrEqual(t, len(harvest.Telemetry), 1)
	}

}

func TestNotFullHarvest(t *testing.T) {
	batch := NewBatch(DefaultBatchSize, false, log.InfoLevel)
	requestStart := time.Now()

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(moreTestTelemetry).Bytes())
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes())

	// This should not get harvested
	batch.AddInvocation(testRequestId3, requestStart.Add(300*time.Millisecond))

	harvested := []*Invocation{}
	if batch.ReadyToHarvest() {
		harvested = batch.Harvest(false)
	}

	assert.Equal(t, 2, len(harvested))
	assert.NotEqual(t, testRequestId3, harvested[0].RequestId)
	assert.NotEqual(t, testRequestId3, harvested[1].RequestId)
}

func TestForcedHarvest(t *testing.T) {
	batch := NewBatch(DefaultBatchSize, false, log.InfoLevel)
	requestStart := time.Now()

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(moreTestTelemetry).Bytes())
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes())

	harvested := batch.Harvest(true)
	assert.Equal(t, 2, len(harvested))
}

func TestBatch_Close(t *testing.T) {
	batch := NewBatch(DefaultBatchSize, false, log.InfoLevel)
	requestStart := time.Now()

	batch.AddInvocation(testRequestId, requestStart)
	batch.AddInvocation(testRequestId2, requestStart.Add(100*time.Millisecond))
	batch.AddInvocation(testRequestId3, requestStart.Add(200*time.Millisecond))

	invocation := batch.AddTelemetry(testRequestId, bytes.NewBufferString(testTelemetry).Bytes())
	assert.NotNil(t, invocation)

	invocation2 := batch.AddTelemetry(testRequestId, bytes.NewBufferString(moreTestTelemetry).Bytes())
	assert.Equal(t, invocation, invocation2)

	batch.AddTelemetry(testRequestId2, bytes.NewBufferString(testTelemetry).Bytes())

	harvested := batch.Close()
	assert.Equal(t, 2, len(harvested))
}
