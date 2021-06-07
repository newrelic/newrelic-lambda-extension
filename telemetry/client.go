package telemetry

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

const (
	InfraEndpointEU string = "https://cloud-collector.eu01.nr-data.net/aws/lambda/v1"
	InfraEndpointUS string = "https://cloud-collector.newrelic.com/aws/lambda/v1"
	LogEndpointEU   string = "https://log-api.eu.newrelic.com/log/v1"
	LogEndpointUS   string = "https://log-api.newrelic.com/log/v1"

	retries int = 3
)

type Client struct {
	httpClient        *http.Client
	licenseKey        string
	telemetryEndpoint string
	logEndpoint       string
	functionName      string
}

// New creates a telemetry client with sensible defaults
func New(functionName string, licenseKey string, telemetryEndpointOverride string, logEndpointOverride string) *Client {
	httpClient := &http.Client{
		Timeout: time.Second * 2,
	}

	return NewWithHTTPClient(httpClient, functionName, licenseKey, telemetryEndpointOverride, logEndpointOverride)
}

// NewWithHTTPClient is just like New, but the HTTP client can be overridden
func NewWithHTTPClient(httpClient *http.Client, functionName string, licenseKey string, telemetryEndpointOverride string, logEndpointOverride string) *Client {
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
func getInfraEndpointURL(licenseKey string, telemetryEndpointOverride string) string {
	if telemetryEndpointOverride != "" {
		return telemetryEndpointOverride
	}

	if strings.HasPrefix(licenseKey, "eu") {
		return InfraEndpointEU
	}

	return InfraEndpointUS
}

// getLogEndpointURL returns the Vortex endpoint for the provided license key
func getLogEndpointURL(licenseKey string, logEndpointOverride string) string {
	if logEndpointOverride != "" {
		return logEndpointOverride
	}

	if strings.HasPrefix(licenseKey, "eu") {
		return LogEndpointEU
	}

	return LogEndpointUS
}

func (c *Client) SendTelemetry(ctx context.Context, invokedFunctionARN string, telemetry [][]byte) (error, int) {
	start := time.Now()
	logEvents := make([]LogsEvent, 0, len(telemetry))
	for _, payload := range telemetry {
		logEvent := LogsEventForBytes(payload)
		logEvents = append(logEvents, logEvent)
	}

	compressedPayloads, err := CompressedPayloadsForLogEvents(logEvents, c.functionName, invokedFunctionARN)
	if err != nil {
		return err, 0
	}

	var builder requestBuilder = func(buffer *bytes.Buffer) (*http.Request, error) {
		return BuildVortexRequest(ctx, c.telemetryEndpoint, buffer, util.Name, c.licenseKey)
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

	return nil, successCount
}

type requestBuilder func(buffer *bytes.Buffer) (*http.Request, error)

func (c *Client) sendPayloads(compressedPayloads []*bytes.Buffer, builder requestBuilder) (successCount int, sentBytes int, err error) {
	successCount = 0
	sentBytes = 0
	for _, p := range compressedPayloads {
		sentBytes += p.Len()
		currentPayloadBytes := p.Bytes()

		var res *http.Response
		var err error
		var responseBody string
		for attemptNum := 1; attemptNum <= retries; attemptNum++ {
			// Construct request for this try
			var req *http.Request
			req, err = builder(bytes.NewBuffer(currentPayloadBytes))
			if err != nil {
				break
			}
			//Make request, check for timeout
			res, err = c.httpClient.Do(req)
			if err == nil {
				// Success. Process response and exit retry loop
				defer util.Close(res.Body)

				bodyBytes, err := ioutil.ReadAll(res.Body)
				if err != nil {
					break
				}

				responseBody = string(bodyBytes)
				break
			} else {
				switch err.(type) {
				case *url.Error:
					// Retry on timeout
					if err.(*url.Error).Timeout() {
						if attemptNum < retries {
							util.Debugln("Retrying after timeout", err)
						} else {
							util.Logf("Request failed. Ran out of retries after %v attempts.", attemptNum)
							//We'll exit the loop naturally at this point
						}
					} else {
						//Other errors are fatal
						break
					}
				default:
					//Other errors are fatal
					break
				}
			}
		}

		if err != nil {
			util.Logf("Telemetry client error: %s", err)
			sentBytes -= p.Len()
		} else if res.StatusCode >= 300 {
			util.Logf("Telemetry client response: [%s] %s", res.Status, responseBody)
		} else {
			successCount += 1
		}
	}

	return successCount, sentBytes, nil
}

func (c *Client) SendFunctionLogs(ctx context.Context, lines []logserver.LogLine) error {
	start := time.Now()

	common := map[string]interface{}{
		"plugin":    util.Id,
		"faas.name": c.functionName,
	}
	logMessages := make([]FunctionLogMessage, 0, len(lines))
	for _, l := range lines {
		// Unix time in ms
		ts := l.Time.UnixNano() / 1e6
		logMessages = append(logMessages, NewFunctionLogMessage(ts, l.RequestID, string(l.Content)))
		util.Debugf("Sending function logs for request %s", l.RequestID)
	}
	// The Log API expects an array
	logData := []DetailedFunctionLog{NewDetailedFunctionLog(common, logMessages)}

	// Since the Log API won't send us more than 1MB, we shouldn't have any issues with payload size.
	compressedPayload, err := CompressedJsonPayload(logData)
	if err != nil {
		return err
	}
	compressedPayloads := []*bytes.Buffer{compressedPayload}

	var builder requestBuilder = func(buffer *bytes.Buffer) (*http.Request, error) {
		req, err := BuildVortexRequest(ctx, c.logEndpoint, buffer, util.Name, c.licenseKey)
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
		"Sent %d/%d New Relic function log batches successfully in %.3fms (%dms to transmit %.1fkB).\n",
		successCount,
		len(compressedPayloads),
		float64(totalTime.Microseconds())/1000.0,
		transmissionTime.Milliseconds(),
		float64(sentBytes)/1024.0,
	)

	return nil
}
