package apm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

type PreconnectReply struct {
	Collector        string           `json:"redirect_host"`
}

func UnmarshalPreConnectReply(body []byte) (*PreconnectReply, error) {
	var preconnect struct {
		Preconnect PreconnectReply `json:"return_value"`
	}
	err := json.Unmarshal(body, &preconnect)
	if nil != err {
		return nil, fmt.Errorf("unable to parse pre-connect reply: %v", err)
	}
	return &preconnect.Preconnect, nil
}

type ConnectReply struct {
	RunID                 string        `json:"agent_run_id"`
	EntityGUID            string            `json:"entity_guid"`
}


func UnmarshalConnectReply(body []byte) (*ConnectReply, error) {
	var reply struct {
		Reply *ConnectReply `json:"return_value"`
	}
	err := json.Unmarshal(body, &reply)
	if nil != err {
		return nil, fmt.Errorf("unable to parse connect reply: %v", err)
	}
	return reply.Reply, nil
}

type preconnectRequest struct {
	SecurityPoliciesToken string `json:"security_policies_token,omitempty"`
	HighSecurity          bool   `json:"high_security"`
}

func PreConnect(cmd RpmCmd, cs *RpmControls) string{
	preconnectData, _ := json.Marshal([]preconnectRequest{{
		SecurityPoliciesToken: "",
		HighSecurity:          false,
	}})
	cmd.Data = preconnectData
	cmd.Name = cmdPreconnect
	resp := CollectorRequest(cmd, cs)
	body, _ := io.ReadAll(resp.GetBody())
	preConnectReponse, _:= UnmarshalPreConnectReply(body)
	return preConnectReponse.Collector
}

func Connect(cmd RpmCmd, cs *RpmControls) (string, string) {

	pid := os.Getpid()
	AppName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	data := []map[string]interface{}{
		{
			"pid":           pid,
			"language":      "go",
			"agent_version": "3.35.1",
			"host":          "AWS Lambda",
			"app_name":      []string{AppName},
			"identifier":     AppName,
		},
	}
	cmd.Data, _ = json.Marshal(data)
	cmd.Name = cmdConnect
	resp := CollectorRequest(cmd, cs)

	body, _ := io.ReadAll(resp.GetBody())
	connectReponse, _:= UnmarshalConnectReply(body)
	if connectReponse != nil  {
		if connectReponse.EntityGUID == "" && connectReponse.RunID == "" {
			fmt.Println("Connect Response Unsuccessful")
			
		}
	}
	cs.SetRunId(connectReponse.RunID)
	SetEntityGuid(connectReponse.EntityGUID)
	return connectReponse.RunID, connectReponse.EntityGUID

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
func sendAPMTelemetryInternal(data []interface{}, dataType string, wg *sync.WaitGroup, run_id string, cmd RpmCmd, cs *RpmControls) {
	defer wg.Done()
	if len(data) > 0 {
		startTimeMetric := time.Now()
		updatedData := ProcessData(data, run_id)
		finalData, _ := json.Marshal(updatedData)
		cmd.Name = dataType
		cmd.Data = finalData
		cmd.RunID = run_id
		rpmResponse := CollectorRequest(cmd, cs)

		fmt.Printf("Status Code %v telemetry: %d\n", dataType, rpmResponse.GetStatusCode())
		endTimeMetric := time.Now()
		durationMetric := endTimeMetric.Sub(startTimeMetric)
		fmt.Printf("Send %v duration: %s\n", dataType, durationMetric)
	}
}

func SendAPMTelemetry(ctx context.Context, invokedFunctionARN string, payload []byte, conf *config.Configuration, cmd RpmCmd, cs *RpmControls) (error, int) {
	util.Debugf("Send APM Telemetry: sending telemetry to New Relic...")
	
	run_id := cs.GetRunId()
	
	data, err := GetServerlessData(payload)
	if err != nil {
		log.Fatalf("failed to decode and decompress: %v", err)
	}
	if reflect.DeepEqual(data, LambdaRawData{}) {
		util.Debugf("SendTelemetry: no telemetry data found in payload")
		return nil, 1
	}
	var wg sync.WaitGroup
	wg.Add(4)
	go sendAPMTelemetryInternal(data.LambdaData.MetricData, CmdMetrics, &wg, run_id, cmd, cs)
	go sendAPMTelemetryInternal(data.LambdaData.SpanEventData, CmdSpanEvents, &wg, run_id, cmd, cs)
	go sendAPMTelemetryInternal(data.LambdaData.ErrorData, CmdErrorData, &wg, run_id, cmd, cs)
	go sendAPMTelemetryInternal(data.LambdaData.ErrorEventData, CmdErrorEvents, &wg, run_id, cmd, cs)
	wg.Wait()
	return nil, 1
}
