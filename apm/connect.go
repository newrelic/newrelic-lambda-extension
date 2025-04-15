package apm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

type telemetryType struct {
	Data     []interface{}
	DataType string
}
type PreconnectReply struct {
	Collector string `json:"redirect_host"`
}

func UnmarshalPreConnectReply(body []byte) (*PreconnectReply, error) {
	var preconnect struct {
		ReturnValue PreconnectReply `json:"return_value"`
	}
	if err := json.Unmarshal(body, &preconnect); err != nil {
		return nil, fmt.Errorf("unable to parse pre-connect reply: %w", err)
	}
	return &preconnect.ReturnValue, nil
}

type ConnectReply struct {
	RunID      string `json:"agent_run_id"`
	EntityGUID string `json:"entity_guid"`
}

func UnmarshalConnectReply(body []byte) (*ConnectReply, error) {
	var reply struct {
		ReturnValue *ConnectReply `json:"return_value"`
	}
	if err := json.Unmarshal(body, &reply); err != nil {
		return nil, fmt.Errorf("unable to parse connect reply: %w", err)
	}
	return reply.ReturnValue, nil
}

type preconnectRequest struct {
	SecurityPoliciesToken string `json:"security_policies_token,omitempty"`
	HighSecurity          bool   `json:"high_security"`
}

func PreConnect(cmd RpmCmd, cs *RpmControls) (string, error) {
	//Prepare preconnect data
	preconnectData := []preconnectRequest{{
		SecurityPoliciesToken: "",
		HighSecurity:          false,
	}}
	marshaledData, err := json.Marshal(preconnectData)

	if err != nil {
		return "", fmt.Errorf("failed to marshal preconnect data: %w", err)
	}
	// Set the command's data and name
	cmd.Data = marshaledData
	cmd.Name = cmdPreconnect

	resp := CollectorRequest(cmd, cs)
	if resp == nil {
		return "", fmt.Errorf("no response received from CollectorRequest")
	}

	// Read the response body
	body, err := io.ReadAll(resp.GetBody())
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	preConnectResponse, err := UnmarshalPreConnectReply(body)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal preconnect response: %w", err)
	}
	// Return the collector name from the response
	return preConnectResponse.Collector, nil
}

func getLambdaARN(cmd RpmCmd) string {
	awsLambdaName := cmd.metaData["AWSFunctionName"].(string)
	awsAccountId := cmd.metaData["AWSAccountId"].(string)
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = os.Getenv("AWS_DEFAULT_REGION")
	}
	return fmt.Sprintf("arn:aws:lambda:%s:%s:function:%s", awsRegion, awsAccountId, awsLambdaName)
}

func getUtilizationData(cmd RpmCmd) map[string]interface{} {
	awsLambdaName := cmd.metaData["AWSFunctionName"].(string)
	awsAccountId := cmd.metaData["AWSAccountId"].(string)
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = os.Getenv("AWS_DEFAULT_REGION")
	}
	awsUnqualifiedLambdaARN := getLambdaARN(cmd)
	utilizationData := map[string]interface{}{
		"vendors": map[string]interface{}{
			"awslambda": map[string]interface{}{
				"aws.arn": awsUnqualifiedLambdaARN,
				"aws.region":awsRegion,
				"aws.accountId": awsAccountId,
				"aws.functionName": awsLambdaName,
			},
		},
	}
	return utilizationData
}

type AgentRuntime string

var (
	Node 	AgentRuntime = "node"
	Python 	AgentRuntime = "python"
	Go 		AgentRuntime = "go"
	Dotnet 	AgentRuntime = "dotnet"
	Ruby   	AgentRuntime = "ruby"
	Java 	AgentRuntime = "java"
	runtimeLookupPath     = "/var/lang/bin"
)

var LambdaRuntimes = []AgentRuntime{Node, Python, Go, Dotnet, Ruby, Java}

func checkRuntime() (AgentRuntime) {
	for _, runtime := range LambdaRuntimes {
		p := filepath.Join(runtimeLookupPath, string(runtime))
		if util.PathExists(p) {
			return runtime
		}
	}
	return Go
}

type agentConfig struct {
	language            AgentRuntime
	agentVersion        string
}

var agentRuntimeConfig = map[AgentRuntime]agentConfig{
	Node: {
		language:     Node,
		agentVersion: "12.17.0",
	},
	Python: {
		language:     Python,
		agentVersion: "10.8.1",
	},
	Ruby: {
		language:     Ruby,
		agentVersion: "9.18.0",
	},
	Go: {
		language:     Go,
		agentVersion: "3.38.0",
	},
	Dotnet: {
		language:    Dotnet,
		agentVersion: "10.40.0",
	},
	Java: {
		language:   Java,
		agentVersion: "2.2.0",
	},
}


