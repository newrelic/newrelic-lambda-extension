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

type uncompressedDataVersion2 map[string]json.RawMessage
type uncompressedDataVersion1 map[string]map[string]json.RawMessage

func parsePayload(data []byte) (uncompressedData map[string]json.RawMessage, err error) {
	var arr [4]json.RawMessage

	if err = json.Unmarshal(data, &arr); err != nil {
		err = fmt.Errorf("unable to unmarshal payload data array: %v", err)
		return
	}
	payloadVersion := strings.Trim(string(arr[0]), `"`)
	var dataCompressed string
	if payloadVersion == "2" {
		dataCompressed = string(arr[3])
	} else {
		dataCompressed = string(arr[2])
	}
	var dataJSON []byte
	compressed := strings.Trim(dataCompressed, `"`)

	if dataJSON, err = decodeUncompress(compressed); err != nil {
		err = fmt.Errorf("unable to uncompress payload: %v", err)
		return
	}

	if payloadVersion == "2" {
		var uncompressed uncompressedDataVersion2
		if err = json.Unmarshal(dataJSON, &uncompressed); err != nil {
			fmt.Printf("unable to unmarshal uncompressed payload: %v", err)
			return nil, err
		}
		uncompressedData = uncompressed
	} else {
		var uncompressed uncompressedDataVersion1
		if err = json.Unmarshal(dataJSON, &uncompressed); err != nil {
			fmt.Printf("unable to unmarshal uncompressed payload: %v", err)
			return nil, err
		}
		uncompressedData = uncompressed["data"]
	}

	return
}

func decodeUncompress(input string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(decoded)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	io.Copy(&out, gz)
	gz.Close()

	return out.Bytes(), nil
}

// ExtractTraceID extracts the trace ID within a payload, if present
func ExtractTraceID(data []byte) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return "", err
	}

	if !bytes.Contains(decoded, []byte("NR_LAMBDA_MONITORING")) {
		return "", nil
	}

	dataSegment, err := parsePayload(decoded)
	if err != nil {
		return "", err
	}


	analyticEvents, ok := dataSegment["analytic_event_data"]
	if ok {
		var parsedAnalyticEvents []json.RawMessage
		if err := json.Unmarshal(analyticEvents, &parsedAnalyticEvents); err != nil {
			return "", err
		}

		if len(parsedAnalyticEvents) > 2 {
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

	spanEvents, ok := dataSegment["span_event_data"]
	if ok {
		var parsedSpanEvents []json.RawMessage
		if err := json.Unmarshal(spanEvents, &parsedSpanEvents); err != nil {
			return "", err
		}

		if len(parsedSpanEvents) > 2 {
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
