package otel

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"
	"github.com/newrelic/newrelic-lambda-extension/telemetry/agentdata"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"strings"
)

type OtelTelemetrySender struct {
	traceProvider *tracesdk.TracerProvider
}

func NewOtelTelemetrySender(ctx context.Context, licenseKey string, endpoint string) OtelTelemetrySender {
	exporter, _ := otlptracegrpc.New(ctx,
		otlptracegrpc.WithHeaders(map[string]string{"api-key": licenseKey}),
		otlptracegrpc.WithEndpoint(endpoint),
	)

	// TODO: Resource config should describe this Lambda function
	tp := tracesdk.NewTracerProvider(tracesdk.WithBatcher(exporter))

	return OtelTelemetrySender{traceProvider: tp}
}

func (o OtelTelemetrySender) SendTelemetry(ctx context.Context, invokedFunctionARN string, telemetry [][]byte) (error, int) {
	tracer := o.traceProvider.Tracer("newrelic-lambda-extension")

	for _, buf := range telemetry {
		if strings.Contains(string(buf), "NR_LAMBDA_MONITORING") {
			parts := make([]interface{}, 0, 4)
			err := json.Unmarshal(buf, &parts)
			if err != nil {
				return err, 0
			}

			var data agentdata.RawData
			if parts[0] == 1 {
				decoded, err := base64.StdEncoding.DecodeString(parts[2].(string))
				if err != nil {
					return err, 0
				}

				var rawAgentData agentdata.RawAgentData
				err = json.Unmarshal(decoded, &rawAgentData)
				if err != nil {
					return err, 0
				}

				data = rawAgentData.Data
			} else if parts[0] == 2 {
				decoded, err := base64.StdEncoding.DecodeString(parts[3].(string))
				if err != nil {
					return err, 0
				}

				err = json.Unmarshal(decoded, &data)
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
		}
	}

	err := o.traceProvider.ForceFlush(ctx)
	if err != nil {
		return err, 0
	}

	return nil, len(telemetry)
}

func (o OtelTelemetrySender) SendFunctionLogs(ctx context.Context, lines []logserver.LogLine) error {
	//TODO implement me
	return nil
}
