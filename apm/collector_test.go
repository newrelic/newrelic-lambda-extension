package apm

import (
	"bytes"
	"compress/gzip"
	"io"
	"sync"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/config"
)

func TestCompress_Success(t *testing.T) {
	gzipWriterPool := &sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
			return w
		},
	}
	data := []byte("test data for compression")
	buf, err := compress(data, gzipWriterPool)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if buf == nil || buf.Len() == 0 {
		t.Fatalf("expected non-empty buffer, got %v", buf)
	}

	gr, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()
	decompressed, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Errorf("decompressed data does not match original, got %q, want %q", decompressed, data)
	}
}

func TestPreconnectHost_Default(t *testing.T) {
	conf := &config.Configuration{
		NewRelicHost: "",
		LicenseKey:   "",
	}
	got := preconnectHost(conf)
	want := "collector.newrelic.com"
	if got != want {
		t.Errorf("preconnectHost() = %q, want %q", got, want)
	}
}

func TestPreconnectHost_CustomHost(t *testing.T) {
	conf := &config.Configuration{
		NewRelicHost: "custom.nr-host.com",
		LicenseKey:   "eu01xx000000000000000000000000000000NRAL",
	}
	got := preconnectHost(conf)
	want := "custom.nr-host.com"
	if got != want {
		t.Errorf("preconnectHost() = %q, want %q", got, want)
	}
}

func TestPreconnectHost_RegionLicense(t *testing.T) {
	conf := &config.Configuration{
		NewRelicHost: "",
		LicenseKey:   "eu01xx000000000000000000000000000000NRAL",
	}
	got := preconnectHost(conf)
	want := "collector.eu01.nr-data.net"
	if got != want {
		t.Errorf("preconnectHost() = %q, want %q", got, want)
	}
}

func TestPreconnectHost_RegionLicense_NoMatch(t *testing.T) {
	conf := &config.Configuration{
		NewRelicHost: "",
		LicenseKey:   "invalidlicense",
	}
	got := preconnectHost(conf)
	want := "collector.newrelic.com"
	if got != want {
		t.Errorf("preconnectHost() = %q, want %q", got, want)
	}
}