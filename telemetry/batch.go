package telemetry

import "time"

const (
	BatchSize uint = Ripe / 100
	VeryOld = 10_000
	Ripe = 3000
)

type Batch struct {
	LastSend    time.Time
	Eldest      time.Time
	Invocations map[string]Invocation
}

func NewBatch() Batch {
	return Batch{
		LastSend: time.Unix(0, 0),
		Eldest: time.Unix(0,0),
		Invocations: make(map[string]Invocation, BatchSize),
	}
}

func (b *Batch) AddTelemetry(requestId string, telemetry []byte) *Invocation {
	inv, ok := b.Invocations[requestId]
	if ok {
		inv.Telemetry = append(inv.Telemetry, telemetry)
		return &inv
	}
	return nil
}

func (b *Batch) Harvest(now time.Time) []Invocation {
	if len(b.Invocations) == 0 {
		return nil
	}

	veryOldTime := now.Add(VeryOld * time.Millisecond)
	if b.LastSend.Before(veryOldTime) {
		return b.aggressiveHarvest()
	}

	ripeTime := now.Add(Ripe * time.Millisecond)
	if b.Eldest.Before(ripeTime) {
		return b.completeHarvest()
	}
	return nil
}

func (b *Batch) aggressiveHarvest() []Invocation {
	ret := make([]Invocation, len(b.Invocations))
	for k, v := range b.Invocations{
		ret = append(ret, v)
		delete(b.Invocations, k)
	}
	return ret
}

func (b *Batch) completeHarvest() []Invocation {
	ret := make([]Invocation, len(b.Invocations))
	for k, v := range b.Invocations{
		if v.IsComplete() {
			ret = append(ret, v)
			delete(b.Invocations, k)
		}
	}
	return ret
}

type Invocation struct {
	Start time.Time
	RequestId string
	Telemetry [][]byte
}

func NewInvocation(start time.Time, requestId string) Invocation {
	return Invocation{
		Start: start,
		RequestId: requestId,
		Telemetry: make([][]byte, 2),
	}
}

func (inv *Invocation) IsComplete() bool {
	return len(inv.Telemetry) >= 2
}
