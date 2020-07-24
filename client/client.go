// Package client is a generic client for the AWS Lambda Extension API.
// The API's lifecycle begins with execution of the extension binary, which is expected to register.
// The extension then makes blocking requests for the next event. The response to the next event request
// is either a notification of the next event, or a shutdown notification.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/newrelic/lambda-extension/api"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

const extensionIdHeader = "Lambda-Extension-Identifier"

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
func New(httpClient http.Client) RegistrationClient {
	exeName := filepath.Base(os.Args[0])

	return RegistrationClient{
		extensionName: exeName,
		version:       api.Version,
		baseUrl:       os.Getenv(api.AwsLambdaRuntimeApi),
		httpClient:    httpClient,
	}
}

// Register for Invoke and Shutdown events, with no configuration parameters.
func (rc *RegistrationClient) RegisterDefault() (*InvocationClient, *api.RegistrationResponse, error) {
	defaultEvents := []string{api.Invoke, api.Shutdown}
	defaultRequest := api.RegistrationRequest{Events: defaultEvents, ConfigurationKeys: nil}
	return rc.Register(defaultRequest)
}

// Register, with custom registration.
func (rc *RegistrationClient) Register(registrationRequest api.RegistrationRequest) (*InvocationClient, *api.RegistrationResponse, error) {
	registrationRequestJson, err := json.Marshal(registrationRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("error occurred while marshaling registration request %s", err)
	}

	registerUrl := fmt.Sprintf("http://%s/%s/extension/register", rc.baseUrl, rc.version)
	req, err := http.NewRequest("POST", registerUrl, bytes.NewBuffer(registrationRequestJson))
	if err != nil {
		return nil, nil, fmt.Errorf("error occurred while creating registration request %s", err)
	}

	req.Header.Set("Lambda-Extension-Name", rc.extensionName)
	res, err := rc.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error occurred while making registration request %s", err)
	}

	//noinspection GoUnhandledErrorResult
	defer res.Body.Close()

	id, exists := res.Header[extensionIdHeader]
	if exists {
		invocationClient := InvocationClient{rc.version, rc.baseUrl, rc.httpClient, id[0]}
		registrationResponse := api.RegistrationResponse{}
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, nil, err
		}

		err = json.Unmarshal(bodyBytes, &registrationResponse)
		if err != nil {
			return nil, nil, err
		}

		return &invocationClient, &registrationResponse, nil
	} else {
		return nil, nil, fmt.Errorf("missing extension identifier")
	}
}

// NextEvent awaits the next event.
func (ic *InvocationClient) NextEvent() (*api.InvocationEvent, error) {
	nextEventUrl := fmt.Sprintf("http://%s/%s/extension/event/next", ic.baseUrl, ic.version)
	req, err := http.NewRequest("GET", nextEventUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("error occurred when creating next request %s", err)
	}

	req.Header.Set(extensionIdHeader, ic.extensionId)

	res, err := ic.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error occurred when calling extension/event/next %s", err)
	}
	//noinspection GoUnhandledErrorResult
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error occurred while reading extension/event/next response body %s", err)
	}

	event := api.InvocationEvent{}
	err = json.Unmarshal(body, &event)
	if err != nil {
		return nil, fmt.Errorf("error occurred while unmarshaling extension/event/next response body %s", err)
	}

	return &event, nil
}
