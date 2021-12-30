package agentdata

import "time"

func ToTimestamp(v float64) time.Time {
	// Sometimes, we get seconds rather than ms. This works after May 1970, until November 2286
	if v < 10_000_000_000 {
		v *= 1000
	}

	return time.UnixMilli(int64(v))
}

type RawAgentData struct {
	metadata Metadata
	data     RawData
}

type Metadata struct {
	Arn                  string `json:"arn"`
	ProtocolVersion      int    `json:"protocol_version"`
	FunctionVersion      string `json:"function_version"`
	ExecutionEnvironment string `json:"execution_environment"`
	AgentVersion         string `json:"agent_version"`
	MetadataVersion      int    `json:"metadata_version"`
	AgentLanguage        string `json:"agent_language"`
}

type RawData struct {
	AnalyticEventData RawAgentEventData `json:"analytic_event_data"`
	SpanEventData     RawAgentEventData `json:"span_event_data"`
	ErrorEventData    RawAgentEventData `json:"error_event_data"`
	CustomEventData   RawAgentEventData `json:"custom_event_data"`
	ErrorData         RawErrorData      `json:"error_data"`
	MetricData        RawMetricData     `json:"metric_data"`
}

type RawAgentEventData []interface{}

func (d *RawAgentEventData) GetAgentEvents() []AgentEvent {
	ret := make([]AgentEvent, 0, len(*d))
	for i := 2; i < len(*d); i++ {
		partsList := (*d)[i].([]map[string]interface{})
		var agentAttrs map[string]interface{}
		if len(partsList) > 2 {
			agentAttrs = partsList[2]
		}

		ret = append(ret, AgentEvent{
			Intrinsics:      partsList[0],
			UserAttributes:  partsList[1],
			AgentAttributes: agentAttrs,
		})
	}
	return ret
}

type RawMetricData []interface{}

const maxMetricName int = 255

func (d *RawMetricData) GetMetricData() []MetricData {
	rawMetrics := (*d)[3].([][]interface{})
	ret := make([]MetricData, 0, len(rawMetrics))

	for _, rawMetric := range rawMetrics {
		name := rawMetric[0].(map[string]interface{})["name"].(string)
		if len(name) > maxMetricName {
			name = name[0:255]
		}

		values := rawMetric[1].([]float64)

		ret = append(ret, MetricData{
			Name:   name,
			Values: values,
		})
	}

	return ret
}

type RawErrorData []interface{}

func (d *RawErrorData) GetTracedErrors() []TracedError {
	rawTracedErrors := (*d)[1].([][]interface{})
	ret := make([]TracedError, 0, len(rawTracedErrors))

	for _, rawTracedError := range rawTracedErrors {
		ts := ToTimestamp(rawTracedError[0].(float64))

		name := rawTracedError[1].(string)
		message := rawTracedError[2].(string)
		errorType := rawTracedError[3].(string)
		attrs := rawTracedError[4].(map[string]interface{})

		ret = append(ret, TracedError{
			Timestamp:       ts,
			TransactionName: name,
			Message:         message,
			ErrorType:       errorType,
			Attrs:           attrs,
		})
	}

	return ret
}

type TracedError struct {
	Timestamp       time.Time
	TransactionName string
	Message         string
	ErrorType       string
	Attrs           map[string]interface{}
}

type AgentEvent struct {
	Intrinsics      map[string]interface{}
	UserAttributes  map[string]interface{}
	AgentAttributes map[string]interface{}
}

func (ae *AgentEvent) Get(key string) interface{} {
	v, ok := ae.Intrinsics[key]
	if ok {
		return v
	}
	v, ok = ae.UserAttributes[key]
	if ok {
		return v
	}
	return ae.AgentAttributes[key]
}

func (ae *AgentEvent) Flatten() map[string]interface{} {
	ret := make(map[string]interface{})

	for k, v := range ae.AgentAttributes {
		ret[k] = v
	}
	for k, v := range ae.UserAttributes {
		switch v.(type) {
		case float64:
			ret[k] = v
		case string:
			ret[k] = v
		case bool:
			ret[k] = v
		}
	}
	for k, v := range ae.Intrinsics {
		if k != "type" {
			ret[k] = v
		}
	}
	return ret
}

type MetricData struct {
	Name   string
	Values []float64
}
