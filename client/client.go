// Package client is a generic client for the AWS Lambda Extension API.
// The API's lifecycle begins with execution of the extension binary, which is expected to register.
// The extension then makes blocking requests for the next event. The response to the next event request
// is either a notification of the next event, or a shutdown notification.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/newrelic/lambda-extension/api"
	"github.com/newrelic/lambda-extension/util"
)

// InvocationClient is used to poll for invocation events. It is produced as a result of successful
// registration. The zero value is not usable.
type InvocationClient struct {
	version     string
	baseUrl     string
	httpClient  http.Client
	extensionId string
}

// RegistrationClient is used to register, and acquire an InvocationClient. The zero value is not usable.
type RegistrationClient struct {
	extensionName string
	version       string
	baseUrl       string
	httpClient    http.Client
}

// Constructs a new RegistrationClient. This is the entry point.
func New(httpClient http.Client) *RegistrationClient {
	exeName := filepath.Base(os.Args[0])

	return &RegistrationClient{
		extensionName: exeName,
		version:       api.Version,
		baseUrl:       os.Getenv(api.LambdaHostPortEnvVar),
		httpClient:    httpClient,
	}
}

// GetRegisterURL returns the Lambda Extension register URL
func (rc *RegistrationClient) GetRegisterURL() string {
	return fmt.Sprintf("http://%s/%s/extension/register", rc.baseUrl, rc.version)
}

// RegisterDefault registers for Invoke and Shutdown events, with no configuration parameters.
func (rc *RegistrationClient) RegisterDefault() (*InvocationClient, *api.RegistrationResponse, error) {
	defaultEvents := []api.LifecycleEvent{api.Invoke, api.Shutdown}
	defaultRequest := api.RegistrationRequest{Events: defaultEvents, ConfigurationKeys: nil}
	return rc.Register(defaultRequest)
}

// Register registers, with custom registration parameters.
func (rc *RegistrationClient) Register(registrationRequest api.RegistrationRequest) (*InvocationClient, *api.RegistrationResponse, error) {
	registrationRequestJson, err := json.Marshal(registrationRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("error occurred while marshaling registration request %s", err)
	}

	req, err := http.NewRequest("POST", rc.GetRegisterURL(), bytes.NewBuffer(registrationRequestJson))
	if err != nil {
		return nil, nil, fmt.Errorf("error occurred while creating registration request %s", err)
	}

	req.Header.Set(api.ExtensionNameHeader, rc.extensionName)
	res, err := rc.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error occurred while making registration request %s", err)
	}

	defer util.Close(res.Body)

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	var registrationResponse api.RegistrationResponse
	err = json.Unmarshal(bodyBytes, &registrationResponse)
	if err != nil {
		return nil, nil, err
	}

	id, exists := res.Header[api.ExtensionIdHeader]
	if !exists {
		return nil, nil, fmt.Errorf("missing extension identifier")
	}

	invocationClient := InvocationClient{rc.version, rc.baseUrl, rc.httpClient, id[0]}
	return &invocationClient, &registrationResponse, nil
}

// GetNextEventURL returns the Lambda Extension next event URL
func (ic *InvocationClient) GetNextEventURL() string {
	return fmt.Sprintf("http://%s/%s/extension/event/next", ic.baseUrl, ic.version)
}

// NextEvent awaits the next event.
func (ic *InvocationClient) NextEvent() (*api.InvocationEvent, error) {
	req, err := http.NewRequest("GET", ic.GetNextEventURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("error occurred when creating next request %s", err)
	}

	req.Header.Set(api.ExtensionIdHeader, ic.extensionId)

	res, err := ic.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error occurred when calling extension/event/next %s", err)
	}

	defer util.Close(res.Body)

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error occurred while reading extension/event/next response body %s", err)
	}

	var event api.InvocationEvent
	err = json.Unmarshal(body, &event)
	if err != nil {
		return nil, fmt.Errorf("error occurred while unmarshaling extension/event/next response body %s", err)
	}

	return &event, nil
}
