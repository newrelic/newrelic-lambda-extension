package apm

import (
	"fmt"
	"testing"

)

func TestConnectBackoff(t *testing.T) {
	attempts := map[int]int{
		0: 15,
		1: 15,
		2: 30,
	}

	for k, v := range attempts {
		if b := getConnectBackoffTime(k); b != v {
			t.Error(fmt.Sprintf("Invalid connect backoff for attempt #%d:", k), v)
		}
	}
}
