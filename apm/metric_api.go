package apm

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

const (
	MetricEndpointEU string = "https://metric-api.eu.newrelic.com/metric/v1"
	MetricEndpointUS string = "https://metric-api.newrelic.com/metric/v1"
)

// Precompile regex patterns at package level for reusability
var (
	basicRe    = regexp.MustCompile(`RequestId: (\S+)\s+Duration: ([\d.]+) ms\s+Billed Duration: (\d+) ms\s+Memory Size: (\d+) MB\s+Max Memory Used: (\d+) MB`)
	initRe     = regexp.MustCompile(`Init Duration: ([\d.]+) ms`)
	faultLogRe = regexp.MustCompile(`RequestId: (\S+)\s+Status: (\S+)(?:\s+ErrorType: (\S+))?`)
)

type Metric struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Value      float64           `json:"value"`
	Timestamp  int64             `json:"timestamp"`
	Attributes map[string]string `json:"attributes"`
	Interval   int64             `json:"interval.ms,omitempty"`
}
type MetricPayload struct {
	Metrics []Metric `json:"metrics"`
}
type LambdaMetrics struct {
	RequestID      string
	Duration       float64
	BilledDuration float64
	MemorySize     int64
	MaxMemoryUsed  int64
	InitDuration   *float64
	Error          string
	ErrorType      string
}

func ParseLambdaFaultLog(logLine string) (*LambdaMetrics, error) {
	matches := faultLogRe.FindStringSubmatch(logLine)
	if matches == nil {
		return nil, fmt.Errorf("error parsing log line: %s", logLine)
	}

	metrics := &LambdaMetrics{
		RequestID: matches[1],
		Error:     matches[2], // Status field repurposed as Error
	}

	// Check for optional ErrorType (index 3)
	if len(matches) > 3 && matches[3] != "" {
		metrics.ErrorType = matches[3]
	}

	return metrics, nil
}

func ParseLambdaReportLog(logLine string) (*LambdaMetrics, error) {
	basicMatches := basicRe.FindStringSubmatch(logLine)
	if basicMatches == nil {
		return ParseLambdaFaultLog(logLine) // Delegate to fault handler
	}

	// Parse fields directly from matches
	duration, err := strconv.ParseFloat(basicMatches[2], 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing duration: %v", err)
	}
	billedDuration, err := strconv.ParseInt(basicMatches[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing billed duration: %v", err)
	}
	memorySize, err := strconv.ParseInt(basicMatches[4], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing memory size: %v", err)
	}
	maxMemoryUsed, err := strconv.ParseInt(basicMatches[5], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing max memory used: %v", err)
	}

	metrics := &LambdaMetrics{
		RequestID:      basicMatches[1],
		Duration:       duration,
		BilledDuration: float64(billedDuration),
		MemorySize:     memorySize,
		MaxMemoryUsed:  maxMemoryUsed,
		InitDuration:   nil,
	}
	// Parse optional init duration
	if initMatches := initRe.FindStringSubmatch(logLine); initMatches != nil {
		if initDuration, err := strconv.ParseFloat(initMatches[1], 64); err == nil {
			metrics.InitDuration = &initDuration
		}
	}

	return metrics, nil
}

func (lm *LambdaMetrics) ConvertToMetrics(prefix string, entityGuid string, functionName string) []Metric {
	timestamp := util.Timestamp()
	attributes := map[string]string{
		"aws.requestId": lm.RequestID,
		"entity.guid":   entityGuid,
		"entity.name":   functionName,
		"entity.type":   "APM",
	}
	// Preallocate slice with estimated capacity to reduce reallocations
	metrics := make([]Metric, 0, 6) // Max possible metrics: 6 (duration, billed, memory, max memory, init, error)
	if lm.Duration != 0 {
		metrics = append(metrics, Metric{
			Name:       prefix + ".duration",
			Type:       "gauge",
			Value:      lm.Duration,
			Timestamp:  timestamp,
			Attributes: attributes,
		})
	}
	if lm.BilledDuration != 0 {
		metrics = append(metrics, Metric{
			Name:       prefix + ".billed_duration",
			Type:       "gauge",
			Value:      lm.BilledDuration,
			Timestamp:  timestamp,
			Attributes: attributes,
		})
	}
	if lm.MemorySize != 0 {
		metrics = append(metrics, Metric{
			Name:       prefix + ".memory_size",
			Type:       "gauge",
			Value:      float64(lm.MemorySize),
			Timestamp:  timestamp,
			Attributes: attributes,
		})
	}

	if lm.MaxMemoryUsed != 0 {
		metrics = append(metrics, Metric{
			Name:       prefix + ".max_memory_used",
			Type:       "gauge",
			Value:      float64(lm.MaxMemoryUsed),
			Timestamp:  timestamp,
			Attributes: attributes,
		})
	}

	// Add init duration metric only if it exists
	if lm.InitDuration != nil {
		metrics = append(metrics, Metric{
			Name:       prefix + ".init_duration",
			Type:       "gauge",
			Value:      *lm.InitDuration,
			Timestamp:  timestamp,
			Attributes: attributes,
		})
	}
	// Add error metric only if it exists
	if lm.Error != "" {
		if lm.ErrorType != "" {
			attributes["Error Type"] = lm.ErrorType
		}
		metrics = append(metrics, Metric{
			Name:       prefix + ".error",
			Type:       "count",
			Value:      1,
			Interval:   10000,
			Timestamp:  timestamp,
			Attributes: attributes,
		})
	}
	return metrics
}

func getMetricEndpointURL(licenseKey string, metricEndpointOverride string) string {
	if metricEndpointOverride != "" {
		return metricEndpointOverride
	}

	if strings.HasPrefix(licenseKey, "eu") {
		return MetricEndpointEU
	}

	return MetricEndpointUS
}

func SendMetrics(apiKey string, metricEndpointOverride string, metrics []Metric, skipTLSVerify bool) (int, string, error) {
	payload := []MetricPayload{
		{
			Metrics: metrics,
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return 0, "", fmt.Errorf("error marshaling JSON: %v", err)
	}
	metricEndpoint := getMetricEndpointURL(apiKey, metricEndpointOverride)
	req, err := http.NewRequest("POST", metricEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, "", fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", apiKey)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLSVerify},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", fmt.Errorf("error reading response: %v", err)
	}
	return resp.StatusCode, string(body), nil
}
