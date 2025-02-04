package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

const (
	maxCompressedPayloadLen = 1000 * 1024
)

// DetailedFunctionLog is the Logs API payload
type DetailedFunctionLog struct {
	Common CommonLogAttrs       `json:"common"`
	Logs   []FunctionLogMessage `json:"logs"`
}

type CommonLogAttrs struct {
	Attributes map[string]interface{} `json:"attributes"`
}

type FunctionLogMessage struct {
	Message    string                 `json:"message"`
	Timestamp  int64                  `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes"`
}

func NewFunctionLogMessage(timestamp int64, requestId, traceId, message string) FunctionLogMessage {
	attributes := map[string]interface{}{
		"aws": map[string]string{
			"lambda_request_id": requestId,
		},
		"faas.execution": requestId,
	}

	if traceId != "" {
		attributes["trace.id"] = traceId
	}

	return FunctionLogMessage{
		Message:    message,
		Timestamp:  timestamp,
		Attributes: attributes,
	}
}

func NewDetailedFunctionLog(common map[string]interface{}, logs []FunctionLogMessage) DetailedFunctionLog {
	return DetailedFunctionLog{
		Common: CommonLogAttrs{
			Attributes: common,
		},
		Logs: logs,
	}
}

// RequestContext is the Vortex request context
type RequestContext struct {
	FunctionName       string `json:"function_name"`
	InvokedFunctionARN string `json:"invoked_function_arn"`
	// Below are not relevant to Lambda Extensions, but ingest requires these to be present
	LogGroupName  string `json:"log_group_name"`
	LogStreamName string `json:"log_stream_name"`
}

// RequestData is the body of the Vortex request
type RequestData struct {
	Context RequestContext `json:"context"`
	Entry   string         `json:"entry"`
}

// LogsEntry is a CloudWatch Logs entry
type LogsEntry struct {
	LogEvents []LogsEvent `json:"logEvents"`
	// Below are not relevant to Lambda Extensions, but ingest expects these to be present
	LogGroup    string `json:"logGroup"`
	LogStream   string `json:"logStream"`
	MessageType string `json:"messageType"`
	Owner       string `json:"owner"`
}

// LogsEvent is a CloudWatch Logs event
type LogsEvent struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

func LogsEventForBytes(payload []byte) LogsEvent {
	return LogsEvent{ID: util.UUID(), Message: string(payload), Timestamp: util.Timestamp()}
}

func CompressedPayloadsForLogEvents(logsEvents []LogsEvent, functionName string, invokedFunctionARN string) ([]*bytes.Buffer, error) {
	logGroupName := fmt.Sprintf("/aws/lambda/%s", functionName)
	logEntry := LogsEntry{
		LogEvents: logsEvents,
		LogGroup:  logGroupName,
	}

	entry, err := json.Marshal(logEntry)
	if err != nil {
		return nil, err
	}

	context := RequestContext{
		FunctionName:       functionName,
		InvokedFunctionARN: invokedFunctionARN,
		LogGroupName:       logGroupName,
		LogStreamName:      util.Id,
	}
	data := RequestData{Context: context, Entry: string(entry)}

	compressed, err := CompressedJsonPayload(data)
	if err != nil {
		return nil, err
	}

	if compressed.Len() <= maxCompressedPayloadLen {
		return []*bytes.Buffer{compressed}, nil
	} else if len(logsEvents) > 1 {
		// Payload is too large, split in half, recursively
		split := len(logsEvents) / 2
		leftRet, err := CompressedPayloadsForLogEvents(logsEvents[0:split], functionName, invokedFunctionARN)
		if err != nil {
			return nil, err
		}

		rightRet, err := CompressedPayloadsForLogEvents(logsEvents[split:], functionName, invokedFunctionARN)
		if err != nil {
			return nil, err
		}

		return append(leftRet, rightRet...), nil
	} else {
		// when there is one event that is too large, we try to send it anyway. It will fail with a 413 error, but wont loop infinitely.
		return []*bytes.Buffer{compressed}, nil
	}
}

// BuildVortexRequest builds a Vortex HTTP request
func BuildVortexRequest(ctx context.Context, url string, compressed *bytes.Buffer, userAgent string, licenseKey string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, compressed)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("X-License-Key", licenseKey)

	return req, nil
}

func CompressedJsonPayload(payload interface{}) (*bytes.Buffer, error) {
	uncompressed, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	compressed, err := util.Compress(uncompressed)
	if err != nil {
		return nil, fmt.Errorf("error compressing data: %v", err)
	}

	return compressed, nil
}
