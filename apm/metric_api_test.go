package apm

import (
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/util"
	"github.com/stretchr/testify/assert"
)

func TestParseLambdaFaultLog(t *testing.T) {
	tests := []struct {
		name      string
		logLine   string
		want      *LambdaMetrics
		wantError bool
	}{
		{
			name:    "Standard Fault Log With ErrorType",
			logLine: "RequestId: 123abc Status: Error ErrorType: Timeout",
			want: &LambdaMetrics{
				RequestID: "123abc",
				Error:     "Error",
				ErrorType: "Timeout",
			},
			wantError: false,
		},
		{
			name:      "Malformed Log Line",
			logLine:   "This is not a valid log line",
			want:      nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLambdaFaultLog(tt.logLine)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.want.RequestID, got.RequestID)
				assert.Equal(t, tt.want.Error, got.Error)
				assert.Equal(t, tt.want.ErrorType, got.ErrorType)
			}
		})
	}
}

func TestParseLambdaReportLog(t *testing.T) {
	tests := []struct {
		name      string
		logLine   string
		want      *LambdaMetrics
		wantError bool
	}{
		{
			name:    "Standard Report Log",
			logLine: "RequestId: abc123 Duration: 123.45 ms Billed Duration: 200 ms Memory Size: 128 MB Max Memory Used: 64 MB",
			want: &LambdaMetrics{
				RequestID:      "abc123",
				Duration:       123.45,
				BilledDuration: 200,
				MemorySize:     128,
				MaxMemoryUsed:  64,
				InitDuration:   nil,
			},
			wantError: false,
		},
		{
			name:    "Report Log With Init Duration",
			logLine: "RequestId: abc123 Duration: 50.5 ms Billed Duration: 100 ms Memory Size: 256 MB Max Memory Used: 128 MB Init Duration: 250.00 ms",
			want: &LambdaMetrics{
				RequestID:      "abc123",
				Duration:       50.5,
				BilledDuration: 100,
				MemorySize:     256,
				MaxMemoryUsed:  128,
				InitDuration:   func() *float64 { v := 250.00; return &v }(),
			},
			wantError: false,
		},
		{
			name:      "Malformed Log Line",
			logLine:   "Not a valid report log",
			want:      nil,
			wantError: true,
		},
		{
			name:    "Report Log With Zero Values",
			logLine: "RequestId: xyz789 Duration: 0 ms Billed Duration: 0 ms Memory Size: 0 MB Max Memory Used: 0 MB",
			want: &LambdaMetrics{
				RequestID:      "xyz789",
				Duration:       0,
				BilledDuration: 0,
				MemorySize:     0,
				MaxMemoryUsed:  0,
				InitDuration:   nil,
			},
			wantError: false,
		},
		{
			name:    "Report Log With Fault Fallback",
			logLine: "RequestId: 123abc Status: Error ErrorType: Timeout",
			want: &LambdaMetrics{
				RequestID: "123abc",
				Error:     "Error",
				ErrorType: "Timeout",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLambdaReportLog(tt.logLine)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.want.RequestID, got.RequestID)
				assert.Equal(t, tt.want.Duration, got.Duration)
				assert.Equal(t, tt.want.BilledDuration, got.BilledDuration)
				assert.Equal(t, tt.want.MemorySize, got.MemorySize)
				assert.Equal(t, tt.want.MaxMemoryUsed, got.MaxMemoryUsed)
				if tt.want.InitDuration != nil {
					assert.NotNil(t, got.InitDuration)
					assert.InDelta(t, *tt.want.InitDuration, *got.InitDuration, 0.0001)
				} else {
					assert.Nil(t, got.InitDuration)
				}
				assert.Equal(t, tt.want.Error, got.Error)
				assert.Equal(t, tt.want.ErrorType, got.ErrorType)
			}
		})
	}
}

