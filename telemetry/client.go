package telemetry

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
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

	// WIP (configuration options?)
	SendTimeoutRetryBase  time.Duration = 200 * time.Millisecond
	SendTimeoutMaxRetries int           = 20
	SendTimeoutMaxBackOff time.Duration = 3 * time.Second

	httpClientTimeout time.Duration = 2400 * time.Millisecond
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
		Timeout: httpClientTimeout,
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
	util.Debugf("SendTelemetry: sending telemetry to New Relic...")
	start := time.Now()
	logEvents := make([]LogsEvent, 0, len(telemetry))
	for _, payload := range telemetry {
		logEvent := LogsEventForBytes(payload)
		logEvents = append(logEvents, logEvent)
	}

	util.Debugf("SendTelemetry: compressing telemetry payloads...")
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
		"Sent %d/%d New Relic Telemetry payload batches with %d log events successfully with certainty in %.3fms (%dms to transmit %.1fkB).\n",
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
	sendPayloadsStartTime := time.Now()
	for _, p := range compressedPayloads {
		payloadSize := p.Len()
		sentBytes += payloadSize
		currentPayloadBytes := p.Bytes()

		var response AttemptData

		// buffer this chanel to allow succesful attempts to go through if possible
		data := make(chan AttemptData, 1)
		ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
		defer cancel()

		go c.attemptSend(ctx, currentPayloadBytes, builder, data)

		select {
		case <-ctx.Done():
			response.Error = fmt.Errorf("failed to send data within user defined timeout period: %s", c.timeout.String())
		case response = <-data:
		}

		if response.Error != nil {
			util.Logf("Telemetry client error: %s, payload size: %d bytes", response.Error, payloadSize)
			sentBytes -= payloadSize
		} else if response.Response.StatusCode >= 300 {
			util.Logf("Telemetry client response: [%s] %s", response.Response.Status, response.ResponseBody)
		} else {
			successCount += 1
		}
	}

	util.Debugf("sendPayloads: took %s to finish sending all payloads", time.Since(sendPayloadsStartTime).String())
	return successCount, sentBytes
}

type AttemptData struct {
	Error        error
	ResponseBody string
	Response     *http.Response
}

func (c *Client) attemptSend(ctx context.Context, currentPayloadBytes []byte, builder requestBuilder, dataChan chan AttemptData) {
	baseSleepTime := SendTimeoutRetryBase

	for attempts := 0; attempts < SendTimeoutMaxRetries; attempts++ {
		select {
		case <-ctx.Done():
			util.Debugln("attemptSend: thread was quit by context timeout")
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

				bodyBytes, err := io.ReadAll(res.Body)
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
				util.Debugln("attemptSend: data sent to New Relic succesfully")
				return
			}

			// if error is http timeout, retry
			if err, ok := err.(net.Error); ok && err.Timeout() {
				timeout := baseSleepTime + time.Duration(math_rand.Intn(200))
				util.Debugf("attemptSend: timeout error, retrying after %s: %v", timeout.String(), err)
				time.Sleep(timeout)

				// double wait time after 3 time out attempts
				if (attempts+1)%3 == 0 {
					baseSleepTime *= 2
				}
				if baseSleepTime > SendTimeoutMaxBackOff {
					baseSleepTime = SendTimeoutMaxBackOff
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

// SendFunctionLogs constructs log payloads and sends them to new relic
func (c *Client) SendFunctionLogs(ctx context.Context, invokedFunctionARN string, lines []logserver.LogLine, entityGuid string) error {
	start := time.Now()
	if len(lines) == 0 {
		util.Debugln("client.SendFunctionLogs invoked with 0 log lines. Returning without sending a payload to New Relic")
		return nil
	}

	compressedPayloads, builder, err := c.buildLogPayloads(ctx, invokedFunctionARN, lines, entityGuid)
	if err != nil {
		return err
	}

	transmitStart := time.Now()
	successCount, sentBytes := c.sendPayloads(compressedPayloads, builder)
	totalTime := time.Since(start)
	transmissionTime := time.Since(transmitStart)
	util.Logf(
		"Sent %d/%d New Relic function log batches successfully with certainty in %.3fms (%dms to transmit %.1fkB).\n",
		successCount,
		len(compressedPayloads),
		float64(totalTime.Microseconds())/1000.0,
		transmissionTime.Milliseconds(),
		float64(sentBytes)/1024.0,
	)

	return nil
}

// getNewRelicTags adds tags to the logs if NR_TAGS has values
func getNewRelicTags(common map[string]interface{}) {
    nrTagsStr := os.Getenv("NR_TAGS")
    nrDelimiter := os.Getenv("NR_ENV_DELIMITER")
    if nrDelimiter == "" {
        nrDelimiter = ";"
    }

    if nrTagsStr != "" {
        tags := strings.Split(nrTagsStr, nrDelimiter)
        nrTags := make(map[string]string)
        for _, tag := range tags {
            keyValue := strings.Split(tag, ":")
            if len(keyValue) == 2 {
                nrTags[keyValue[0]] = keyValue[1]
            }
        }

        for k, v := range nrTags {
            common[k] = v
        }
    }
}

// buildLogPayloads is a helper function that improves readability of the SendFunctionLogs method
func (c *Client) buildLogPayloads(ctx context.Context, invokedFunctionARN string, lines []logserver.LogLine, entityGuid string) ([]*bytes.Buffer, requestBuilder, error) {
	common := map[string]interface{}{
		"plugin":    util.Id,
		"faas.arn":  invokedFunctionARN,
		"faas.name": c.functionName,
	}
	if entityGuid != "" {
		common["entity.guid"] = entityGuid
		common["entity.type"] = "APM"
		common["entity.name"] = c.functionName
	}
	getNewRelicTags(common)

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
	}
	// The Log API expects an array
	logData := []DetailedFunctionLog{NewDetailedFunctionLog(common, logMessages)}

	// Since the Log API won't send us more than 1MB, we shouldn't have any issues with payload size.
	compressedPayload, err := CompressedJsonPayload(logData)
	if err != nil {
		return nil, nil, err
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

	return compressedPayloads, builder, nil
}
