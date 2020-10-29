package telemetry

import (
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

const (
	InfraEndpointEU string = "https://cloud-collector.eu01.nr-data.net/aws/lambda/v1"
	InfraEndpointUS string = "https://cloud-collector.newrelic.com/aws/lambda/v1"
)

type Client struct {
	httpClient        *http.Client
	licenseKey        string
	telemetryEndpoint string
	functionName      string
}

// New creates a telemetry client with sensible defaults
func New(functionName string, licenseKey string, telemetryEndpointOverride *string) *Client {
	httpClient := &http.Client{
		Timeout: time.Second * 2,
	}

	endpoint := getInfraEndpointURL(licenseKey, telemetryEndpointOverride)
	return &Client{httpClient, licenseKey, endpoint, functionName}
}

// NewWithHTTPClient is just like New, but the HTTP client can be overridden
func NewWithHTTPClient(httpClient *http.Client, functionName string, licenseKey string, telemetryEndpointOverride *string) *Client {
	endpoint := getInfraEndpointURL(licenseKey, telemetryEndpointOverride)
	return &Client{httpClient, licenseKey, endpoint, functionName}
}

// GetInfraEndpointURL returns the Vortex endpoint for the provided license key
func getInfraEndpointURL(licenseKey string, telemetryEndpointOverride *string) string {
	if telemetryEndpointOverride != nil {
		return *telemetryEndpointOverride
	}
	if strings.HasPrefix(licenseKey, "eu") {
		return InfraEndpointEU
	}

	return InfraEndpointUS
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

	successCount := 0
	transmitStart := time.Now()
	sentBytes := 0
	for _, p := range compressedPayloads {
		sentBytes += p.Len()
		req, err := BuildVortexRequest(err, c.telemetryEndpoint, p, "newrelic-lambda-extension", c.licenseKey)
		if err != nil {
			return err
		}
		res, body, err := c.sendRequest(req)
		if err != nil {
			log.Printf("Telemetry client error: %s", err)
		} else if res.StatusCode >= 300 {
			log.Printf("Telemetry client response: [%s] %s", res.Status, body)
		} else {
			successCount += 1
		}
	}
	end := time.Now()
	totalTime := end.Sub(start)
	transmissionTime := end.Sub(transmitStart)
	log.Printf(
		"Sent %d/%d New Relic payload batches with %d log events successfully in %.3fms (%dms to transmit %.1fkB).\n",
		successCount,
		len(compressedPayloads),
		len(telemetry),
		float64(totalTime.Microseconds()) / 1000.0,
		transmissionTime.Milliseconds(),
		float64(sentBytes) / 1024.0,
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
