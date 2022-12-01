package telemetryApi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"net/http"
	"strconv"
	"github.com/google/uuid"
)

const (
	LogEndpointEU   string = "https://log-api.eu.newrelic.com/log/v1"
	LogEndpointUS   string = "https://log-api.newrelic.com/log/v1"

	MetricsEndpointEU   string = "https://metric-api.eu.newrelic.com/metric/v1"
	MetricsEndpointUS   string = "https://metric-api.newrelic.com/metric/v1"

	EventsEndpointEU   string = "https://insights-collector.eu01.nr-data.net"
	EventsEndpointUS   string = "https://insights-collector.newrelic.com"

	TracesEndpointEU   string = "https://trace-api.eu.newrelic.com/trace/v1"
	TracesEndpointUS   string = "https://trace-api.newrelic.com/trace/v1"
)

func getEndpointURL(licenseKey string, typ string, EndpointOverride string) string {
        if EndpointOverride != "" {
                return EndpointOverride
        }
	switch typ {
		case "logging":
			if strings.HasPrefix(licenseKey, "eu") {
				return LogEndpointEU
			} else {
				return LogEndpointUS
			}
		case "metrics":
			if strings.HasPrefix(licenseKey, "eu") {
				return MetricsEndpointEU
			} else {
				return MetricsEndpointUS
			}
		case "events":
			if strings.HasPrefix(licenseKey, "eu") {
				return EventsEndpointEU
			} else {
				return EventsEndpointUS
			}
		case "traces":
			if strings.HasPrefix(licenseKey, "eu") {
				return TracesEndpointEU
			} else {
				return TracesEndpointUS
			}
	}
	return ""
}

func sendBatch(ctx context.Context, d *Dispatcher, uri string, bodyBytes []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", uri, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
// the headers might be different for different endpoints
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", d.licenseKey)
	_, err = d.httpClient.Do(req)

	return err
}

func sendDataToNR(ctx context.Context, logEntries []interface{}, d *Dispatcher) error {

	// will be replaced later
        var lambda_name = "---"
// should be as below
//        var lambda_name = d.functionName
	var agent_name = path.Base(os.Args[0])

	// NB "." is not allowed in NR eventType
	var replacer = strings.NewReplacer(".", "_")

	data := make(map[string][]string)
//	data := make(map[string][]interface{})
        data["events"] = []string{}
        data["traces"] = []string{}
        data["logging"] = []string{}
        data["metrics"] = []string{}

	for _, event := range logEntries {
//  do some processing and add line to payload
//		payload.AddLogLine(time.Now().UnixMilli(), "debug", "message")

		msInt, err := strconv.ParseInt(event["time"], 10, 64)
		if err != nil {
			return err
		}
		// events
		data["events"] := append(data["events"], fmt.Sprintf("%v",`{
			"timestamp": time.UnixMilli(msInt)
			"eventType": "Lambda_Ext_"+ replacer.Replace(event["type"])
		}`))

		// logging
		if val, ok := event["record"]; ok {
			if len(val) > 0 {
				data["logging"] := append(data["logging"], fmt.Sprintf("%v",`{
					"timestamp": time.UnixMilli(msInt),
					"message": event["record"],
					"attributes": {
						"aws": {
							"event": event["type"],
							"lambda": lambda_name
						}
					}
				}`))
			}
		}
		// metrics
		if reflect.ValueOf(event["record"]).Kind() == reflect.Map && val, ok := event["record"]["metrics"]; ok {
			mts := [...]string{}
			for key := range val {
				mts := appand(mts, fmt.Sprintf("%v",`{
					"name": "aws.telemetry.lambda_ext."+lambda_name+"."+key,
					"value": event["record"]["metrics"][key]
				}`))
			}
			rid := ""
			if val, ok := event["record"]["requestId"]; ok {
				rid = val
			}
			data["metrics"] := append(data["metrics"], fmt.Sprintf("%v",`{
				"common" : {
					"timestamp": time.UnixMilli(msInt),
					"attributes": {
						"event": event["type"],
						"requestId": rid,
						"extension": agent_name
						}
				},
				"metrics": mts
			}`))
		}
		// spans
		if (reflect.ValueOf(event["record"]).Kind() == reflect.Map) && val, ok := event["record"]["spans"]; ok {
			spans := [...]string{}
			for span := range val {
				el := `{
					"trace.id": event["record"]["requestId"],
					"id": uuid.New().String(),
					"attributes": {
						"event": event["type"],
						"service.name": agent_name
						}
				}`
				start, err := strconv.ParseInt(event["start"], 10, 64)
		                if err != nil {
					return err
				}
				for key := range span {
					if (key == "durationMs") {
						el["attributes"]["duration.ms"] := span[key]
					} else if (key =="start") {
						el["timestamp"] := time.UnixMilli(start)
					} else {
						el["attributes"][key] := span[key]
					}
				}
				data["traces"] := append(data["traces"], fmt.Sprintf("%v",el))
			}
		}
	}
	// data ready
	if (len(data) > 0) {
// send logs
		if (len(data["logging"]) > 0) {
//bodyBytes := payload.Marshal()
//bodyBytes, _ := json.Marshal(map[string]string{"message": fmt.Sprintf("%v", logEntries)})
			dt := NewLogPayload(`{
				"common": {
					"attributes": {
						"aws": {
							"logType": "aws/lambda-ext",
							"function": lambda_name,
							"extension": agent_name
							}
						}
				},
				"logs": data["logging"]
			}`)
			bodyBytes := dt.Marshal()
fmt.Println(reflect.TypeOf(bodyBytes))
			err := sendBatch(ctx, d, getEndpointURL(d.licenseKey,"logging"), bodyBytes)
		}
// send metrics
		if (len(data["metrics"]) > 0) {
			for payload := range data["metrics"] {
				bodyBytes := NewLogPayload(payload).Marshal()
				err := sendBatch(ctx, d, getEndpointURL(d.licenseKey,"metrics"), bodyBytes)
			}
		}
// send events
                if (len(data["events"]) > 0) {
			bodyBytes := NewLogPayload(data["events"]).Marshal()
			err := sendBatch(ctx, d, getEndpointURL(d.licenseKey,"events"), bodyBytes)
		}
// send traces
		if (len(data["traces"]) > 0) {
			dt := NewLogPayload(`{
				"common": {
					"attributes": {
						"host": "aws.amazon.com",
						"service.name": lambda_name
					}
				},
				"spans": data["traces"]
			}`)
			bodyBytes := dt.Marshal()
			err := sendBatch(ctx, d, getEndpointURL(d.licenseKey,"traces"), bodyBytes)
		}
	}

        return err // if one of the sents failed, it'd be nice to know which
}
