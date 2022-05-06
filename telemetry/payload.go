package telemetry

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

func parsePayload(data []byte) (metadata, uncompressedData map[string]json.RawMessage, err error) {
	var arr [4]json.RawMessage

	if err = json.Unmarshal(data, &arr); nil != err {
		err = fmt.Errorf("unable to unmarshal payload data array: %v", err)
		return
	}

	var dataJSON []byte
	compressed := strings.Trim(string(arr[3]), `"`)

	if dataJSON, err = decodeUncompress(compressed); nil != err {
		err = fmt.Errorf("unable to uncompress payload: %v", err)
		return
	}

	if err = json.Unmarshal(dataJSON, &uncompressedData); nil != err {
		err = fmt.Errorf("unable to unmarshal uncompressed payload: %v", err)
		return
	}

	if err = json.Unmarshal(arr[2], &metadata); nil != err {
		err = fmt.Errorf("unable to unmarshal payload metadata: %v", err)
		return
	}

	return
}

func decodeUncompress(input string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(input)
	if nil != err {
		return nil, err
	}

	buf := bytes.NewBuffer(decoded)
	gz, err := gzip.NewReader(buf)
	if nil != err {
		return nil, err
	}

	var out bytes.Buffer
	io.Copy(&out, gz)
	gz.Close()

	return out.Bytes(), nil
}

// ExtracTraceID extracts the trace ID within a payload, if present
func ExtractTraceID(data []byte) (string, error) {
	_, segments, err := parsePayload(data)
	if err != nil {
		return "", err
	}

	analyticEvents, ok := segments["analytic_event_data"]
	if ok {
		var parsedAnalyticEvents []json.RawMessage
		if err := json.Unmarshal(analyticEvents, &parsedAnalyticEvents); err != nil {
			return "", err
		}

		if len(parsedAnalyticEvents) == 3 {
			var analyticEvent [][]struct {
				TraceID string `json:"traceId"`
			}
			if err := json.Unmarshal(parsedAnalyticEvents[2], &analyticEvent); err != nil {
				return "", err
			}
			if len(analyticEvent) > 0 && len(analyticEvent[0]) > 0 {
				return analyticEvent[0][0].TraceID, nil
			}
		}
	}

	spanEvents, ok := segments["span_event_data"]
	if ok {
		var parsedSpanEvents []json.RawMessage
		if err := json.Unmarshal(spanEvents, &parsedSpanEvents); err != nil {
			return "", err
		}

		if len(parsedSpanEvents) == 3 {
			var spanEvent [][]struct {
				TraceID string `json:"traceId"`
			}

			if err := json.Unmarshal(parsedSpanEvents[2], &spanEvent); err != nil {
				return "", err
			}

			if len(spanEvent) > 0 && len(spanEvent[0]) > 0 {
				return spanEvent[0][0].TraceID, nil
			}
		}
	}

	return "", errors.New("No trace ID found in payload")
}
