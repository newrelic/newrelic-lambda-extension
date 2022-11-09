package telemetry

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	crypto_rand "crypto/rand"
	math_rand "math/rand"

	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

const (
	InfraEndpointEU string = "https://cloud-collector.eu01.nr-data.net/aws/lambda/v1"
	InfraEndpointUS string = "https://cloud-collector.newrelic.com/aws/lambda/v1"
	LogEndpointEU   string = "https://log-api.eu.newrelic.com/log/v1"
	LogEndpointUS   string = "https://log-api.newrelic.com/log/v1"
)

type Client struct {
	httpClient        *http.Client
	batch             *Batch
	timeout           time.Duration
	licenseKey        string
	telemetryEndpoint string
	logEndpoint       string
	functionName      string
	collectTraceID    bool
}

// New creates a telemetry client with sensible defaults
func New(functionName string, licenseKey string, telemetryEndpointOverride string, logEndpointOverride string, batch *Batch, collectTraceID bool, clientTimeout time.Duration) *Client {
	httpClient := &http.Client{
		Timeout: 800 * time.Millisecond,
	}

	// Create random seed for timeout to avoid instances created at the same time
	// from creating a wall of retry requests to the collector
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
		log.Fatal("cannot seed math/rand package with cryptographically secure random number generator")
	}
	math_rand.Seed(int64(binary.LittleEndian.Uint64(b[:])))

	return NewWithHTTPClient(httpClient, functionName, licenseKey, telemetryEndpointOverride, logEndpointOverride, batch, collectTraceID, clientTimeout)
}

// NewWithHTTPClient is just like New, but the HTTP client can be overridden
func NewWithHTTPClient(httpClient *http.Client, functionName string, licenseKey string, telemetryEndpointOverride string, logEndpointOverride string, batch *Batch, collectTraceID bool, clientTimeout time.Duration) *Client {
	telemetryEndpoint := getInfraEndpointURL(licenseKey, telemetryEndpointOverride)
	logEndpoint := getLogEndpointURL(licenseKey, logEndpointOverride)
	return &Client{
		httpClient:        httpClient,
		licenseKey:        licenseKey,
		telemetryEndpoint: telemetryEndpoint,
		logEndpoint:       logEndpoint,
		functionName:      functionName,
		batch:             batch,
		collectTraceID:    collectTraceID,
		timeout:           clientTimeout,
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
	successCount, sentBytes := c.sendPayloads(compressedPayloads, builder)
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

func (c *Client) sendPayloads(compressedPayloads []*bytes.Buffer, builder requestBuilder) (successCount int, sentBytes int) {
	successCount = 0
	sentBytes = 0
	for _, p := range compressedPayloads {
		sentBytes += p.Len()
		currentPayloadBytes := p.Bytes()

		var response AttemptData

		timer := time.NewTimer(c.timeout)

		quit := make(chan bool, 1)
		data := make(chan AttemptData)
		go c.attemptSend(currentPayloadBytes, builder, data, quit)

		select {
		case <-timer.C:
			response.Error = fmt.Errorf("failed to send data within user defined timeout period: %s", c.timeout.String())
			quit <- true
		case response = <-data:
			timer.Stop()
		}

		if response.Error != nil {
			util.Logf("Telemetry client error: %s", response.Error)
			sentBytes -= p.Len()
		} else if response.Response.StatusCode >= 300 {
			util.Logf("Telemetry client response: [%s] %s", response.Response.Status, response.ResponseBody)
		} else {
			successCount += 1
		}
	}

	return successCount, sentBytes
}

type AttemptData struct {
	Error        error
	ResponseBody string
	Response     *http.Response
}

func (c *Client) attemptSend(currentPayloadBytes []byte, builder requestBuilder, dataChan chan AttemptData, quit chan bool) {
	baseSleepTime := 200 * time.Millisecond

	for attempts := 0; ; attempts++ {
		select {
		case <-quit:
			return
		default:
			// Construct request for this try
			req, err := builder(bytes.NewBuffer(currentPayloadBytes))
			if err != nil {
				dataChan <- AttemptData{
					Error: err,
				}
				return
			}
			//Make request, check for timeout
			res, err := c.httpClient.Do(req)

			// send response data and exit
			if err == nil {
				// Success. Process response and exit retry loop
				defer util.Close(res.Body)

				bodyBytes, err := ioutil.ReadAll(res.Body)
				if err != nil {
					dataChan <- AttemptData{
						Error: err,
					}
					return
				}

				// Successfully sent bytes
				dataChan <- AttemptData{
					Error:        nil,
					ResponseBody: string(bodyBytes),
					Response:     res,
				}
				return
			}

			// if error is http timeout, retry
			if err, ok := err.(net.Error); ok && err.Timeout() {
				util.Debugln("Retrying after timeout", err)
				time.Sleep(baseSleepTime + time.Duration(math_rand.Intn(400)))

				// double wait time after 3 timed out attempts
				if attempts%3 == 0 {
					baseSleepTime *= 2
				}
			} else {
				// All other error types are fatal
				dataChan <- AttemptData{
					Error: err,
				}
				return
			}
		}
	}
}

func (c *Client) SendFunctionLogs(ctx context.Context, invokedFunctionARN string, lines []logserver.LogLine) error {
	start := time.Now()

	common := map[string]interface{}{
		"plugin":    util.Id,
		"faas.arn":  invokedFunctionARN,
		"faas.name": c.functionName,
	}

	logMessages := make([]FunctionLogMessage, 0, len(lines))
	for _, l := range lines {
		// Unix time in ms
		ts := l.Time.UnixNano() / 1e6
		var traceId string
		if c.batch != nil && c.collectTraceID {
			// There is a race condition here. Telemetry batch may be late, so the trace
			// ID would be blank. This would require a lock to handle, which would delay
			// logs being sent. Not sure if worth the performance hit yet.
			traceId = c.batch.RetrieveTraceID(l.RequestID)
		}
		logMessages = append(logMessages, NewFunctionLogMessage(ts, l.RequestID, traceId, string(l.Content)))
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
	successCount, sentBytes := c.sendPayloads(compressedPayloads, builder)
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
