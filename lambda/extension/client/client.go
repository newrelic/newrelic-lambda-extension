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
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
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
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exeName := filepath.Base(exePath)

	return &RegistrationClient{
		extensionName: exeName,
		version:       api.Version,
		baseUrl:       os.Getenv(api.LambdaHostPortEnvVar),
		httpClient:    httpClient,
	}
}

// getRegisterURL returns the Lambda Extension register URL
func (rc *RegistrationClient) getRegisterURL() string {
	return fmt.Sprintf("http://%s/%s/extension/register", rc.baseUrl, rc.version)
}

// RegisterDefault registers for Invoke and Shutdown events, with no configuration parameters.
func (rc *RegistrationClient) RegisterDefault() (*InvocationClient, *api.RegistrationResponse, error) {
	defaultEvents := []api.LifecycleEvent{api.Invoke, api.Shutdown}
	defaultRequest := api.RegistrationRequest{Events: defaultEvents}
	return rc.Register(defaultRequest)
}

// Register registers, with custom registration parameters.
func (rc *RegistrationClient) Register(registrationRequest api.RegistrationRequest) (*InvocationClient, *api.RegistrationResponse, error) {
	registrationRequestJson, err := json.Marshal(registrationRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("error occurred while marshaling registration request %s", err)
	}

	req, err := http.NewRequest("POST", rc.getRegisterURL(), bytes.NewBuffer(registrationRequestJson))
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
		return nil, nil, fmt.Errorf("missing extension identifier. Response body %s", bodyBytes)
	}

	invocationClient := InvocationClient{rc.version, rc.baseUrl, rc.httpClient, id[0]}
	return &invocationClient, &registrationResponse, nil
}

// getNextEventURL returns the Lambda Extension next event URL
func (ic *InvocationClient) getNextEventURL() string {
	return fmt.Sprintf("http://%s/%s/extension/event/next", ic.baseUrl, ic.version)
}

// getInitErrorURL returns the Lambda Extension next event URL
func (ic *InvocationClient) getInitErrorURL() string {
	return fmt.Sprintf("http://%s/%s/extension/init/error", ic.baseUrl, ic.version)
}

// getExitErrorURL returns the Lambda Extension next event URL
func (ic *InvocationClient) getExitErrorURL() string {
	return fmt.Sprintf("http://%s/%s/extension/exit/error", ic.baseUrl, ic.version)
}

func (ic *InvocationClient) getLogRegistrationURL() string {
	return fmt.Sprintf("http://%s/%s/logs", ic.baseUrl, api.LogsApiVersion)
}

// LogRegister registers for log events
func (ic *InvocationClient) LogRegister(subscriptionRequest *api.LogSubscription) error {
	subscriptionRequestJson, err := json.Marshal(subscriptionRequest)
	if err != nil {
		return fmt.Errorf("error occurred while marshaling subscription request %s", err)
	}
	log.Println("Log registration with request ", string(subscriptionRequestJson))

	req, err := http.NewRequest("PUT", ic.getLogRegistrationURL(), bytes.NewBuffer(subscriptionRequestJson))
	if err != nil {
		return fmt.Errorf("error occurred while creating subscription request %s", err)
	}

	req.Header.Set(api.ExtensionIdHeader, ic.extensionId)

	res, err := ic.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error occurred while making log subscription request %s", err)
	}

	defer util.Close(res.Body)

	responseBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	log.Println("Registered for logs. Got response code ", res.StatusCode, string(responseBody))

	return nil
}

// NextEvent awaits the next event.
func (ic *InvocationClient) NextEvent() (*api.InvocationEvent, error) {
	req, err := http.NewRequest("GET", ic.getNextEventURL(), nil)
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

func (ic *InvocationClient) InitError(errorEnum string, initError error) (error) {
	errorBuf := bytes.NewBufferString(initError.Error())
	req, err := http.NewRequest("POST", ic.getInitErrorURL(), errorBuf)
	if err != nil {
		return fmt.Errorf("error occurred when creating init error request %s", err)
	}

	req.Header.Set(api.ExtensionIdHeader, ic.extensionId)
	req.Header.Set(api.ExtensionErrorTypeHeader, errorEnum)

	res, err := ic.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error occurred when calling extension/init/error %s", err)
	}

	defer util.Close(res.Body)

	return nil
}

func (ic *InvocationClient) ExitError(errorEnum string, exitError error) (error) {
	errorBuf := bytes.NewBufferString(exitError.Error())
	req, err := http.NewRequest("POST", ic.getExitErrorURL(), errorBuf)
	if err != nil {
		return fmt.Errorf("error occurred when creating exit error request %s", err)
	}

	req.Header.Set(api.ExtensionIdHeader, ic.extensionId)
	req.Header.Set(api.ExtensionErrorTypeHeader, errorEnum)

	res, err := ic.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error occurred when calling extension/exit/error %s", err)
	}

	defer util.Close(res.Body)

	return nil
}
