package telemetry

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerialize_DetailedFunctionLog(t *testing.T) {
	commons := map[string]interface{}{
		"foo": "bar",
	}
	messages := []FunctionLogMessage{
		NewFunctionLogMessage(1234, "test1", "123456789", "message1"),
		NewFunctionLogMessage(1235, "test2", "123456789", "message2"),
	}
	dfl := NewDetailedFunctionLog(commons, messages)

	json_bytes, err := json.Marshal(dfl)
	assert.NoError(t, err)
	assert.Equal(t, "{\"common\":{\"attributes\":{\"foo\":\"bar\"}},\"logs\":[{\"message\":\"message1\",\"timestamp\":1234,\"attributes\":{\"aws\":{\"lambda_request_id\":\"test1\"},\"faas.execution\":\"test1\",\"trace.id\":\"123456789\"}},{\"message\":\"message2\",\"timestamp\":1235,\"attributes\":{\"aws\":{\"lambda_request_id\":\"test2\"},\"faas.execution\":\"test2\",\"trace.id\":\"123456789\"}}]}", string(json_bytes))
}
