package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePayload(t *testing.T) {
	payload := []byte("[1, \"NR_LAMBDA_MONITORING\", {\"arn\": \"AWS_LAMBDA_FUNCTION_ARN\", \"protocol_version\": 16, \"function_version\": \"15\", \"execution_environment\": \"AWS_Lambda_python3.6\", \"agent_version\": \"4.2.0.100\", \"metadata_version\": 2, \"agent_language\": \"python\"}, \"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\"]")
	metadata, data, err := parsePayload(payload)
	assert.NotNil(t, metadata)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestParsePayloadBlank(t *testing.T) {
	payload := []byte("")
	metadata, data, err := parsePayload(payload)
	assert.Nil(t, metadata)
	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestParsePayloadInvalidMetaData(t *testing.T) {
	payload := []byte("[1, \"NR_LAMBDA_MONITORING\", \"foobar\", null]")
	metadata, data, err := parsePayload(payload)
	assert.Nil(t, metadata)
	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestParsePayloadInvalidCompressedData(t *testing.T) {
	payload := []byte("[1, \"NR_LAMBDA_MONITORING\", {\"arn\": \"AWS_LAMBDA_FUNCTION_ARN\", \"protocol_version\": 16, \"function_version\": \"15\", \"execution_environment\": \"AWS_Lambda_python3.6\", \"agent_version\": \"4.2.0.100\", \"metadata_version\": 2, \"agent_language\": \"python\"}, \"foobar\"]")
	metadata, data, err := parsePayload(payload)
	assert.Nil(t, metadata)
	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestParsePayloadInvalidData(t *testing.T) {
	payload := []byte("[1, \"NR_LAMBDA_MONITORING\", {\"arn\": \"AWS_LAMBDA_FUNCTION_ARN\", \"protocol_version\": 16, \"function_version\": \"15\", \"execution_environment\": \"AWS_Lambda_python3.6\", \"agent_version\": \"4.2.0.100\", \"metadata_version\": 2, \"agent_language\": \"python\"}, \"H4sIAK6pdWIC/0vLz09KLAIAlR/2ngYAAAA=\"]")
	metadata, data, err := parsePayload(payload)
	assert.Nil(t, metadata)
	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestExtractTraceIDAnalyticEvent(t *testing.T) {
	payload := []byte("[1, \"NR_LAMBDA_MONITORING\", {\"arn\": \"AWS_LAMBDA_FUNCTION_ARN\", \"protocol_version\": 16, \"function_version\": \"15\", \"execution_environment\": \"AWS_Lambda_python3.6\", \"agent_version\": \"4.2.0.100\", \"metadata_version\": 2, \"agent_language\": \"python\"}, \"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\"]")

	traceId, err := ExtractTraceID(payload)

	assert.Equal(t, "24d071916e1f00ee", traceId)
	assert.Nil(t, err)
}

func TestExtractTraceIDSpanEvent(t *testing.T) {
	payload := []byte("[1, \"NR_LAMBDA_MONITORING\", {\"arn\": \"AWS_LAMBDA_FUNCTION_ARN\", \"protocol_version\": 16, \"function_version\": \"15\", \"execution_environment\": \"AWS_Lambda_python3.6\", \"agent_version\": \"4.2.0.100\", \"metadata_version\": 2, \"agent_language\": \"python\"}, \"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\"]")

	traceId, err := ExtractTraceID(payload)

	assert.Equal(t, "446cf2064d931f4e", traceId)
	assert.Nil(t, err)
}

func TestExtractTraceIDInvalid(t *testing.T) {
	payload := []byte("[1, \"NR_LAMBDA_MONITORING\", {\"arn\": \"AWS_LAMBDA_FUNCTION_ARN\", \"protocol_version\": 16, \"function_version\": \"15\", \"execution_environment\": \"AWS_Lambda_python3.6\", \"agent_version\": \"4.2.0.100\", \"metadata_version\": 2, \"agent_language\": \"python\"}, \"H4sIAK6pdWIC/0vLz09KLAIAlR/2ngYAAAA=\"]")

	traceId, err := ExtractTraceID(payload)

	assert.Empty(t, traceId)
	assert.Error(t, err)
}
