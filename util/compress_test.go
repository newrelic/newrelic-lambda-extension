package util

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	benchmarkSlice = []byte("asdfasdfasdfasdfasdfasdfasdfasdfasdfasdfasdfasdf")
)

func ControlCompress(b []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	w := gzip.NewWriter(&buf)
	_, err := w.Write(b)
	if err != nil {
		return nil, err
	}

	defer Close(w)
	return &buf, nil
}

func BenchmarkControl(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ControlCompress(benchmarkSlice)
	}
}

func BenchmarkCompressTool(b *testing.B) {
	ct := NewCompressTool()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ct.Compress(benchmarkSlice)
	}
}

func TestCompress(t *testing.T) {
	ct := NewCompressTool()
	b, err := ct.Compress([]byte("foobar"))
	assert.Nil(t, err)
	assert.NotEmpty(t, b)
}

func TestUncompress(t *testing.T) {
	b, err := Uncompress([]byte("foobar"))
	assert.Error(t, err)

	ct := NewCompressTool()
	c, err := ct.Compress([]byte("foobar"))
	assert.Nil(t, err)

	b, err = Uncompress(c.Bytes())
	assert.Nil(t, err)
	assert.NotEmpty(t, b)
}
