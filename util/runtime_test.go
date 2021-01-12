package util

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestDetectRuntime(t *testing.T) {
	assert.Equal(t, DetectRuntime(), "unknown")

	defer func() {
		os.Unsetenv("AWS_EXECUTION_ENV")
	}()

	os.Setenv("AWS_EXECUTION_ENV", "")
	assert.Equal(t, DetectRuntime(), "unknown")

	// AWS capitalizes the value according to the docs
	os.Setenv("AWS_EXECUTION_ENV", "aws_lambda_python3.6")
	assert.Equal(t, DetectRuntime(), "unknown")

	os.Setenv("AWS_EXECUTION_ENV", "AWS_Lambda_python3.6")
	assert.Equal(t, DetectRuntime(), "python3.6")
}
