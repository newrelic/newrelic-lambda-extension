package apm

import (
	"crypto/rand"
	"sync"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

// TraceIDGenerator creates identifiers for distributed tracing.
type TraceIDGenerator struct {
	sync.Mutex
}

// NewTraceIDGenerator creates a new trace identifier generator.
// The seed parameter is kept for backward compatibility but ignored since we use crypto/rand.
func NewTraceIDGenerator(seed int64) *TraceIDGenerator {
	return &TraceIDGenerator{}
}

// Float32 returns a random float32 using crypto/rand.
func (tg *TraceIDGenerator) Float32() float32 {
	tg.Lock()
	defer tg.Unlock()

	// Generate 4 random bytes and convert to float32
	var bytes [4]byte
	_, err := rand.Read(bytes[:])
	if err != nil {
		// crypto/rand.Read only fails if system randomness is unavailable
		// In such cases, we perform a graceful shutdown
		util.Fatal("crypto/rand.Read failed - system randomness unavailable:", err)
	}
	// Convert to uint32 then to float32 in range [0,1)
	return float32(uint32(bytes[0])<<24|uint32(bytes[1])<<16|uint32(bytes[2])<<8|uint32(bytes[3])) / float32(1<<32)
}

const (
	traceIDByteLen = 16
	// TraceIDHexStringLen is the length of the trace ID when represented
	// as a hex string.
	TraceIDHexStringLen = 32
	spanIDByteLen       = 8
	maxIDByteLen        = 16
)

const (
	hextable = "0123456789abcdef"
)

// GenerateTraceID creates a new trace identifier, which is a 32 character hex string.
func (tg *TraceIDGenerator) GenerateTraceID() string {
	return tg.generateID(traceIDByteLen)
}

// GenerateSpanID creates a new span identifier, which is a 16 character hex string.
func (tg *TraceIDGenerator) GenerateSpanID() string {
	return tg.generateID(spanIDByteLen)
}

func (tg *TraceIDGenerator) generateID(len int) string {
	var bits [maxIDByteLen * 2]byte
	tg.Lock()
	defer tg.Unlock()
	_, err := rand.Read(bits[:len])
	if err != nil {
		// crypto/rand.Read only fails if system randomness is unavailable
		// In such cases, we perform a graceful shutdown
		util.Fatal("crypto/rand.Read failed - system randomness unavailable:", err)
	}

	// In-place encode
	for i := len - 1; i >= 0; i-- {
		bits[i*2+1] = hextable[bits[i]&0x0f]
		bits[i*2] = hextable[bits[i]>>4]
	}
	return string(bits[:len*2])
}
