package util

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

// Compress gzips the given input.
func Compress(b []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	w := gzip.NewWriter(&buf)
	_, err := w.Write(b)
	if err != nil {
		return nil, err
	}

	defer Close(w)

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

	return ioutil.ReadAll(gz)
}
