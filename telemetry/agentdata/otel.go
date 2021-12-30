package agentdata

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"time"
)

func ReplaySpans(ctx context.Context, nrSpans []AgentEvent, tracer trace.Tracer) {
	childIndex, root := indexAgentEvents(nrSpans)

	ReplaySpan(ctx, tracer, root, childIndex)
}

func ReplaySpan(ctx context.Context, tracer trace.Tracer, nrSpan *AgentEvent, index map[string][]*AgentEvent) {
	ts := ToTimestamp(nrSpan.Get("timestamp").(float64))
	attrs := nrSpanAttributes(nrSpan)
	spanCtx, span := tracer.Start(ctx, nrSpan.Get("name").(string), trace.WithTimestamp(ts), trace.WithAttributes(attrs...))

	id := nrSpan.Get("guid").(string)
	for _, child := range index[id] {
		ReplaySpan(spanCtx, tracer, child, index)
	}

	durationSeconds := nrSpan.Get("duration").(float64)
	endTs := ts.Add(time.Duration(durationSeconds) * time.Second)

	span.End(trace.WithTimestamp(endTs))
}

func indexAgentEvents(agentEvents []AgentEvent) (map[string][]*AgentEvent, *AgentEvent) {
	ret := make(map[string][]*AgentEvent)

	var root *AgentEvent
	for _, ae := range agentEvents {
		entryPoint := ae.Get("nr.entryPoint").(bool)
		if entryPoint {
			root = &ae
		} else {
			parentId := ae.Get("parentId").(string)
			ret[parentId] = append(ret[parentId], &ae)
		}
	}

	return ret, root
}

func nrSpanAttributes(nrSpan *AgentEvent) []attribute.KeyValue {
	ret := make([]attribute.KeyValue, 0)
	for k, v := range nrSpan.Flatten() {
		attr := toAttribute(k, v)
		if attr.Valid() {
			ret = append(ret, attr)
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
