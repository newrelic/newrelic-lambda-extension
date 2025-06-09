package apm

import (
	"testing"
)

func TestTraceIDGenerator(t *testing.T) {
	tg := NewTraceIDGenerator(112233)
	traceID := tg.GenerateTraceID()
	if traceID != "ddd5ddce8b8426988b123ebc9ab968e3" {
		t.Error(traceID)
	}
	spanID := tg.GenerateSpanID()
	if spanID != "9cfe63dc9910e170" {
		t.Error(spanID)
	}
	if p := tg.Float32(); p != 0.117286205 {
		t.Error(p)
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
