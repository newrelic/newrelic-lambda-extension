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
}
