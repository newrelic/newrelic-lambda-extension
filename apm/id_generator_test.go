package apm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateID(t *testing.T) {
	seed := int64(12345)
	tg := NewTraceIDGenerator(seed)

	t.Run("GenerateTraceID", func(t *testing.T) {
		traceID := tg.GenerateTraceID()
		assert.NotEmpty(t, traceID, "TraceID should not be empty")
		assert.Equal(t, TraceIDHexStringLen, len(traceID), "TraceID should have the correct length")
	})

	t.Run("GenerateSpanID", func(t *testing.T) {
		spanID := tg.GenerateSpanID()
		assert.NotEmpty(t, spanID, "SpanID should not be empty")
		assert.Equal(t, spanIDByteLen*2, len(spanID), "SpanID should have the correct length")
	})

	t.Run("GenerateUniqueIDs", func(t *testing.T) {
		id1 := tg.GenerateTraceID()
		id2 := tg.GenerateTraceID()
		assert.NotEqual(t, id1, id2, "Generated TraceIDs should be unique")

		id3 := tg.GenerateSpanID()
		id4 := tg.GenerateSpanID()
		assert.NotEqual(t, id3, id4, "Generated SpanIDs should be unique")
	})
}
