package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUUID(t *testing.T) {
	assert.NotEmpty(t, UUID())
}
