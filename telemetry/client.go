package telemetry

import (
	api2 "github.com/newrelic/lambda-extension/lambda/extension/api"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/newrelic/lambda-extension/util"
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

// NewWithHTTPClient is just like New, but the HTTP client can be voerridden
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

// Send sends the payload to the Vortex endpoint
func (c *Client) Send(invocationEvent *api2.InvocationEvent, payload []byte) (*http.Response, string, error) {
	req, err := BuildRequest(payload, invocationEvent, c.functionName, c.licenseKey, c.telemetryEndpoint, "lambda-extension")
	if err != nil {
		return nil, "", err
	}

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
