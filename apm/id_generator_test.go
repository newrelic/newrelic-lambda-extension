package apm

import (
	"testing"
)

func TestTraceIDGenerator(t *testing.T) {
	tg := NewTraceIDGenerator(112233) // seed is ignored but kept for compatibility
	
	// Test trace ID generation
	traceID := tg.GenerateTraceID()
	if len(traceID) != TraceIDHexStringLen {
		t.Errorf("Expected trace ID length %d, got %d: %s", TraceIDHexStringLen, len(traceID), traceID)
	}
	// Verify it's a valid hex string
	for _, char := range traceID {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			t.Errorf("Invalid hex character in trace ID: %c", char)
		}
	}
	
	// Test span ID generation
	spanID := tg.GenerateSpanID()
	if len(spanID) != 16 { // spanIDByteLen * 2
		t.Errorf("Expected span ID length 16, got %d: %s", len(spanID), spanID)
	}
	// Verify it's a valid hex string
	for _, char := range spanID {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			t.Errorf("Invalid hex character in span ID: %c", char)
		}
	}
	
	// Test Float32 generation
	p := tg.Float32()
	if p < 0 || p >= 1 {
		t.Errorf("Float32 should be in range [0,1), got %f", p)
	}
	
	// Test uniqueness - generate multiple IDs and ensure they're different
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := tg.GenerateTraceID()
		if ids[id] {
			t.Errorf("Generated duplicate trace ID: %s", id)
		}
		ids[id] = true
	}
}

func BenchmarkTraceIDGenerator(b *testing.B) {
	tg := NewTraceIDGenerator(12345)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if id := tg.GenerateSpanID(); id == "" {
			b.Fatal(id)
		}
	}
}

func BenchmarkTraceIDGeneratorParallel(b *testing.B) {
	tg := NewTraceIDGenerator(112233)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			if id := tg.GenerateSpanID(); id == "" {
				b.Fatal(id)
			}
		}
	})
}
