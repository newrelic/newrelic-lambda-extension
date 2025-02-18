package telemetryApi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

        "github.com/aws/aws-sdk-go/aws/session"
        "github.com/aws/aws-sdk-go/service/secretsmanager"
        "github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"

	"github.com/google/uuid"
)

const EXTENSION_LAMBDA_VERSION = "1.0"

const (
	LogEndpointEU string = "https://log-api.eu.newrelic.com/log/v1"
	LogEndpointUS string = "https://log-api.newrelic.com/log/v1"

	MetricsEndpointEU string = "https://metric-api.eu.newrelic.com/metric/v1"
	MetricsEndpointUS string = "https://metric-api.newrelic.com/metric/v1"

	EventsEndpointEU string = "https://insights-collector.eu01.nr-data.net/v1/accounts/"
	EventsEndpointUS string = "https://insights-collector.newrelic.com/v1/accounts/"

	TracesEndpointEU string = "https://trace-api.eu.newrelic.com/trace/v1"
	TracesEndpointUS string = "https://trace-api.newrelic.com/trace/v1"
)

var (
        sess = session.Must(session.NewSessionWithOptions(session.Options{
                SharedConfigState: session.SharedConfigEnable,
        }))
        secrets secretsmanageriface.SecretsManagerAPI
)

type licenseKeySecret struct {
        LicenseKey string
}

func init() {
        secrets = secretsmanager.New(sess)
}

func decodeLicenseKey(rawJson *string) (string, error) {
        var lks licenseKeySecret

        err := json.Unmarshal([]byte(*rawJson), &lks)
        if err != nil {
                return "", err
        }
	if lks.LicenseKey == "" {
                return "", fmt.Errorf("malformed license key secret; missing \"LicenseKey\" attribute")
        }

        return lks.LicenseKey, nil
}

func getNewRelicLicenseKey(ctx context.Context) (string, error) {
	sId := "NEW_RELIC_LICENSE_KEY"
	v := os.Getenv("NEW_RELIC_LICENSE_KEY_SECRET")
	if len(v) > 0 {
		sId = v
	}
        secretValueInput := secretsmanager.GetSecretValueInput{SecretId: &sId}
        secretValueOutput, err := secrets.GetSecretValueWithContext(ctx, &secretValueInput)
        if err != nil {
		return "", err
        }
        return decodeLicenseKey(secretValueOutput.SecretString)
}

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
	if strings.Contains(uri, "trace") {
	        req.Header.Set("Data-Format", "newrelic")
		req.Header.Set("Data-Format-Version", "1")
	}
	_, err = d.httpClient.Do(req)

	return err
}

