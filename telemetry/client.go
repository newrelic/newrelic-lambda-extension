package telemetry

import (
	"io/ioutil"
	"net/http"

	"os"
	"strings"
	"time"

	"github.com/newrelic/lambda-extension/api"
	"github.com/newrelic/lambda-extension/util"
)

const (
	InfraEndpointEU string = "https://cloud-collector.eu01.nr-data.net/aws/lambda/v1/"
	InfraEndpointUS string = "https://cloud-collector.newrelic.com/aws/lambda/v1/"
)

type Client struct {
	httpClient           *http.Client
	licenseKey           string
	registrationResponse *api.RegistrationResponse
}

// New creates a telemetry client with sensible defaults
func New(registrationResponse *api.RegistrationResponse, licenseKey string) *Client {
	httpClient := &http.Client{
		Timeout: time.Second * 2,
	}

	return &Client{httpClient, licenseKey, registrationResponse}
}

// NewWithHTTPClient is just like New, but the HTTP client can be voerridden
func NewWithHTTPClient(httpClient *http.Client, registrationResponse *api.RegistrationResponse, licenseKey string) *Client {
	return &Client{httpClient, licenseKey, registrationResponse}
}

// GetInfraEndpointURL returns the Vortex endpoint for the provided license key
func (c *Client) GetInfraEndpointURL() string {
	endpointOverride := os.Getenv("NEWRELIC_INFRA_ENDPOINT")
	if endpointOverride != "" {
		return endpointOverride
	}
	if strings.HasPrefix(c.licenseKey, "eu") {
		return InfraEndpointEU
	}

	return InfraEndpointUS
}

// Send sends the payload to the Vortex endpoint
func (c *Client) Send(invocationEvent *api.InvocationEvent, payload []byte) (*http.Response, string, error) {
	req, err := BuildRequest(payload, invocationEvent, c.registrationResponse, c.licenseKey, c.GetInfraEndpointURL(), "lambda-extension")
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
