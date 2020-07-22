package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	INVOKE                 = "INVOKE"
	SHUTDOWN               = "SHUTDOWN"
	VERSION                = "2020-01-01"
	AWS_LAMBDA_RUNTIME_API = "AWS_LAMBDA_RUNTIME_API"
)

type InvocationEvent struct {
	EventType          string            `json:"eventType"`
	DeadlineMs         int               `json:"deadlineMs"`
	RequestId          string            `json:"requestId"`
	InvokedFunctionArn string            `json:"invokedFunctionArn"`
	Tracing            map[string]string `json:"tracing"`
}

func registerRequest(rc RegistrationClient) (*http.Response, error) {
	e := []string{INVOKE, SHUTDOWN}
	rr := RegistrationRequest{Events: e, ConfigurationKeys: nil}

	b, err := json.Marshal(rr)

	if err != nil {
		return nil, fmt.Errorf("error occurred while marshaling registration request %s", err)
	}

	register_url := fmt.Sprintf("http://%s/%s/extension/register", rc.base_url, rc.version)
	req, err := http.NewRequest("POST", register_url, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("error occurred while creating registration request %s", err)
	}
	req.Header.Set("Lambda-Extension-Name", rc.extension_name)
	res, err := rc.http_client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error occurred while making registration request %s", err)
	}

	res.Body.Close()
	return res, nil
}

func nextEvent(ic InvocationClient) error {
	next_event_url := fmt.Sprintf("http://%s/%s/extension/event/next", ic.base_url, ic.version)
	req, err := http.NewRequest("GET", next_event_url, nil)
	if err != nil {
		return fmt.Errorf("error occurred when creating next request %s", err)
	}
	req.Header.Set("Lambda-Extension-Identifier", ic.extension_id)

	res, err := ic.http_client.Do(req)
	if err != nil {
		return fmt.Errorf("error occurred when calling extension/event/next %s", err)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error occurred while reading extension/event/next response body %s", err)
	}

	event := InvocationEvent{}
	err = json.Unmarshal(body, &event)
	if err != nil {
		return fmt.Errorf("error occurred while unmarshaling extension/event/next response body %s", err)
	}

	if event.EventType == INVOKE {
		invokeRequest(event)
	}
	if event.EventType == SHUTDOWN {
		shutdownRequest(event)
	}

	res.Body.Close()
	return nil
}

func invokeRequest(event InvocationEvent) {
	// do things with invoke event...
	b, _ := json.Marshal(event)
	fmt.Println(string(b))
}

func shutdownRequest(event InvocationEvent) {
	// do things with shutdown event...
	b, _ := json.Marshal(event)
	fmt.Println(string(b))
}
