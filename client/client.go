package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"github.com/newrelic/lambda-extension/api"
)


type InvocationClient struct {
	version      string
	base_url     string
	http_client  http.Client
	extension_id string
}


type RegistrationClient struct {
	extension_name string
	version        string
	base_url       string
	http_client    http.Client
}


func registerRequest(rc RegistrationClient) (*http.Response, error) {
	e := []string{api.INVOKE, api.SHUTDOWN}
	rr := api.RegistrationRequest{Events: e, ConfigurationKeys: nil}

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

func NextEvent(ic InvocationClient) error {
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

	event := api.InvocationEvent{}
	err = json.Unmarshal(body, &event)
	if err != nil {
		return fmt.Errorf("error occurred while unmarshaling extension/event/next response body %s", err)
	}

	if event.EventType == api.INVOKE {
		invokeRequest(event)
	}
	if event.EventType == api.SHUTDOWN {
		shutdownRequest(event)
	}

	res.Body.Close()
	return nil
}

func invokeRequest(event api.InvocationEvent) {
	// do things with invoke event...
	b, _ := json.Marshal(event)
	fmt.Println(string(b))
}

func shutdownRequest(event api.InvocationEvent) {
	// do things with shutdown event...
	b, _ := json.Marshal(event)
	fmt.Println(string(b))
}

func RegisterDefault(client http.Client) (ic InvocationClient, err error) {
	rc := RegistrationClient{
		extension_name: "extension_golang",
		version:        api.VERSION,
		base_url:       os.Getenv(api.AWS_LAMBDA_RUNTIME_API),
		http_client:    client,
	}
	res, err := registerRequest(rc)
	if err != nil {
		return InvocationClient{}, err
	}
	id, exists := res.Header["Lambda-Extension-Identifier"]
	if exists {
		ic = InvocationClient{rc.version, rc.base_url, rc.http_client, id[0]}
	}
	return ic, nil
}
