package otel

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"
	"github.com/newrelic/newrelic-lambda-extension/telemetry/agentdata"
	"github.com/newrelic/newrelic-lambda-extension/util"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"strings"
)

type OtelTelemetrySender struct {
	traceProvider *tracesdk.TracerProvider
}

func NewOtelTelemetrySender(ctx context.Context, licenseKey string, endpoint string, functionName string) OtelTelemetrySender {
	exporter, _ := otlptracegrpc.New(ctx,
		otlptracegrpc.WithHeaders(map[string]string{"api-key": licenseKey}),
		otlptracegrpc.WithEndpoint(endpoint),
	)

	// TODO: Resource config should describe this Lambda function
	//attributes := resource.WithAttributes(
	//	semconv.CloudPlatformAWSLambda,
	//	semconv.FaaSNameKey.String(functionName),
	//)
	//r, err := resource.New(ctx, attributes)
	//if err != nil {
	//	util.Panic(err)
	//}
	//tp := tracesdk.NewTracerProvider(tracesdk.WithSyncer(exporter), tracesdk.WithResource(r))

	tp := tracesdk.NewTracerProvider(tracesdk.WithSyncer(exporter))

	return OtelTelemetrySender{traceProvider: tp}
}

func unmarshalEncodedPayload(encoded string, target interface{}) error {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}

	zipReader, err := gzip.NewReader(bytes.NewBuffer(decoded))
	if err != nil {
		return err
	}

	err = json.NewDecoder(zipReader).Decode(target)
	return err
}

func (o OtelTelemetrySender) SendTelemetry(ctx context.Context, invokedFunctionARN string, telemetry [][]byte) (error, int) {
	tracer := o.traceProvider.Tracer("newrelic-lambda-extension")

	util.Logln("Starting OTLP send")
	count := 0
	for _, buf := range telemetry {
		if strings.Contains(string(buf), "NR_LAMBDA_MONITORING") {
			count++

			parts := make([]interface{}, 0, 4)
			err := json.Unmarshal(buf, &parts)
			if err != nil {
				return err, 0
			}

			var data agentdata.RawData
			version := int(parts[0].(float64))
			if version == 1 {
				var rawAgentData agentdata.RawAgentData
				err = unmarshalEncodedPayload(parts[2].(string), &rawAgentData)
				if err != nil {
					return err, 0
				}
				data = rawAgentData.Data
			} else if version == 2 {
				err = unmarshalEncodedPayload(parts[3].(string), &data)
				if err != nil {
					return err, 0
				}
			}
			ReplaySpans(
				ctx,
				data.SpanEventData.GetAgentEvents(),
				tracer,
				data.ErrorEventData.GetAgentEvents(),
				data.CustomEventData.GetAgentEvents(),
			)
			util.Logln("Finished trace.")
		}
	}

	//util.Logln("Forcing flush")
	//err := o.traceProvider.ForceFlush(ctx)
	//if err != nil {
	//	return err, 0
	//}

	util.Logf("Sent %v traces", count)

	return nil, len(telemetry)
}

func (o OtelTelemetrySender) SendFunctionLogs(ctx context.Context, lines []logserver.LogLine) error {
	//TODO implement me
	return nil
}