func Connect(cmd RpmCmd, cs *RpmControls) (string, string, error) {
	runtime := checkRuntime()
	runtimeConfig := agentRuntimeConfig[runtime]
	pid := os.Getpid()
	appName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	if appName == "" {
		return "", "", fmt.Errorf("AWS_LAMBDA_FUNCTION_NAME environment variable not set")
	}
	data := []map[string]interface{}{
		{
			"pid":           pid,
			"language":      runtimeConfig.language,
			"agent_version": runtimeConfig.agentVersion,
			"host":          "AWS Lambda",
			"app_name":      []string{appName},
			"identifier":    appName,
			"utilization":   getUtilizationData(cmd),
		},
	}
	marshaledData, err := json.Marshal(data)
	util.Debugf("Marshalled Data for Connect Call for runtime %s: %s\n", string(runtime), string(marshaledData))
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal connect data: %w", err)
	}
	cmd.Data = marshaledData
	cmd.Name = cmdConnect

	resp := CollectorRequest(cmd, cs)
	if resp == nil {
		return "", "", fmt.Errorf("no response received from CollectorRequest")
	}

	body, err := io.ReadAll(resp.GetBody())
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}
	connectResponse, err := UnmarshalConnectReply(body)
	if err != nil {
		return "", "", fmt.Errorf("failed to unmarshal connect response: %w", err)
	}

	if connectResponse == nil || (connectResponse.EntityGUID == "" && connectResponse.RunID == "") {
		return "", "", fmt.Errorf("connect response unsuccessful: missing required fields")
	}

	cs.SetRunId(connectResponse.RunID)
	SetEntityGuid(connectResponse.EntityGUID)

	return connectResponse.RunID, connectResponse.EntityGUID, nil

}


func SendErrorEvent(cmd RpmCmd, cs *RpmControls, errorData []interface{}) {
	tg := NewTraceIDGenerator(1453)
	spanId := tg.GenerateSpanID()
	traceId := tg.GenerateTraceID()
	guid := tg.GenerateTraceID()
	if len(errorData) > 0 {
		startTimeMetric := time.Now()
		updatedData, _ := MapToErrorEventData(errorData, cs.GetRunId(), spanId, traceId, guid)
		finalData, _ := json.Marshal(updatedData)
		cmd.Name = CmdErrorEvents
		cmd.Data = finalData
		cmd.RunID = cs.GetRunId()
		rpmResponse := CollectorRequest(cmd, cs)
		fmt.Printf("Status Code %v telemetry: %d\n", CmdErrorEvents, rpmResponse.GetStatusCode())
		endTimeMetric := time.Now()
		durationMetric := endTimeMetric.Sub(startTimeMetric)
		fmt.Printf("Send %v duration: %s\n", CmdErrorEvents, durationMetric)
	}
}

// Function to send data based on the type specified
func sendAPMTelemetryInternal(data []interface{}, dataType string, wg *sync.WaitGroup, runID string, cmd RpmCmd, cs *RpmControls) error {
	if len(data) == 0 {
		return nil
	}
	wg.Add(1)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in sendAPMTelemetryInternal: %v", r)
		}
		wg.Done()
	}()

	startTimeMetric := time.Now()
	updatedData := ProcessData(data, runID)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(updatedData); err != nil {
		util.Logf("Error encoding data for %s: %v", dataType, err)
		return err
	}

	cmd.Name = dataType
	cmd.Data = buf.Bytes()
	cmd.RunID = runID

	rpmResponse := CollectorRequest(cmd, cs)

	if rpmResponse == nil {
		log.Printf("No response received for %s telemetry", dataType)
		return fmt.Errorf("no response received for telemetry")
	}

	statusCode := rpmResponse.GetStatusCode()
	util.Debugf("Status Code for %s telemetry: %d", dataType, statusCode)
	if err := rpmResponse.GetError(); err != nil {
		util.Logf("Error in telemetry response for %s: %v", dataType, err)
		return err
	}

	durationMetric := time.Since(startTimeMetric)
	fmt.Printf("Send %v duration: %s\n", dataType, durationMetric)

	return nil
}

func SendAPMTelemetry(ctx context.Context, invokedFunctionARN string, payload []byte, conf *config.Configuration, cmd RpmCmd, cs *RpmControls) (error, int) {
	util.Debugf("Send APM Telemetry: sending telemetry to New Relic...")

	runID := cs.GetRunId()

	// Decode and decompress payload
	datav1, datav2, pv, err := GetServerlessData(payload)
	if err != nil {
		return fmt.Errorf("failed to decode and decompress: %w", err), 0
	}

	// Extract telemetry data based on protocol version
	telemetryData, err := extractTelemetryData(datav1, datav2, pv)
	if err != nil {
		return err, 0
	}

	return sendTelemetryData(ctx, telemetryData, runID, cmd, cs)
}

