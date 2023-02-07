package util

import (
	"bytes"
	"compress/gzip"
	"io"
	"sync"
)

type CompressTool struct {
	writers *sync.Pool
}

func NewCompressTool() *CompressTool {
	return &CompressTool{
		writers: &sync.Pool{
			New: func() any {
				return gzip.NewWriter(io.Discard)
			},
		},
	}
}

// Compress gzips the given input.
func (ct *CompressTool) Compress(b []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	w := ct.writers.Get().(*gzip.Writer)
	defer func() {
		Close(w)
		ct.writers.Put(w)
	}()

	w.Reset(&buf)
	_, err := w.Write(b)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

// Uncompress un-gzips the given input.
func Uncompress(b []byte) ([]byte, error) {
	buf := bytes.NewBuffer(b)

	gz, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}

	defer Close(gz)

	return io.ReadAll(gz)
}
