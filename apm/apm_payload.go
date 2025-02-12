package apm

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	MetricData          []interface{}   `json:"metric_data"`
	CustomEventData     []interface{}   `json:"custom_event_data"`
	LogEventData        []interface{}   `json:"log_event_data"`
	AnalyticEventData   []interface{}   `json:"analytic_event_data"`
	ErrorEventData      []interface{}   `json:"error_event_data"`
	ErrorData 		    []interface{}   `json:"error_data"`
	SpanEventData       []interface{}   `json:"span_event_data"`
	UpdateLoadedModules []interface{}   `json:"update_loaded_modules"`
}

type LambdaRawData struct {
	LambdaMetadata LambdaMetadata `json:"metadata"`
	LambdaData     LambdaData     `json:"data"`
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

type uncompressedData map[string]map[string]json.RawMessage

func parsePayload(data []byte) (uncompressedData uncompressedData, err error) {
	var arr [3]json.RawMessage

	if err = json.Unmarshal(data, &arr); err != nil {
		err = fmt.Errorf("unable to unmarshal payload data array: %v", err)
		return
	}

	var dataJSON []byte
	compressed := strings.Trim(string(arr[2]), `"`)

	if dataJSON, err = decodeUncompress(compressed); err != nil {
		err = fmt.Errorf("unable to uncompress payload: %v", err)
		return
	}

	if err = json.Unmarshal(dataJSON, &uncompressedData); err != nil {
		err = fmt.Errorf("unable to unmarshal uncompressed payload: %v", err)
		return
	}

	return
}

func GetServerlessData(data []byte) (LambdaRawData, error) {
	if data != nil && data[0] != '[' {
		return LambdaRawData{}, nil
	}

	jsonData := strings.Trim(string(data), "[]")
	components := strings.Split(jsonData, ",")
	encodedPart := strings.Trim(components[2], "\"")

	uncompressedJSON, err := decodeUncompress(encodedPart)

	if err != nil {
		log.Fatalf("Failed to decode and decompress data: %v", err)
	}
	var result LambdaRawData
	if err = json.Unmarshal(uncompressedJSON, &result); err != nil {
		log.Fatalf("Unable to unmarshal JSON data into struct: %v", err)
	}
	return result, nil
}
