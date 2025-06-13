// Package client is a generic client for the AWS Lambda Extension API.
// The API's lifecycle begins with execution of the extension binary, which is expected to register.
// The extension then makes blocking requests for the next event. The response to the next event request
// is either a notification of the next event, or a shutdown notification.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	extensionFeature string
}

// Constructs a new RegistrationClient. This is the entry point.
func New(httpClient http.Client) *RegistrationClient {
	exePath, err := os.Executable()
	if err != nil {
		util.Fatal(err)
	}

	exeName := filepath.Base(exePath)

	return &RegistrationClient{
		extensionName: exeName,
		version:       api.Version,
		baseUrl:       os.Getenv(api.LambdaHostPortEnvVar),
		httpClient:    httpClient,
		extensionFeature: "accountId",
	}
}

// getRegisterURL returns the Lambda Extension register URL
func (rc *RegistrationClient) getRegisterURL() string {
	return fmt.Sprintf("http://%s/%s/extension/register", rc.baseUrl, rc.version)
}

// RegisterDefault registers for Invoke and Shutdown events, with no configuration parameters.
func (rc *RegistrationClient) RegisterDefault(ctx context.Context) (*InvocationClient, *api.RegistrationResponse, error) {
	defaultEvents := []api.LifecycleEvent{api.Invoke, api.Shutdown}
	defaultRequest := api.RegistrationRequest{Events: defaultEvents}
	return rc.Register(ctx, defaultRequest)
}

// Register registers, with custom registration parameters.
func (rc *RegistrationClient) Register(ctx context.Context, registrationRequest api.RegistrationRequest) (*InvocationClient, *api.RegistrationResponse, error) {
	registrationRequestJson, err := json.Marshal(registrationRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("error occurred while marshaling registration request %s", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", rc.getRegisterURL(), bytes.NewBuffer(registrationRequestJson))
	if err != nil {
		return nil, nil, fmt.Errorf("error occurred while creating registration request %s", err)
	}

	req.Header.Set(api.ExtensionNameHeader, rc.extensionName)
	req.Header.Set(api.ExtensionFeatureHeader, rc.extensionFeature)

	res, err := rc.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error occurred while making registration request %s", err)
	}

	defer util.Close(res.Body)

	if res.StatusCode == http.StatusInternalServerError {
		util.Panic("error occurred while making registration request: ", res.Status)
	}

	if res.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("error occurred while making registration request: %s", res.Status)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	util.Debugf("Registration response: %s", bodyBytes)

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

// getInitErrorURL returns the Lambda Extension initialization error URL
func (ic *InvocationClient) getInitErrorURL() string {
	return fmt.Sprintf("http://%s/%s/extension/init/error", ic.baseUrl, ic.version)
}

// getExitErrorURL returns the Lambda exit error URL
func (ic *InvocationClient) getExitErrorURL() string {
	return fmt.Sprintf("http://%s/%s/extension/exit/error", ic.baseUrl, ic.version)
}

// getLogRegistrationURL returns the Lambda Log Registration URL
func (ic *InvocationClient) getLogRegistrationURL() string {
	return fmt.Sprintf("http://%s/%s/logs", ic.baseUrl, api.LogsApiVersion)
}

// LogRegister registers for log events
func (ic *InvocationClient) LogRegister(ctx context.Context, subscriptionRequest *api.LogSubscription) error {
	subscriptionRequestJson, err := json.Marshal(subscriptionRequest)
	if err != nil {
		return fmt.Errorf("error occurred while marshaling subscription request %s", err)
	}

	util.Debugln("Log registration with request ", string(subscriptionRequestJson))

	req, err := http.NewRequestWithContext(ctx, "PUT", ic.getLogRegistrationURL(), bytes.NewBuffer(subscriptionRequestJson))
	if err != nil {
		return fmt.Errorf("error occurred while creating subscription request %s", err)
	}

	req.Header.Set(api.ExtensionIdHeader, ic.extensionId)

	res, err := ic.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error occurred while making log subscription request %s", err)
	}

	defer util.Close(res.Body)

	if res.StatusCode == http.StatusInternalServerError {
		util.Panic("error occurred while making log subscription request: ", res.Status)
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("error occurred while making log subscription request: %s", res.Status)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	util.Debugln("Registered for logs. Got response code ", res.StatusCode, string(responseBody))

	return nil
}

// NextEvent awaits the next event.
func (ic *InvocationClient) NextEvent(ctx context.Context) (*api.InvocationEvent, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", ic.getNextEventURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("error occurred when creating next request %s", err)
	}

	req.Header.Set(api.ExtensionIdHeader, ic.extensionId)

	res, err := ic.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error occurred when calling extension/event/next %s", err)
	}

	defer util.Close(res.Body)

	if res.StatusCode == http.StatusInternalServerError {
		util.Panic("error occurred when calling extension/event/next: ", res.Status)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error occurred when calling extension/event/next: %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
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

// InitError sends an initialization error to the lambda platform
func (ic *InvocationClient) InitError(ctx context.Context, errorEnum string, initError error) error {
	errorBuf := bytes.NewBufferString(initError.Error())

	req, err := http.NewRequestWithContext(ctx, "POST", ic.getInitErrorURL(), errorBuf)
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

	if res.StatusCode == http.StatusInternalServerError {
		util.Panic("error occurred while making init error request: ", res.Status)
	}

	if res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("error occurred while making init error request: %s", res.Status)
	}

	return nil
}

// ExitError sends an exit error to the lambda platform
func (ic *InvocationClient) ExitError(ctx context.Context, errorEnum string, exitError error) error {
	errorBuf := bytes.NewBufferString(exitError.Error())
	req, err := http.NewRequestWithContext(ctx, "POST", ic.getExitErrorURL(), errorBuf)
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

	if res.StatusCode == http.StatusInternalServerError {
		util.Panic("error occurred while making exit error request: ", res.Status)
	}

	if res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("error occurred while making exit error request: %s", res.Status)
	}

	return nil
}
