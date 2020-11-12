package api

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_formatLogsEndpoint(t *testing.T) {
	endpoint := formatLogsEndpoint(1234)

	assert.Equal(t, "http://sandbox:1234", endpoint)
}

func Test_DefaultLogSubscription(t *testing.T) {
	types := []LogEventType{Platform}
	sub := DefaultLogSubscription(types, 2345)

	assert.Equal(t, LogBufferDefaultBytes, sub.Buffering.MaxBytes)
	assert.Equal(t, LogBufferDefaultItems, sub.Buffering.MaxItems)
	assert.Equal(t, LogBufferDefaultTimeout, sub.Buffering.TimeoutMs)

	assert.Equal(t, "http://sandbox:2345", sub.Destination.URI)
	assert.Equal(t, "HTTP", sub.Destination.Protocol)
	assert.Equal(t, types, sub.Types)
}
