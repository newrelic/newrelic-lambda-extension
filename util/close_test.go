package util

import (
	"fmt"
	"testing"
)

type mockCloseable struct{}

func (mockCloseable) Close() error {
	return fmt.Errorf("Something went wrong")
}

func TestClose(t *testing.T) {
	c := &mockCloseable{}
	Close(c)
}
