package telemetry

import (
	"encoding/base64"
	"math"
	"sync"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

// The Unix epoch instant; used as a nil time for eldest and lastHarvest
var epochStart = time.Unix(0, 0)

type Batch struct {
	extractTraceID  bool
	lastHarvest     time.Time
	eldest          time.Time
	ripeDuration    time.Duration
	veryOldDuration time.Duration
	invocations     map[string]*Invocation
	lock            sync.RWMutex
	storeTraceID    map[string]string
}

func (b *Batch) SetTraceIDValue(key string, value string) {

	b.storeTraceID[key] = value
}

// NewBatch constructs a new batch.
func NewBatch(ripeMillis, rotMillis int64, extractTraceID bool) *Batch {
	initialSize := uint32(math.Min(float64(ripeMillis)/100, 100))
	return &Batch{
		lastHarvest:     epochStart,
		eldest:          epochStart,
		invocations:     make(map[string]*Invocation, initialSize),
		ripeDuration:    time.Duration(ripeMillis) * time.Millisecond,
		veryOldDuration: time.Duration(rotMillis) * time.Millisecond,
		extractTraceID:  extractTraceID,
		storeTraceID:    make(map[string]string, initialSize),
	}
}

// AddInvocation should be called just after the next API response. It creates the Invocation record so that we can attach telemetry later.
func (b *Batch) AddInvocation(requestId string, start time.Time) {
	b.lock.Lock()
	defer b.lock.Unlock()

	invocation := NewInvocation(requestId, start)
	b.invocations[requestId] = &invocation
}

// AddTelemetry attaches telemetry to an existing Invocation, identified by requestId
func (b *Batch) AddTelemetry(requestId string, telemetry []byte, isAPMTelemetry bool) *Invocation {
	b.lock.Lock()
	defer b.lock.Unlock()

	inv, ok := b.invocations[requestId]
	if ok {
		inv.Telemetry = append(inv.Telemetry, telemetry)
		if b.eldest.Equal(epochStart) {
			b.eldest = inv.Start
		}
		if b.extractTraceID && isAPMTelemetry {
			telemetryBytesEncoded := []byte(base64.StdEncoding.EncodeToString(telemetry))
			traceId, err := ExtractTraceID(telemetryBytesEncoded)
			if err != nil {
				util.Debugln(err)
			}

			// We don't want to unset a previously set trace ID
			if traceId != "" {
				inv.TraceId = traceId
				b.SetTraceIDValue(requestId, traceId)
			}
		}
		return inv
	}
	return nil
}

// Harvest checks to see if it's time to harvest, and returns harvested invocations, or nil. The caller must ensure that harvested invocations are sent.
func (b *Batch) Harvest(now time.Time) []*Invocation {
	b.lock.Lock()
	defer b.lock.Unlock()

	if len(b.invocations) == 0 {
		return nil
	}

	veryOldTime := now.Add(-b.veryOldDuration)
	if b.lastHarvest.Before(veryOldTime) {
		return b.aggressiveHarvest(now)
	}

	ripeTime := now.Add(-b.ripeDuration)
	if b.eldest.Before(ripeTime) {
		return b.ripeHarvest(now)
	}

	return nil
}

// Close aggressively harvests all telemetry from the Batch. The Batch is no longer valid.
func (b *Batch) Close() []*Invocation {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.aggressiveHarvest(time.Now())
}

// aggressiveHarvest harvests all invocations, ripe or not. It removes harvested invocations from the batch and updates the lastHarvest timestamp.
func (b *Batch) aggressiveHarvest(now time.Time) []*Invocation {
	ret := make([]*Invocation, 0, len(b.invocations))
	for k, v := range b.invocations {
		if !v.IsEmpty() {
			ret = append(ret, v)
			delete(b.invocations, k)
		}
	}
	if len(ret) > 0 {
		b.lastHarvest = now
		b.eldest = epochStart
	}
	util.Debugf("Aggressive harvest yielded %d invocations\n", len(ret))
	return ret
}

// ripeHarvest harvests all ripe invocations. It removes harvested invocations from the batch and updates the lastHarvest and eldest timestamps.
func (b *Batch) ripeHarvest(now time.Time) []*Invocation {
	ret := make([]*Invocation, 0, len(b.invocations))
	newEldest := epochStart
	for k, v := range b.invocations {
		if v.IsRipe() {
			ret = append(ret, v)
			delete(b.invocations, k)
		} else if newEldest.Equal(epochStart) || v.Start.Before(newEldest) {
			newEldest = v.Start
		}
	}
	b.eldest = newEldest
	if len(ret) > 0 {
		b.lastHarvest = now
	}
	util.Debugf("Ripe harvest yielded %d invocations\n", len(ret))
	return ret
}

// RetrieveTraceID looks up a trace ID using the provided request ID
func (b *Batch) RetrieveTraceID(requestId string) string {
	b.lock.RLock()
	defer b.lock.RUnlock()
	traceId, exists := b.storeTraceID[requestId]
	if exists {
		return traceId
	}
	return ""
}

// An Invocation holds telemetry for a request, and knows when the request began.
// Invocations are parts of a Batch, and should only be used by the batch object.
type Invocation struct {
	Start     time.Time
	RequestId string
	TraceId   string
	Telemetry [][]byte
}

// NewInvocation creates an Invocation, which can hold telemetry
func NewInvocation(requestId string, start time.Time) Invocation {
	return Invocation{
		Start:     start,
		RequestId: requestId,
		Telemetry: make([][]byte, 0, 2),
	}
}

// IsRipe indicates that an Invocation has all the telemetry it's likely to get. Sending a ripe invocation won't omit data.
func (inv *Invocation) IsRipe() bool {
	return len(inv.Telemetry) >= 2
}

// IsEmpty is true when the invocation has no telemetry. The invocation has begun, but has received no agent payload, nor platform logs.
func (inv *Invocation) IsEmpty() bool {
	return len(inv.Telemetry) == 0
}
