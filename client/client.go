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

type InvocationClient struct {
	version     string
	baseUrl     string
	httpClient  http.Client
	extensionId string
}

type RegistrationClient struct {
	extensionName string
	version       string
	baseUrl       string
	httpClient    http.Client
}

func New(httpClient http.Client) RegistrationClient {
	exeName := filepath.Base(os.Args[0])

	return RegistrationClient{
		extensionName: exeName,
		version:       api.Version,
		baseUrl:       os.Getenv(api.AwsLambdaRuntimeApi),
		httpClient:    httpClient,
	}
}

func (rc *RegistrationClient) RegisterDefault() (InvocationClient, error) {
	res, err := rc.registerRequest()
	if err != nil {
		return InvocationClient{}, err
	}
	id, exists := res.Header["Lambda-Extension-Identifier"]
	if exists {
		return InvocationClient{rc.version, rc.baseUrl, rc.httpClient, id[0]}, nil
	} else {
		return InvocationClient{}, fmt.Errorf("missing extension identifier")
	}
}

func (rc *RegistrationClient) registerRequest() (*http.Response, error) {
	e := []string{api.Invoke, api.Shutdown}
	rr := api.RegistrationRequest{Events: e, ConfigurationKeys: nil}

	b, err := json.Marshal(rr)

	if err != nil {
		return nil, fmt.Errorf("error occurred while marshaling registration request %s", err)
	}

	registerUrl := fmt.Sprintf("http://%s/%s/extension/register", rc.baseUrl, rc.version)
	req, err := http.NewRequest("POST", registerUrl, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("error occurred while creating registration request %s", err)
	}
	req.Header.Set("Lambda-Extension-Name", rc.extensionName)
	res, err := rc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error occurred while making registration request %s", err)
	}
	//noinspection GoUnhandledErrorResult
	defer res.Body.Close()

	return res, nil
}

func (ic *InvocationClient) NextEvent() (*api.InvocationEvent, error) {
	nextEventUrl := fmt.Sprintf("http://%s/%s/extension/event/next", ic.baseUrl, ic.version)
	req, err := http.NewRequest("GET", nextEventUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("error occurred when creating next request %s", err)
	}
	req.Header.Set("Lambda-Extension-Identifier", ic.extensionId)

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
