package apm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMapToErrorEventData(t *testing.T) {
	errorData := []interface{}{
		"Lambda.Timedout",                      // ErrorClass
		nil,                                    // Unused
		"123a1a12-1234-1234-1234-123456789012", // AwsRequestId
		"Task timed out after 1.00 seconds",    // ErrorMessage
		"myLambdaFunc",                         // LambdaFunctionName
		"123456789012",                         // LambdaAccountId
		"42",                                   // LambdaFunctionVersion
	}
	runID := "test-run-id"
	spanID := "test-span-id"
	traceID := "test-trace-id"
	guid := "test-guid"

	event, err := MapToErrorEventData(errorData, runID, spanID, traceID, guid)
	assert.NoError(t, err)
	assert.Len(t, event, 3)

	assert.Equal(t, runID, event[0])

	metrics, ok := event[1].(map[string]int)
	assert.True(t, ok)
	assert.Equal(t, 1, metrics["events_seen"])
	assert.Equal(t, 100, metrics["reservoir_size"])

	events, ok := event[2].([][]interface{})
	assert.True(t, ok)
	assert.Len(t, events, 1)
	eventDetail, ok := events[0][0].(EventDetail)
	assert.True(t, ok)
	assert.Equal(t, 0.1, eventDetail.Duration)
	assert.Equal(t, "Lambda.Timedout", eventDetail.ErrorClass)
	assert.False(t, eventDetail.ErrorExpected)
	assert.Equal(t, "Lambda.Timedout:Task timed out after 1.00 seconds", eventDetail.ErrorMessage)
	assert.Equal(t, guid, eventDetail.Guid)
	assert.Equal(t, guid, eventDetail.TransactionGuid)
	assert.Equal(t, 1.5, eventDetail.Priority)
	assert.True(t, eventDetail.Sampled)
	assert.Equal(t, spanID, eventDetail.SpanId)
	assert.InDelta(t, float64(time.Now().UnixMilli()), float64(eventDetail.Timestamp), 1000)
	assert.Equal(t, traceID, eventDetail.TraceId)
	assert.Equal(t, "OtherTransaction/Function/myLambdaFunc", eventDetail.TransactionName)
	assert.Equal(t, "TransactionError", eventDetail.Type)

	customAttrs, ok := events[0][2].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "arn:aws:lambda:region:123456789012:function:myLambdaFunc:42", customAttrs["aws.lambda.arn"])
	assert.Equal(t, "42", customAttrs["aws.lambda.functionVersion"])
	assert.Equal(t, "123a1a12-1234-1234-1234-123456789012", customAttrs["aws.requestId"])
}