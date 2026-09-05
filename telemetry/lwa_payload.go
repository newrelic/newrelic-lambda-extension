// Package telemetry provides tools to process and enrich telemetry payloads for Lambda Web Adapter.
// This package specifically handles telemetry processing when NEW_RELIC_LAMBDA_WEB_ADAPTER is enabled.
//
// Lambda Web Adapter (LWA) Processing:
// - Decodes base64-encoded telemetry from the Newrelic agent
// - Enriches telemetry data with AWS Lambda context (RequestID, ARN, Function Version, Cold Start)
// - Returns processed telemetry as raw JSON bytes instead of base64-encoded strings
// - Supports both protocol versions 1 and 2 from the Newrelic agent
package telemetry

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

// LambdaMetadata contains metadata about the Lambda execution environment.
type LambdaMetadata struct {
	MetadataVersion      int    `json:"metadata_version"`
	ARN                  string `json:"arn,omitempty"`
	ProtocolVersion      int    `json:"protocol_version"`
	ExecutionEnvironment string `json:"execution_environment,omitempty"`
	AgentVersion         string `json:"agent_version"`
	AgentLanguage        string `json:"agent_language"`
	FunctionVersion      string `json:"function_version,omitempty"`
}

// LambdaData holds the various telemetry data points.
type LambdaData struct {
	MetricData            []interface{} `json:"metric_data"`
	CustomEventData       []interface{} `json:"custom_event_data"`
	LogEventData          []interface{} `json:"log_event_data"`
	AnalyticEventData     []interface{} `json:"analytic_event_data"`
	ErrorEventData        []interface{} `json:"error_event_data"`
	ErrorData             []interface{} `json:"error_data"`
	SpanEventData         []interface{} `json:"span_event_data"`
	UpdateLoadedModules   []interface{} `json:"update_loaded_modules"`
	TransactionSampleData []interface{} `json:"transaction_sample_data"`
}

// LambdaRawData is the top-level structure for a Protocol v1 payload.
type LambdaRawData struct {
	LambdaMetadata LambdaMetadata `json:"metadata"`
	LambdaData     LambdaData     `json:"data"`
}

// AWSLambdaContext provides the AWS-specific data to be added to telemetry payloads
// when processing through Lambda Web Adapter.
type AWSLambdaContext struct {
	RequestID       string
	ARN             string
	FunctionVersion string
	ColdStart       bool
}


// ProcessTelemetry takes a raw telemetry string and AWS context, and returns enriched
// telemetry data as JSON bytes for Lambda Web Adapter processing.
// This is the main entry point for Lambda Web Adapter telemetry enrichment.
func ProcessTelemetry(telemetryBytes string, awsContext AWSLambdaContext) ([]byte, error) {
	rawData, lambdaData, protocolVersion, err := getServerlessData([]byte(telemetryBytes))
	if err != nil {
		util.Logf("Error decoding serverless data: %v", err)
		return nil, fmt.Errorf("error decoding serverless data: %w", err)
	}

	if err := addAWSData(&lambdaData, awsContext); err != nil {
		util.Logf("Error adding AWS data: %v", err)
		return nil, fmt.Errorf("error adding AWS data: %w", err)
	}

	finalPayload, err := createTelemetryBytes(lambdaData, rawData.LambdaMetadata, awsContext, protocolVersion, "NR_LAMBDA_MONITORING")
	if err != nil {
		util.Logf("Error creating final telemetry bytes: %v", err)
		return nil, fmt.Errorf("error creating final telemetry bytes: %w", err)
	}

	return finalPayload, nil
}

// getServerlessData decodes and parses base64-encoded telemetry data from the Newrelic agent
// for processing in Lambda Web Adapter. Supports both protocol versions 1 and 2.
func getServerlessData(data []byte) (LambdaRawData, LambdaData, int, error) {
	decodedJSON, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		util.Logf("Failed to base64 decode payload: %v", err)
		return LambdaRawData{}, LambdaData{}, 0, fmt.Errorf("failed to base64 decode payload: %w", err)
	}

	var rawPayload []json.RawMessage
	if err := json.Unmarshal(decodedJSON, &rawPayload); err != nil {
		util.Logf("Failed to unmarshal JSON array: %v", err)
		return LambdaRawData{}, LambdaData{}, 0, fmt.Errorf("failed to unmarshal JSON array: %w", err)
	}
	if len(rawPayload) < 3 {
		util.Logf("Invalid payload structure, expected at least 3 elements, got %d", len(rawPayload))
		return LambdaRawData{}, LambdaData{}, 0, fmt.Errorf("invalid payload structure, expected at least 3 elements")
	}

	var protocolVersion int
	if err := json.Unmarshal(rawPayload[0], &protocolVersion); err != nil {
		util.Logf("Failed to parse protocol version: %v", err)
		return LambdaRawData{}, LambdaData{}, 0, fmt.Errorf("failed to parse protocol version: %w", err)
	}

	var encodedPart string
	if err := json.Unmarshal(rawPayload[2], &encodedPart); err != nil {
		util.Logf("Failed to parse encoded data part: %v", err)
		return LambdaRawData{}, LambdaData{}, 0, fmt.Errorf("failed to parse encoded data part: %w", err)
	}

	uncompressedJSON, err := decodeUncompress(encodedPart)
	if err != nil {
		util.Logf("Failed to decode and decompress data: %v", err)
		return LambdaRawData{}, LambdaData{}, 0, fmt.Errorf("failed to decode and decompress data: %w", err)
	}

	switch protocolVersion {
	case 1:
		util.Debugf("Processing protocol version 1 for Lambda Web Adapter")
		var result LambdaRawData
		if err := json.Unmarshal(uncompressedJSON, &result); err != nil {
			util.Logf("Unable to unmarshal JSON data into LambdaRawData: %v", err)
			return LambdaRawData{}, LambdaData{}, 1, fmt.Errorf("unable to unmarshal JSON data into LambdaRawData: %w", err)
		}
		return result, result.LambdaData, 1, nil
	case 2:
		util.Debugf("Processing protocol version 2 for Lambda Web Adapter")
		var result LambdaData
		if err := json.Unmarshal(uncompressedJSON, &result); err != nil {
			util.Logf("Unable to unmarshal JSON data into LambdaData: %v", err)
			return LambdaRawData{}, LambdaData{}, 2, fmt.Errorf("unable to unmarshal JSON data into LambdaData: %w", err)
		}
		return LambdaRawData{}, result, 2, nil
	default:
		util.Logf("Unsupported protocol version for Lambda Web Adapter: %d", protocolVersion)
		return LambdaRawData{}, LambdaData{}, 0, fmt.Errorf("unsupported protocol version: %d", protocolVersion)
	}
}