func sendDataToNR(ctx context.Context, logEntries []interface{}, d *Dispatcher) error {

	var lambda_name = d.functionName
	var agent_name = path.Base(os.Args[0])

	// NB "." is not allowed in NR eventType
	var replacer = strings.NewReplacer(".", "_")

	data := make(map[string][]map[string]interface{})
	data["events"] = []map[string]interface{}{}
	data["traces"] = []map[string]interface{}{}
	data["logging"] = []map[string]interface{}{}
	data["metrics"] = []map[string]interface{}{}

	// current logic - terminate processing on an error, can be changed later
	for _, event := range logEntries {
		msInt, err := time.Parse(time.RFC3339, event.(LambdaTelemetryEvent).Time)
		if err != nil {
			return err
		}
		// events
		data["events"] = append(data["events"], map[string]interface{}{
			"timestamp": msInt.UnixMilli(),
			"eventType": "AwsLambdaExtension",
			"extension.name": agent_name,
			"extension.version": EXTENSION_LAMBDA_VERSION,
			"lambda.name": lambda_name,
			"lambda.logevent.type": replacer.Replace(event.(LambdaTelemetryEvent).Type),
		})
		// logs
		if event.(LambdaTelemetryEvent).Record != nil {
			data["logging"] = append(data["logging"], map[string]interface{}{
				"timestamp": msInt.UnixMilli(),
				"message":   event.(LambdaTelemetryEvent).Record,
				"attributes": map[string]map[string]string{
					"plugin" : { "type": "lambda extension"},
					"aws": {
						"lambda.logevent.type": event.(LambdaTelemetryEvent).Type,
						"extension.name": agent_name,
						"extension.version": EXTENSION_LAMBDA_VERSION,
						"lambda.name": lambda_name,
					},
				},
			})

		if reflect.ValueOf(event.(LambdaTelemetryEvent).Record).Kind() == reflect.Map {
			eventRecord := event.(LambdaTelemetryEvent).Record.(map[string]interface{})
		// metrics
			rid := ""
			if v, okk := eventRecord["requestId"].(string); okk {
				rid = v
			}
			if val, ok := eventRecord["metrics"].(map[string]interface{}); ok {
				for key := range val {
					data["metrics"] = append(data["metrics"], map[string]interface{}{
						"name": "aws.telemetry.lambda_ext."+key,
						"value": val[key],
						"timestamp": msInt.UnixMilli(),
						"attributes": map[string]interface{}{
							"lambda.logevent.type": event.(LambdaTelemetryEvent).Type,
							"requestId": rid,
							"extension.name": agent_name,
							"extension.version": EXTENSION_LAMBDA_VERSION,
							"lambda.name": lambda_name,
						},
					})
				}
			}
		// spans
			if val, ok := eventRecord["spans"].([]interface{}); ok {
				for _, span := range val {
					attributes := make(map[string]interface{})
                                        attributes["event"] = event.(LambdaTelemetryEvent).Type
                                        attributes["service.name"] = agent_name
					var start time.Time
					for key,v := range span.(map[string]interface{}) {
						if key == "durationMs" {
                                                        attributes["duration.ms"] = v.(float64)
						} else if key == "start" {
							start, err = time.Parse(time.RFC3339, v.(string))
							if err != nil {
								return err
							}
						} else {
							attributes[key] = v.(string)
						}
					}
					el := map[string]interface{}{
						"trace.id": rid,
						"timestamp": start.UnixMilli(),
						"id": (uuid.New()).String(),
						"attributes": attributes,
					}
					data["traces"] = append(data["traces"], el)
				}
			}
			}
		}
	}
	// data ready
	if len(data) > 0 {
		// send logs
		if len(data["logging"]) > 0 {
			bodyBytes, _ := json.Marshal(data["logging"])
			er := sendBatch(ctx, d, getEndpointURL(d.licenseKey, "logging", ""), bodyBytes)
			if er != nil {
				return er
			}
		}
		// send metrics
		if len(data["metrics"]) > 0 {
			var dataMet[]map[string][]map[string]interface{}
			dataMet = append(dataMet, map[string][]map[string]interface{}{
				"metrics": data["metrics"],
			})
			bodyBytes, _ := json.Marshal(dataMet)
			er := sendBatch(ctx, d, getEndpointURL(d.licenseKey, "metrics", ""), bodyBytes)
			if er != nil {
				return er
			}
		}
		// send events
		if len(data["events"]) > 0 {
			ACCOUNT_ID := os.Getenv("NEW_RELIC_ACCOUNT_ID")
		        if len(ACCOUNT_ID) > 0 {
				bodyBytes, _ := json.Marshal(data["events"])
				er := sendBatch(ctx, d, getEndpointURL(d.licenseKey, "events", "")+ACCOUNT_ID+"/events", bodyBytes)
				if er != nil {
					return er
				}
		        } else {
			        l.Info("NEW_RELIC_ACCOUNT_ID is not set, therefore no events data sent")
			}
		}
		// send traces
		if len(data["traces"]) > 0 {
			var dataTraces[]map[string]interface{}
			dataTraces = append(dataTraces, map[string]interface{}{
				"common": map[string]map[string]string{
					"attributes": {
						"host": "aws.amazon.com",
						"service.name": lambda_name,
					},
				},
				"spans": data["traces"],
			})
			bodyBytes, _ := json.Marshal(dataTraces)
			er := sendBatch(ctx, d, getEndpointURL(d.licenseKey, "traces", ""), bodyBytes)
			if er != nil {
				return er
			}
		}
	}

	return nil // success
}