func extractTelemetryData(datav1 LambdaRawData, datav2 LambdaData, pv int) (struct {
	MetricData     	[]interface{}
	SpanEventData  	[]interface{}
	ErrorData      	[]interface{}
	ErrorEventData 	[]interface{}
	CustomEventData []interface{}
	AnalyticEventData []interface{}
	TransactionSampleData []interface{}
}, error) {
	var telemetryData struct {
		MetricData     	[]interface{}
		SpanEventData  	[]interface{}
		ErrorData      	[]interface{}
		ErrorEventData 	[]interface{}
		CustomEventData []interface{}
		AnalyticEventData []interface{}
		TransactionSampleData []interface{}
	}

	switch pv {
	case 2:
		if reflect.DeepEqual(datav2, LambdaData{}) {
			util.Debugf("SendTelemetry: no telemetry data found in payload")
			return telemetryData, nil
		}
		telemetryData = struct {
			MetricData     []interface{}
			SpanEventData  []interface{}
			ErrorData      []interface{}
			ErrorEventData []interface{}
			CustomEventData []interface{}
			AnalyticEventData []interface{}
			TransactionSampleData []interface{}
		}{
			MetricData:     datav2.MetricData,
			SpanEventData:  datav2.SpanEventData,
			ErrorData:      datav2.ErrorData,
			ErrorEventData: datav2.ErrorEventData,
			CustomEventData: datav2.CustomEventData,
			AnalyticEventData: datav2.AnalyticEventData,
			TransactionSampleData: datav2.TransactionSampleData,
		}
	default: // Assuming default case is for v1 data
		if reflect.DeepEqual(datav1, LambdaRawData{}) {
			util.Debugf("SendTelemetry: no telemetry data found in payload")
			return telemetryData, nil
		}
		telemetryData = struct {
			MetricData     	[]interface{}
			SpanEventData  	[]interface{}
			ErrorData      	[]interface{}
			ErrorEventData 	[]interface{}
			CustomEventData []interface{}
			AnalyticEventData []interface{}
			TransactionSampleData []interface{}
		}{
			MetricData:     datav1.LambdaData.MetricData,
			SpanEventData:  datav1.LambdaData.SpanEventData,
			ErrorData:      datav1.LambdaData.ErrorData,
			ErrorEventData: datav1.LambdaData.ErrorEventData,
			CustomEventData: datav1.LambdaData.CustomEventData,
			AnalyticEventData: datav1.LambdaData.AnalyticEventData,
			TransactionSampleData: datav1.LambdaData.TransactionSampleData,
		}
	}

	return telemetryData, nil
}

func sendTelemetryData(ctx context.Context, data struct {
	MetricData            []interface{}
	SpanEventData         []interface{}
	ErrorData             []interface{}
	ErrorEventData        []interface{}
	CustomEventData	      []interface{}
	AnalyticEventData     []interface{}
	TransactionSampleData []interface{}
}, runID string, cmd RpmCmd, cs *RpmControls) (error, int) {
	// Define telemetry tasks
	telemetryTasks := []telemetryType{
		{data.MetricData, CmdMetrics},
		{data.SpanEventData, CmdSpanEvents},
		{data.ErrorData, CmdErrorData},
		{data.ErrorEventData, CmdErrorEvents},
		{data.CustomEventData, CmdCustomEvents},
		{data.AnalyticEventData, cmdAnalyticEvents},
		{data.TransactionSampleData, cmdTxnTraces},
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(telemetryTasks))

	for _, task := range telemetryTasks {
		sendSingleTelemetry(task, &wg, errChan, runID, cmd, cs)
	}

	// Wait for goroutines or context cancellation
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("operation canceled: %w", ctx.Err()), 0
	case <-done:
	}

	close(errChan)

	// Aggregate errors
	return aggregateErrors(errChan)
}

func sendSingleTelemetry(task telemetryType, wg *sync.WaitGroup, errChan chan<- error, runID string, cmd RpmCmd, cs *RpmControls) {
	if len(task.Data) == 0 {
		util.Debugf("No %s telemetry to send", task.DataType)
		return
	}
	wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in sendAPMTelemetryInternal for %s: %v", task.DataType, r)
			}
			wg.Done()
		}()

		if err := sendAPMTelemetryInternal(task.Data, task.DataType, wg, runID, cmd, cs); err != nil {
			errChan <- fmt.Errorf("error sending %s telemetry: %w", task.DataType, err)
		}
	}()
}

func aggregateErrors(errChan <-chan error) (error, int) {
	var combinedErr error
	for err := range errChan {
		if combinedErr == nil {
			combinedErr = err
		} else {
			combinedErr = fmt.Errorf("%v; %w", combinedErr, err)
		}
	}

	if combinedErr != nil {
		return combinedErr, 0
	}

	log.Print("SendAPMTelemetry: completed sending telemetry to New Relic")
	return nil, 1
}
