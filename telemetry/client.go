package telemetry

import (
	"bytes"
	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

const (
	InfraEndpointEU string = "https://cloud-collector.eu01.nr-data.net/aws/lambda/v1"
	InfraEndpointUS string = "https://cloud-collector.newrelic.com/aws/lambda/v1"
	LogEndpointEU string = "https://log-api.eu.newrelic.com/log/v1"
	LogEndpointUS string = "https://log-api.newrelic.com/log/v1"
)

type Client struct {
	httpClient        *http.Client
	licenseKey        string
	telemetryEndpoint string
	logEndpoint       string
	functionName      string
}

// New creates a telemetry client with sensible defaults
func New(functionName string, licenseKey string, telemetryEndpointOverride *string, logEndpointOverride *string) *Client {
	httpClient := &http.Client{
		Timeout: time.Second * 2,
	}

	return NewWithHTTPClient(httpClient, functionName, licenseKey, telemetryEndpointOverride, logEndpointOverride)
}

// NewWithHTTPClient is just like New, but the HTTP client can be overridden
func NewWithHTTPClient(httpClient *http.Client, functionName string, licenseKey string, telemetryEndpointOverride *string, logEndpointOverride *string) *Client {
	telemetryEndpoint := getInfraEndpointURL(licenseKey, telemetryEndpointOverride)
	logEndpoint := getLogEndpointURL(licenseKey, logEndpointOverride)
	return &Client{
		httpClient:        httpClient,
		licenseKey:        licenseKey,
		telemetryEndpoint: telemetryEndpoint,
		logEndpoint:       logEndpoint,
		functionName:      functionName,
	}
}

// getInfraEndpointURL returns the Vortex endpoint for the provided license key
func getInfraEndpointURL(licenseKey string, telemetryEndpointOverride *string) string {
	if telemetryEndpointOverride != nil {
		return *telemetryEndpointOverride
	}
	if strings.HasPrefix(licenseKey, "eu") {
		return InfraEndpointEU
	}

	return InfraEndpointUS
}

// getLogEndpointURL returns the Vortex endpoint for the provided license key
func getLogEndpointURL(licenseKey string, logEndpointOverride *string) string {
	if logEndpointOverride != nil {
		return *logEndpointOverride
	}
	if strings.HasPrefix(licenseKey, "eu") {
		return LogEndpointEU
	}

	return LogEndpointUS
}

func (c *Client) SendTelemetry(invokedFunctionARN string, telemetry [][]byte) error {
	start := time.Now()
	logEvents := make([]LogsEvent, 0, len(telemetry))
	for _, payload := range telemetry {
		logEvent := LogsEventForBytes(payload)
		logEvents = append(logEvents, logEvent)
	}

	compressedPayloads, err := CompressedPayloadsForLogEvents(logEvents, c.functionName, invokedFunctionARN)
	if err != nil {
		return err
	}

	var builder requestBuilder = func(buffer *bytes.Buffer) (*http.Request, error) {
		return BuildVortexRequest(c.telemetryEndpoint, buffer, util.Name, c.licenseKey)
	}

	transmitStart := time.Now()
	successCount, sentBytes, err := c.sendPayloads(compressedPayloads, builder)
	end := time.Now()
	totalTime := end.Sub(start)
	transmissionTime := end.Sub(transmitStart)
	util.Logf(
		"Sent %d/%d New Relic payload batches with %d log events successfully in %.3fms (%dms to transmit %.1fkB).\n",
		successCount,
		len(compressedPayloads),
		len(telemetry),
		float64(totalTime.Microseconds())/1000.0,
		transmissionTime.Milliseconds(),
		float64(sentBytes)/1024.0,
	)

	return nil
}

type requestBuilder func(buffer *bytes.Buffer) (*http.Request, error)

func (c *Client) sendPayloads(compressedPayloads []*bytes.Buffer, builder requestBuilder) (successCount int, sentBytes int, err error) {
	successCount = 0
	sentBytes = 0
	for _, p := range compressedPayloads {
		sentBytes += p.Len()
		req, err := builder(p)
		if err != nil {
			return successCount, sentBytes, err
		}
		res, body, err := c.sendRequest(req)
		if err != nil {
			util.Logf("Telemetry client error: %s", err)
			sentBytes -= p.Len()
		} else if res.StatusCode >= 300 {
			util.Logf("Telemetry client response: [%s] %s", res.Status, body)
		} else {
			successCount += 1
		}
	}
	return successCount, sentBytes, nil
}

func (c *Client) SendFunctionLogs(lines []logserver.LogLine) error {
	start := time.Now()

	// TODO: make sensible decisions re: common attrs
	common := map[string]interface{}{
		"plugin": util.Id,
	}
	logMessages := make([]FunctionLogMessage, 0, len(lines))
	for _, l := range lines {
		// Unix time in ms
		ts := l.Time.UnixNano() / 1e6
		logMessages = append(logMessages, NewFunctionLogMessage(ts, l.RequestID, string(l.Content)))
	}
	logData := NewDetailedFunctionLog(common, logMessages)

	// TODO: if necessary, deal with splitting. The buffering settings for the log API should make that unnecessary.
	compressedPayload, err := CompressedJsonPayload(logData)
	if err != nil {
		return err
	}
	compressedPayloads := []*bytes.Buffer{compressedPayload}

	var builder requestBuilder = func(buffer *bytes.Buffer) (*http.Request, error) {
		req, err := BuildVortexRequest(c.logEndpoint, buffer, util.Name, c.licenseKey)
		if err != nil {
			return nil, err
		}

		req.Header.Add("X-Event-Source", "logs")
		return req, err
	}

	transmitStart := time.Now()
	successCount, sentBytes, err := c.sendPayloads(compressedPayloads, builder)
	end := time.Now()
	totalTime := end.Sub(start)
	transmissionTime := end.Sub(transmitStart)
	util.Logf(
		"Sent %d/%d New Relic log batches successfully in %.3fms (%dms to transmit %.1fkB).\n",
		successCount,
		len(compressedPayloads),
		float64(totalTime.Microseconds())/1000.0,
		transmissionTime.Milliseconds(),
		float64(sentBytes)/1024.0,
	)

	return nil
}

func (c *Client) sendRequest(req *http.Request) (*http.Response, string, error) {
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}

	defer util.Close(res.Body)

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}

	return res, string(bodyBytes), nil
}
