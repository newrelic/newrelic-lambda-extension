package apm

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type LambdaMetadata struct {
	MetadataVersion      int    `json:"metadata_version"`
	ARN                  string `json:"arn,omitempty"`
	ProtocolVersion      int    `json:"protocol_version"`
	ExecutionEnvironment string `json:"execution_environment,omitempty"`
	AgentVersion         string `json:"agent_version"`
	AgentLanguage        string `json:"agent_language"`
}

type LambdaData struct {
	MetricData          []interface{} `json:"metric_data"`
	CustomEventData     []interface{} `json:"custom_event_data"`
	LogEventData        []interface{} `json:"log_event_data"`
	AnalyticEventData   []interface{} `json:"analytic_event_data"`
	ErrorEventData      []interface{} `json:"error_event_data"`
	ErrorData           []interface{} `json:"error_data"`
	SpanEventData       []interface{} `json:"span_event_data"`
	UpdateLoadedModules []interface{} `json:"update_loaded_modules"`
}

type LambdaRawData struct {
	LambdaMetadata LambdaMetadata `json:"metadata"`
	LambdaData     LambdaData     `json:"data"`
}

func decodeUncompress(input string) ([]byte, error) {
	// Decode base64 first since it's less CPU intensive

	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return nil, err
	}

	// Use a buffer with a larger capacity to reduce allocations
	buf := make([]byte, 0, len(decoded)*2)

	// Directly read into the buffer
	reader, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return nil, fmt.Errorf("error creating gzip reader: %w", err)
	}
	defer reader.Close()

	// Use io.ReadAll to potentially reduce allocations
	uncompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error decompressing data: %w", err)
	}

	// Append to buf to avoid creating a new slice
	buf = append(buf, uncompressed...)

	return buf, nil
}

type UnCompressedData map[string]map[string]json.RawMessage

// This code is essential for processing the payload data received from the Agent
// and extracting the relevant information for further analysis or processing.
func parsePayload(data []byte) (uncompressedData UnCompressedData, err error) {
	var arr [3]json.RawMessage
	if err = json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("unable to unmarshal payload data array: %v", err)
	}

	compressed := strings.Trim(string(arr[2]), `"`)

	// Decode and decompress the data
	dataJSON, err := decodeUncompress(compressed)
	if err != nil {
		return nil, fmt.Errorf("unable to uncompress payload: %v", err)
	}

	var result UnCompressedData
	if err = json.Unmarshal(dataJSON, &result); err != nil {
		return nil, fmt.Errorf("unable to unmarshal uncompressed payload: %v", err)
	}

	return result, nil
}

func GetServerlessData(data []byte) (LambdaRawData, LambdaData, int, error) {
	if len(data) == 0 || data[0] != '[' {
		return LambdaRawData{}, LambdaData{}, 0, nil
	}

	// Remove the square brackets
	jsonData := strings.Trim(strings.TrimSpace(string(data)), `[]`)

	// Get the encoded and compressed part; use TrimSuffix to remove any trailing characters
	components := strings.Split(jsonData, ",")
	if len(components) < 2 {
		return LambdaRawData{}, LambdaData{}, 0, fmt.Errorf("insufficient data components")
	}

	protocolVersion := string(components[0])

	encodedPart := strings.Trim(components[len(components)-1], `"`)

	// Decode and decompress the data encoded data
	uncompressedJSON, err := decodeUncompress(encodedPart)
	if err != nil {
		return LambdaRawData{}, LambdaData{}, 0, fmt.Errorf("failed to decode and decompress data: %w", err)
	}

	switch protocolVersion {
	case "2":
		var result LambdaData
		if err := json.Unmarshal(uncompressedJSON, &result); err != nil {
			return LambdaRawData{}, LambdaData{}, 2, fmt.Errorf("unable to unmarshal JSON data into LambdaData: %w", err)
		}
		return LambdaRawData{}, result, 2, nil
	case "1":
		var result LambdaRawData
		if err := json.Unmarshal(uncompressedJSON, &result); err != nil {
			return LambdaRawData{}, LambdaData{}, 1, fmt.Errorf("unable to unmarshal JSON data into LambdaRawData: %w", err)
		}
		return result, LambdaData{}, 1, nil
	default:
		return LambdaRawData{}, LambdaData{}, 0, fmt.Errorf("unsupported protocol version: %s", protocolVersion)
	}
}
