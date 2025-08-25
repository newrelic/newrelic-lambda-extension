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
	"strings"
	"sync"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/checks"
	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/telemetry"
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

func PreConnect(cmd RpmCmd, cs *rpmControls) (string, error) {
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

type Label struct {
	LabelType  string `json:"label_type"`
	LabelValue string `json:"label_value"`
}

func getLabels(cmd RpmCmd) []Label {
	lambdaARN := getLambdaARN(cmd)
	labels := []Label{
		{LabelType: "aws.arn", LabelValue: lambdaARN},
		{LabelType: "isLambdaFunction", LabelValue: "true"},
	}
	awsTags := map[string]interface{}{}
	telemetry.GetNewRelicTags(awsTags)
	for k, v := range awsTags {
		labels = append(labels, Label{LabelType: k, LabelValue: fmt.Sprintf("%v", v)})
	}
	return labels
}

type LambdaRuntime string

var (
	NodeLambda   	LambdaRuntime = "node"
	PythonLambda  	LambdaRuntime = "python"
	DotnetLambda 	LambdaRuntime = "dotnet"
	RubyLambda   	LambdaRuntime = "ruby"
	DefaultLambda	LambdaRuntime = "go" 
	runtimeLookupPath     = "/var/lang/bin"
)

var LambdaRuntimes = []LambdaRuntime{NodeLambda, PythonLambda, DotnetLambda, RubyLambda}

func checkRuntime() (LambdaRuntime) {
	for _, runtime := range LambdaRuntimes {
		p := filepath.Join(runtimeLookupPath, string(runtime))
		if util.PathExists(p) {
			return runtime
		}
	}
	return DefaultLambda
}

func getAgentVersion(runtime string) (string, string, error) {
	var layerAgentPaths []string
	var agentVersionFile string
	if runtime == "node" {
		layerAgentPaths = checks.LayerAgentPathNode
		agentVersionFile = "package.json"
	} else if runtime == "python" {
		layerAgentPaths = checks.LayerAgentPathsPython
		agentVersionFile = "version.txt"
	} else if runtime == "dotnet" {
		layerAgentPaths = checks.LayerAgentPathDotnet
		agentVersionFile = "version.txt"
	} 
		

	for i := range layerAgentPaths {
		f := filepath.Join(layerAgentPaths[i], agentVersionFile)
		if !util.PathExists(f) {
			continue
		}

		b, err := os.ReadFile(f)
		if err != nil {
			return "", "", err
		}
		var version string
		if runtime == "python" {
			version = strings.TrimSpace(string(b))
			return "python", version, nil
		} else if runtime == "dotnet" {
			version = strings.TrimSpace(string(b))
			return "dotnet", version, nil
		} else {
			v := checks.LayerAgentVersion{}
			err = json.Unmarshal([]byte(b), &v)
			if err != nil {
				return "", "", err
			}
			return "nodejs", v.Version, nil
		}
	}

	return "", "", fmt.Errorf("agent version file not found in layer paths: %v", layerAgentPaths)
}

func Connect(cmd RpmCmd, cs *rpmControls) (string, string, error) {
	runtimeLanguage := checkRuntime()
	NRAgentLanguage, NRAgentVersion, err := getAgentVersion(string(runtimeLanguage))
	util.Logf("Connect: Detected runtime %s with agent language %s and version %s", runtimeLanguage, NRAgentLanguage, NRAgentVersion)
	if err != nil {
		NRAgentLanguage = "go"
		NRAgentVersion = "3.39.0"
	}
	pid := os.Getpid()
	appName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	if appName == "" {
		return "", "", fmt.Errorf("AWS_LAMBDA_FUNCTION_NAME environment variable not set")
	}
	data := []map[string]interface{}{
		{
			"pid":           pid,
			"language":      NRAgentLanguage,
			"agent_version": NRAgentVersion,
			"host":          "AWS Lambda",
			"app_name":      []string{appName},
			"identifier":    appName,
			"utilization":   getUtilizationData(cmd),
			"labels": 		 getLabels(cmd),
		},
	}
	marshaledData, err := json.Marshal(data)
	util.Debugf("APM Connect Call for runtime %s: %s\n", string(runtimeLanguage), string(marshaledData))
	if err != nil {
		util.Fatal(fmt.Errorf("Extension shutdown: failed to perform APM connect: %w", err))
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


func SendErrorEvent(cmd RpmCmd, cs *rpmControls, errorData []interface{}) {
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
		util.Debugf("Status Code %v telemetry: %d\n", CmdErrorEvents, rpmResponse.GetStatusCode())
		endTimeMetric := time.Now()
		durationMetric := endTimeMetric.Sub(startTimeMetric)
		util.Debugf("Send %v duration: %s\n", CmdErrorEvents, durationMetric)
	}
}

// Function to send data based on the type specified
func sendAPMTelemetryInternal(data []interface{}, dataType string, wg *sync.WaitGroup, runID string, cmd RpmCmd, cs *rpmControls) rpmResponse {
	if len(data) == 0 {
		return *newRPMResponse(nil)
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
		return *newRPMResponse(err)
	}

	cmd.Name = dataType
	cmd.Data = buf.Bytes()
	cmd.RunID = runID

	rpmResponse := CollectorRequest(cmd, cs)

	if rpmResponse == nil {
		log.Printf("No response received for %s telemetry", dataType)
		return *newRPMResponse(fmt.Errorf("no response received"))
	}

	statusCode := rpmResponse.GetStatusCode()
	util.Debugf("Status Code for %s telemetry: %d", dataType, statusCode)
	if err := rpmResponse.GetError(); err != nil {
		util.Logf("Error in telemetry response for %s: %v", dataType, err)
		return *newRPMResponse(err)
	}

	durationMetric := time.Since(startTimeMetric)
	util.Debugf("Send %v duration: %s\n", dataType, durationMetric)

	return *newRPMResponse(nil)
}

func SendAPMTelemetry(ctx context.Context, payload []byte, conf *config.Configuration, cmd RpmCmd, cs *rpmControls, runId string) (error, int) {
	util.Debugf("Send APM Telemetry: sending telemetry to New Relic...")

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

	return sendTelemetryData(ctx, telemetryData, runId, cmd, cs)
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
}, runID string, cmd RpmCmd, cs *rpmControls) (error, int) {
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

func sendSingleTelemetry(task telemetryType, wg *sync.WaitGroup, errChan chan<- error, runID string, cmd RpmCmd, cs *rpmControls) {
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

		if err := sendAPMTelemetryInternal(task.Data, task.DataType, wg, runID, cmd, cs); err.err != nil {
			errChan <- fmt.Errorf("error sending %s telemetry: %w", task.DataType, err.GetError())
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
