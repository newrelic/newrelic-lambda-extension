package apm

import (
	"testing"
	"time"
)

func TestGetConnectBackoffTime(t *testing.T) {
	attempts := []time.Duration{
		200 * time.Millisecond,
		500 * time.Millisecond,
		900 * time.Millisecond,
	}
	for k, v := range attempts {
		b := getConnectBackoffTime(k)
		if b != v {
			t.Errorf("Invalid connect backoff for attempt #%d: got %v, want %v", k, b, v)
		}
	}
	b := getConnectBackoffTime(10)
	if b != 900*time.Millisecond {
		t.Errorf("Out-of-bounds attempt should return last backoff, got %v", b)
	}
}
func TestSetState(t *testing.T) {
	app := &InternalAPMApp{}
	run := &appRun{}
	err := error(nil)
	app.setState(run, err)
	if app.Run != run {
		t.Error("Run not set correctly")
	}
	if app.err != err {
		t.Error("Error not set correctly")
	}
}

func TestHarvestStruct(t *testing.T) {
	h := &harvest{data: [][]byte{[]byte("test")}}
	if len(h.data) != 1 {
		t.Error("Expected harvest data length 1")
	}
}