func TestLambdaMetrics_ConvertToMetrics(t *testing.T) {
	prefix := "lambda"
	entityGuid := "test-guid"
	functionName := "test-func"

	timestamp := util.Timestamp()

	tests := []struct {
		name   string
		input  LambdaMetrics
		verify func(t *testing.T, metrics []Metric)
	}{
		{
			name: "All fields set, including InitDuration and ErrorType",
			input: LambdaMetrics{
				RequestID:      "req-1",
				Duration:       100.5,
				BilledDuration: 200,
				MemorySize:     128,
				MaxMemoryUsed:  64,
				InitDuration:   func() *float64 { v := 50.0; return &v }(),
				Error:          "Error",
				ErrorType:      "Timeout",
			},
			verify: func(t *testing.T, metrics []Metric) {
				assert.Len(t, metrics, 6)
				names := []string{}
				for _, m := range metrics {
					names = append(names, m.Name)
				}
				assert.Contains(t, names, "lambda.duration")
				assert.Contains(t, names, "lambda.billed_duration")
				assert.Contains(t, names, "lambda.memory_size")
				assert.Contains(t, names, "lambda.max_memory_used")
				assert.Contains(t, names, "lambda.init_duration")
				assert.Contains(t, names, "lambda.error")

				for _, m := range metrics {
					assert.Equal(t, entityGuid, m.Attributes["entity.guid"])
					assert.Equal(t, functionName, m.Attributes["entity.name"])
					assert.Equal(t, "APM", m.Attributes["entity.type"])
					assert.Equal(t, "req-1", m.Attributes["aws.requestId"])
					assert.InDelta(t, float64(timestamp), float64(m.Timestamp), 10)
				}
				// Check error metric
				for _, m := range metrics {
					if m.Name == "lambda.error" {
						assert.Equal(t, "count", m.Type)
						assert.Equal(t, float64(1), m.Value)
						assert.Equal(t, int64(10000), m.Interval)
						assert.Equal(t, "Timeout", m.Attributes["Error Type"])
					}
				}
			},
		},
		{
			name: "Only duration and memory metrics, no error or init",
			input: LambdaMetrics{
				RequestID:      "req-2",
				Duration:       10,
				BilledDuration: 20,
				MemorySize:     256,
				MaxMemoryUsed:  128,
			},
			verify: func(t *testing.T, metrics []Metric) {
				assert.Len(t, metrics, 4)
				names := []string{}
				for _, m := range metrics {
					names = append(names, m.Name)
				}
				assert.ElementsMatch(t, names, []string{
					"lambda.duration",
					"lambda.billed_duration",
					"lambda.memory_size",
					"lambda.max_memory_used",
				})
				for _, m := range metrics {
					assert.Equal(t, "req-2", m.Attributes["aws.requestId"])
					assert.NotContains(t, m.Attributes, "Error Type")
				}
			},
		},
		{
			name: "Only error metric, no other values",
			input: LambdaMetrics{
				RequestID: "req-3",
				Error:     "Error",
			},
			verify: func(t *testing.T, metrics []Metric) {
				assert.Len(t, metrics, 1)
				m := metrics[0]
				assert.Equal(t, "lambda.error", m.Name)
				assert.Equal(t, "count", m.Type)
				assert.Equal(t, float64(1), m.Value)
				assert.Equal(t, int64(10000), m.Interval)
				assert.Equal(t, "req-3", m.Attributes["aws.requestId"])
			},
		},
		{
			name: "Zero values for all fields",
			input: LambdaMetrics{
				RequestID: "req-4",
			},
			verify: func(t *testing.T, metrics []Metric) {
				assert.Len(t, metrics, 0)
			},
		},
		{
			name: "InitDuration only",
			input: LambdaMetrics{
				RequestID:    "req-5",
				InitDuration: func() *float64 { v := 123.45; return &v }(),
			},
			verify: func(t *testing.T, metrics []Metric) {
				assert.Len(t, metrics, 1)
				m := metrics[0]
				assert.Equal(t, "lambda.init_duration", m.Name)
				assert.Equal(t, float64(123.45), m.Value)
				assert.Equal(t, "req-5", m.Attributes["aws.requestId"])
			},
		},
		{
			name: "Error metric without ErrorType",
			input: LambdaMetrics{
				RequestID: "req-6",
				Error:     "Failed",
			},
			verify: func(t *testing.T, metrics []Metric) {
				assert.Len(t, metrics, 1)
				m := metrics[0]
				assert.Equal(t, "lambda.error", m.Name)
				assert.NotContains(t, m.Attributes, "Error Type")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := tt.input.ConvertToMetrics(prefix, entityGuid, functionName)
			tt.verify(t, metrics)
		})
	}
}

func Test_getMetricEndpointURL(t *testing.T) {
	tests := []struct {
		name                 string
		licenseKey           string
		metricEndpointOverride string
		want                 string
	}{
		{
			name:       "Override provided",
			licenseKey: "eu1234567890abcdef",
			metricEndpointOverride: "https://custom-endpoint.com/metric/v1",
			want:       "https://custom-endpoint.com/metric/v1",
		},
		{
			name:       "EU license key, no override",
			licenseKey: "eu01xx1234567890abcdef",
			metricEndpointOverride: "",
			want:       MetricEndpointEU,
		},
		{
			name:       "US license key, no override",
			licenseKey: "us01xx1234567890abcdef",
			metricEndpointOverride: "",
			want:       MetricEndpointUS,
		},
		{
			name:       "Non-EU, non-US license key, no override",
			licenseKey: "xx01xx1234567890abcdef",
			metricEndpointOverride: "",
			want:       MetricEndpointUS,
		},
		{
			name:       "Empty license key, no override",
			licenseKey: "",
			metricEndpointOverride: "",
			want:       MetricEndpointUS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMetricEndpointURL(tt.licenseKey, tt.metricEndpointOverride)
			assert.Equal(t, tt.want, got)
		})
	}
}
