package agentTelemetry

import (
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	MustHarvestError = "telemetry can not be added until batch is harvested"
)

var l = log.WithFields(log.Fields{"pkg": "agentTelemetry"})

// Batch represents the unsent invocations and their telemetry, along with timing data.
type Batch struct {
	extractTraceID bool
	harvestSize    int
	invocations    map[string]*Invocation
}

// NewBatch constructs a new batch.
func NewBatch(size int, extractTraceID bool, logLevel log.Level) *Batch {
	log.SetLevel(logLevel)
	return &Batch{
		invocations:    make(map[string]*Invocation),
		harvestSize:    size,
		extractTraceID: extractTraceID,
	}
}

func (b *Batch) ReadyToHarvest() bool {
	l.Debugf("[agentTelemetry] Have %d / %d invocations needed to do a harvest", len(b.invocations), b.harvestSize)
	return len(b.invocations) >= b.harvestSize
}

// AddInvocation should be called just after the next API response. It creates the Invocation record so that we can attach telemetry later.
func (b *Batch) AddInvocation(requestID string, start time.Time) {
	invocation := NewInvocation(requestID, start)
	b.invocations[requestID] = &invocation
}

// HasInvocation checks if an invocation has been created for this request ID
func (b *Batch) HasInvocation(requestID string) bool {
	_, ok := b.invocations[requestID]
	return ok
}

// AddTelemetry attaches telemetry to an existing Invocation, identified by requestId
func (b *Batch) AddTelemetry(requestId string, telemetry []byte) *Invocation {
	inv, ok := b.invocations[requestId]
	if ok {
		inv.Telemetry = append(inv.Telemetry, telemetry)
		if b.extractTraceID {
			traceId, err := ExtractTraceID(telemetry)
			if err != nil {
				l.Errorf("[agent telemtry: ExtractTraceID] %v", err)
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

// Close aggressively harvests all telemetry from the Batch. The Batch is no longer valid.
func (b *Batch) Close() []*Invocation {
	return b.Harvest(true)
}

// Harvest all ready invocations. It removes harvested invocations from the batch and updates the lastHarvest timestamp.
func (b *Batch) Harvest(force bool) []*Invocation {
	ret := make([]*Invocation, 0, len(b.invocations))
	for k, v := range b.invocations {
		if force {
			if !v.IsEmpty() {
				ret = append(ret, v)
				delete(b.invocations, k)
			}
		} else {
			if !v.IsEmpty() && v.IsRipe() {
				ret = append(ret, v)
				delete(b.invocations, k)
			}
		}
	}

	l.Debugf("[agentTelemetry] harvesting %d invocations\n", len(ret))
	return ret
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
	return len(inv.Telemetry) >= 1
}

// IsEmpty is true when the invocation has no telemetry. The invocation has begun, but has received no agent payload, nor platform logs.
func (inv *Invocation) IsEmpty() bool {
	return len(inv.Telemetry) == 0
}
