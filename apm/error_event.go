package apm

import (
	"time"
)

type EventDetail struct {
	Duration        float64 `json:"duration"`
	ErrorClass      string  `json:"error.class"`
	ErrorExpected   bool    `json:"error.expected"`
	ErrorMessage    string  `json:"error.message"`
	Guid            string  `json:"guid"`
	TransactionGuid string  `json:"nr.transactionGuid"`
	Priority        float64 `json:"priority"`
	Sampled         bool    `json:"sampled"`
	SpanId          string  `json:"spanId"`
	Timestamp       int64   `json:"timestamp"`
	TraceId         string  `json:"traceId"`
	TransactionName string  `json:"transactionName"`
	Type            string  `json:"type"`
}

type Event struct {
	ArtID  string      `json:"art_id"`
	Detail EventDetail `json:"detail"`
}

func MapToErrorEventData(errorData []interface{}, run_id, spanId, traceId, guid  string) ([]interface{}, error) {
	ErrorClass := errorData[0].(string)
	ErrorMessage := errorData[3].(string)
	ErrorMessageForEvent := ErrorClass + ":" + ErrorMessage
	AwsRequestId := errorData[2].(string)
	LambdaFunctionName := errorData[4].(string)
	LambdaAccountId := errorData[5].(string)
	LambdaFunctionVersion := errorData[6].(string)
	TransactionName := "OtherTransaction/Function/" + LambdaFunctionName
	LambadFunctionARN := "arn:aws:lambda:" + "region" + ":" + LambdaAccountId + ":function:" + LambdaFunctionName + ":" + LambdaFunctionVersion
	event := []interface{}{
		run_id,
		map[string]int{
			"events_seen":    1,
			"reservoir_size": 100,
		},
		[][]interface{}{
			{
				EventDetail{
					Duration:        0.1,
					ErrorClass:      ErrorClass,
					ErrorExpected:   false,
					ErrorMessage:    ErrorMessageForEvent,
					Guid:            guid,
					TransactionGuid: guid,
					Priority:        1.5,
					Sampled:         true,
					SpanId:          spanId,
					Timestamp:       time.Now().UnixMilli(),
					TraceId:         traceId,
					TransactionName: TransactionName,
					Type:            "TransactionError",
				},
				struct{}{},
				map[string]interface{}{
					"aws.lambda.arn":             LambadFunctionARN,
					"aws.lambda.functionVersion": LambdaFunctionVersion,
					"aws.requestId":              AwsRequestId,
				},
			},
		},
	}
	return event, nil
}


