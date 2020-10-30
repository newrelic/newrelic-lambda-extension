package api

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_formatLogsEndpoint(t *testing.T) {
	endpoint := formatLogsEndpoint(1234)

	assert.Equal(t, "http://sandbox:1234", endpoint)
}
