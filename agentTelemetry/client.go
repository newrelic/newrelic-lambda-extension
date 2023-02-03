package agentTelemetry

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	crypto_rand "crypto/rand"
	math_rand "math/rand"

	"newrelic-lambda-extension/config"
	"newrelic-lambda-extension/util"
)

const (
	InfraEndpointEU string = "https://cloud-collector.eu01.nr-data.net/aws/lambda/v1"
	InfraEndpointUS string = "https://cloud-collector.newrelic.com/aws/lambda/v1"

	// WIP (configuration options?)
	SendTimeoutRetryBase  time.Duration = 200 * time.Millisecond
	SendTimeoutMaxRetries int           = 20
	SendTimeoutMaxBackOff time.Duration = 5 * time.Second
)

type Client struct {
	httpClient        *http.Client
	batch             *Batch
	timeout           time.Duration
	licenseKey        string
	telemetryEndpoint string
	functionName      string
	collectTraceID    bool
}

// New creates a telemetry client with sensible defaults
func New(conf config.Config, batch *Batch, collectTraceID bool) *Client {
	httpClient := &http.Client{
		Timeout: 2400 * time.Millisecond, //TODO: make this much lower once collector repaired
	}

	// Create random seed for timeout to avoid instances created at the same time
	// from creating a wall of retry requests to the collector
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
		log.Fatal("[New Client] cannot seed math/rand package with cryptographically secure random number generator")
	}
	math_rand.Seed(int64(binary.LittleEndian.Uint64(b[:])))

	return NewWithHTTPClient(httpClient, conf.ExtensionName, conf.LicenseKey, conf.AgentTelemetryRegion, batch, collectTraceID, conf.DataCollectionTimeout)
}

// NewWithHTTPClient is just like New, but the HTTP client can be overridden
func NewWithHTTPClient(httpClient *http.Client, functionName string, licenseKey string, telemetryEndpointOverride string, batch *Batch, collectTraceID bool, clientTimeout time.Duration) *Client {
	telemetryEndpoint := getInfraEndpointURL(licenseKey, telemetryEndpointOverride)
	return &Client{
		httpClient:        httpClient,
		licenseKey:        licenseKey,
		telemetryEndpoint: telemetryEndpoint,
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

// SendTelemetry attempts to send telemetry data to new relic and returns an error and a succesful payload count
func (c *Client) SendTelemetry(ctx context.Context, invokedFunctionARN string, telemetry [][]byte) (int, error) {
	start := time.Now()
	logEvents := make([]LogsEvent, 0, len(telemetry))
	for _, payload := range telemetry {
		logEvent := LogsEventForBytes(payload)
		logEvents = append(logEvents, logEvent)
	}

	compressedPayloads, err := CompressedPayloadsForLogEvents(logEvents, c.functionName, invokedFunctionARN)
	if err != nil {
		return 0, err
	}

	var builder requestBuilder = func(buffer *bytes.Buffer) (*http.Request, error) {
		return BuildVortexRequest(ctx, c.telemetryEndpoint, buffer, util.Name, c.licenseKey)
	}

	transmitStart := time.Now()
	successCount, sentBytes := c.sendPayloads(compressedPayloads, builder)
	end := time.Now()
	totalTime := end.Sub(start)
	transmissionTime := end.Sub(transmitStart)
	l.Infof(
		"[SendTelemetry] Sent %d/%d New Relic payload batches successfully in %.3fms (%dms to transmit %.1fkB)",
		successCount,
		len(compressedPayloads),
		float64(totalTime.Microseconds())/1000.0,
		transmissionTime.Milliseconds(),
		float64(sentBytes)/1024.0,
	)

	return successCount, nil
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
			response.Error = fmt.Errorf("[sendPayloads] failed to send data within user defined timeout period: %s", c.timeout.String())
			quit <- true
		case response = <-data:
			timer.Stop()
		}

		if response.Error != nil {
			l.Infof("[sendPayloads] Telemetry client error: %s", response.Error)
			sentBytes -= p.Len()
		} else if response.Response.StatusCode >= 300 {
			l.Infof("[sendPayloads] Telemetry client response: [%s] %s", response.Response.Status, response.ResponseBody)
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
	baseSleepTime := SendTimeoutRetryBase

	for attempts := 0; attempts < SendTimeoutMaxRetries; attempts++ {
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
				return
			}

			// if error is http timeout, retry
			if err, ok := err.(net.Error); ok && err.Timeout() {
				l.Debug("[attemptSend] Retrying after timeout", err)
				time.Sleep(baseSleepTime + time.Duration(math_rand.Intn(400)))

				// double wait time after 3 timed out attempts
				if attempts%3 == 0 {
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
