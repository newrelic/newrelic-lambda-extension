package telemetry

import (
	"math"
	"sync"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

// The Unix epoch instant; used as a nil time for eldest and lastHarvest
var epochStart = time.Unix(0, 0)

// Batch represents the unsent invocations and their telemetry, along with timing data.
type Batch struct {
	extractTraceID  bool
	lastHarvest     time.Time
	eldest          time.Time
	ripeDuration    time.Duration
	veryOldDuration time.Duration
	invocations     map[string]*InvocationState
	lock            sync.RWMutex
}

// NewBatch constructs a new batch.
func NewBatch(ripeMillis, rotMillis int64, extractTraceID bool) *Batch {
	initialSize := uint32(math.Min(float64(ripeMillis)/100, 100))
	return &Batch{
		lastHarvest:     epochStart,
		eldest:          epochStart,
		invocations:     make(map[string]*InvocationState, initialSize),
		ripeDuration:    time.Duration(ripeMillis) * time.Millisecond,
		veryOldDuration: time.Duration(rotMillis) * time.Millisecond,
		extractTraceID:  extractTraceID,
	}
}

// AddInvocation should be called just after the next API response. It creates the Invocation record so that we can attach telemetry later.
func (b *Batch) AddInvocation(requestId string, start time.Time) {
	b.lock.Lock()
	defer b.lock.Unlock()

	_, ok := b.invocations[requestId]
	if ok {
		return
	}

	invocation := NewInvocation(requestId, start)
	b.invocations[requestId] = &InvocationState{Invocation: &invocation}
}

// AddTelemetry attaches telemetry to an existing Invocation, identified by requestId
func (b *Batch) AddTelemetry(requestId string, telemetry []byte) *Invocation {
	b.lock.Lock()
	defer b.lock.Unlock()

	state, ok := b.invocations[requestId]
	if state == nil || state.Sent {
		return nil
	}

	inv := state.Invocation
	if ok {
		inv.Telemetry = append(inv.Telemetry, telemetry)
		if b.eldest.Equal(epochStart) {
			b.eldest = inv.Start
		}
		if b.extractTraceID {
			traceId, err := ExtractTraceID(telemetry)
			if err != nil {
				util.Debugln(err)
			}
			// We don't want to unset a previously set trace ID
			if traceId != "" {
				inv.TraceId = traceId
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
	for _, v := range b.invocations {
		if !v.IsEmpty() {
			ret = append(ret, v.Invocation)
			v.MarkSent()
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
	for _, v := range b.invocations {
		if v.IsRipe() {
			ret = append(ret, v.Invocation)
			v.MarkSent()
		} else if !v.Sent && (newEldest.Equal(epochStart) || v.Start.Before(newEldest)) {
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

	inv, ok := b.invocations[requestId]
	if ok {
		return inv.TraceId
	}
	return ""
}

type InvocationState struct {
	Sent bool
	*Invocation
}

// MarkSent sets Sent to true, and deletes the reference to the Invocation Pointer, triggering garbage collection
func (state *InvocationState) MarkSent() {
	state.Invocation = nil
	state.Sent = true
}

// IsRipe indicates that an Invocation has all the telemetry it's likely to get. Sending a ripe invocation won't omit data.
func (state *InvocationState) IsRipe() bool {
	if state.Invocation == nil || state.Sent {
		return false
	}
	return len(state.Invocation.Telemetry) >= 2
}

// IsEmpty is true when the invocation has no telemetry. The invocation has begun, but has received no agent payload, nor platform logs.
func (state *InvocationState) IsEmpty() bool {
	if state.Invocation == nil || state.Sent {
		return true
	}
	return len(state.Invocation.Telemetry) == 0
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
