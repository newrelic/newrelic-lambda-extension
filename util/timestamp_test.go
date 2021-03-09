package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimestamp(t *testing.T) {
	assert.NotEmpty(t, Timestamp())
}