func addAWSData(lambdaData *LambdaData, awsContext AWSLambdaContext) error {
	util.Debugf("Adding AWS context data for Lambda Web Adapter: RequestID=%s, ARN=%s, FunctionVersion=%s, ColdStart=%t", 
		awsContext.RequestID, awsContext.ARN, awsContext.FunctionVersion, awsContext.ColdStart)
	
	awsAttributes := map[string]interface{}{
		"aws.requestId":            awsContext.RequestID,
		"aws.lambda.arn":           awsContext.ARN,
		"aws.lambda.functionVersion": awsContext.FunctionVersion,
	}

	if awsContext.ColdStart {
		awsAttributes["aws.lambda.coldStart"] = true
	}

	if err := processEventData(lambdaData.AnalyticEventData, awsAttributes); err != nil {
		util.Logf("Failed to process analytic event data: %v", err)
		return fmt.Errorf("failed to process analytic event data: %w", err)
	}
	if err := processEventData(lambdaData.SpanEventData, awsAttributes); err != nil {
		util.Logf("Failed to process span event data: %v", err)
		return fmt.Errorf("failed to process span event data: %w", err)
	}
	return nil
}

// createTelemetryBytes creates the final telemetry payload as JSON bytes for Lambda Web Adapter.
// Returns raw JSON bytes instead of base64-encoded string for direct consumption.
func createTelemetryBytes(data LambdaData, metadata LambdaMetadata, awsContext AWSLambdaContext, protocolVersion int, agentName string) ([]byte, error) {
	util.Debugf("Creating telemetry bytes for Lambda Web Adapter - protocol version %d with agent %s", protocolVersion, agentName)
	
	metadata.ARN = awsContext.ARN
	metadata.FunctionVersion = awsContext.FunctionVersion

	rawData := LambdaRawData{
		LambdaMetadata: metadata,
		LambdaData:     data,
	}

	rawJSON, err := json.Marshal(rawData)
	if err != nil {
		util.Logf("Failed to marshal raw data: %v", err)
		return nil, fmt.Errorf("failed to marshal raw data: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(rawJSON); err != nil {
		util.Logf("Failed to gzip data: %v", err)
		return nil, fmt.Errorf("failed to gzip data: %w", err)
	}
	if err := gz.Close(); err != nil {
		util.Logf("Failed to close gzip writer: %v", err)
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	innerPayload := base64.StdEncoding.EncodeToString(buf.Bytes())

	finalArray := []interface{}{
		protocolVersion,
		agentName,
		innerPayload,
	}

	finalJSON, err := json.Marshal(finalArray)
	if err != nil {
		util.Logf("Failed to marshal final array: %v", err)
		return nil, fmt.Errorf("failed to marshal final array: %w", err)
	}

	util.Debugf("Successfully created Lambda Web Adapter telemetry JSON with length %d bytes", len(finalJSON))
	return finalJSON, nil
}

// processEventData enriches event data with AWS attributes for Lambda Web Adapter telemetry.
// This function adds AWS context information to analytic and span events.
func processEventData(eventData []interface{}, attributes map[string]interface{}) error {
	if len(eventData) < 3 {
		util.Debugf("Event data has insufficient elements (%d), skipping processing", len(eventData))
		return nil
	}
	eventsList, ok := eventData[2].([]interface{})
	if !ok {
		util.Logf("Expected events list at index 2 to be a []interface{}, but it was not")
		return fmt.Errorf("expected events list at index 2 to be a []interface{}, but it was not")
	}
	
	processedEvents := 0
	for _, eventEntry := range eventsList {
		event, ok := eventEntry.([]interface{})
		if !ok || len(event) < 3 {
			util.Debugf("Skipping invalid event entry")
			continue
		}
		agentAttributes, ok := event[2].(map[string]interface{})
		if !ok {
			util.Debugf("Skipping event with invalid attributes structure")
			continue
		}
		for key, value := range attributes {
			agentAttributes[key] = value
		}
		processedEvents++
	}
	
	util.Debugf("Processed %d events with AWS attributes for Lambda Web Adapter", processedEvents)
	return nil
}