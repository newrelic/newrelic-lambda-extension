package agentdata

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"time"
)

func foo(ctx context.Context, licenseKey string) trace.Tracer {
	exporter, _ := otlptracegrpc.New(ctx,
		otlptracegrpc.WithHeaders(map[string]string{"api-key": licenseKey}),
		otlptracegrpc.WithEndpoint("https://otlp.nr-data.net:4317/"),
	)

	tp := tracesdk.NewTracerProvider(tracesdk.WithBatcher(exporter))
	return tp.Tracer("newrelic-lambda-extension")
}

func ReplaySpans(ctx context.Context, nrSpans []AgentEvent, tracer trace.Tracer) {
	childIndex, root := indexAgentEvents(nrSpans)

	ReplaySpan(ctx, tracer, root, childIndex, nil, nil)
}

func ReplaySpan(
	ctx context.Context,
	tracer trace.Tracer,
	nrSpan *AgentEvent,
	index map[string][]*AgentEvent,
	nrErrorEvents []*AgentEvent,
	nrCustomEvents []*AgentEvent,
) {
	ts := ToTimestamp(nrSpan.Get("timestamp").(float64))
	entryPoint := nrSpan.Get("nr.entryPoint")
	spanKind := trace.SpanKindInternal
	if entryPoint != nil && entryPoint.(bool) {
		spanKind = trace.SpanKindServer
	} else {
		kindStr := nrSpan.Get("span.kind")
		if kindStr != nil && kindStr.(string) == "client" {
			spanKind = trace.SpanKindClient
		}
	}
	attrs := agentEventAttributes(nrSpan)
	spanCtx, span := tracer.Start(
		ctx,
		nrSpan.Get("name").(string),
		trace.WithTimestamp(ts),
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(spanKind),
	)

	//Errors
	for _, ev := range nrErrorEvents {
		eventTs := ToTimestamp(ev.Get("timestamp").(float64))
		errorClass := ev.Get("error.class")
		errorMessage := ev.Get("error.message")

		eventAttrs := agentEventAttributes(ev)
		if errorClass != nil {
			eventAttrs = append(eventAttrs, attribute.String("exception.type", errorClass.(string)))
		}
		if errorMessage != nil {
			eventAttrs = append(eventAttrs, attribute.String("exception.message", errorMessage.(string)))
		}

		var err error
		if errorClass != nil && errorMessage != nil {
			err = fmt.Errorf("%v: %v", errorClass, errorMessage)
		} else if errorClass != nil {
			err = fmt.Errorf("%v", errorClass)
		} else if errorMessage != nil {
			err = fmt.Errorf("%v", errorMessage)
		} else {
			err = fmt.Errorf("an unknown error occurred")
		}

		span.RecordError(err, trace.WithTimestamp(eventTs), trace.WithAttributes(eventAttrs...))
	}

	//Custom events
	for _, ev := range nrCustomEvents {
		name := ev.Get("type").(string)
		eventTs := ToTimestamp(ev.Get("timestamp").(float64))
		eventAttrs := agentEventAttributes(ev)
		span.AddEvent(name, trace.WithTimestamp(eventTs), trace.WithAttributes(eventAttrs...))
	}

	id := nrSpan.Get("guid").(string)
	for _, child := range index[id] {
		ReplaySpan(spanCtx, tracer, child, index, nil, nil)
	}

	durationSeconds := nrSpan.Get("duration").(float64)
	endTs := ts.Add(time.Duration(durationSeconds) * time.Second)

	span.End(trace.WithTimestamp(endTs))
}

func indexAgentEvents(agentEvents []AgentEvent) (map[string][]*AgentEvent, *AgentEvent) {
	ret := make(map[string][]*AgentEvent)

	var root *AgentEvent
	for _, ae := range agentEvents {
		entryPoint := ae.Get("nr.entryPoint")
		if entryPoint != nil && entryPoint.(bool) {
			root = &ae
		} else {
			parentId := ae.Get("parentId").(string)
			ret[parentId] = append(ret[parentId], &ae)
		}
	}

	return ret, root
}

// attrFilter is a set of attributes that should not be copied across to OTel spans
var attrFilter = map[string]bool{
	"timestamp": true,
	"guid":      true,
	"duration":  true,
	"name":      true,
	"parentId":  true,
	"type":      true,
	"sampled":   true,
	"priority":  true,
}

func agentEventAttributes(nrSpan *AgentEvent) []attribute.KeyValue {
	ret := make([]attribute.KeyValue, 0)
	for k, v := range nrSpan.Flatten() {
		if !attrFilter[k] {
			attr := toAttribute(k, v)
			if attr.Valid() {
				ret = append(ret, attr)
			}
		}
	}
	return ret
}

func toAttribute(k string, v interface{}) attribute.KeyValue {
	var ret attribute.KeyValue
	switch v.(type) {
	case float64:
		ret = attribute.Float64(k, v.(float64))
	case string:
		ret = attribute.String(k, v.(string))
	case bool:
		ret = attribute.Bool(k, v.(bool))
	}
	return ret
}
