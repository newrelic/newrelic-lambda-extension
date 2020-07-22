package main

import (
	"net/http"
	"os"
)

type RegistrationClient struct {
	extension_name string
	version        string
	base_url       string
	http_client    http.Client
}

type RegistrationRequest struct {
	Events            []string `json:"events"`
	ConfigurationKeys []string `json:"configurationKeys"`
}

type InvocationClient struct {
	version      string
	base_url     string
	http_client  http.Client
	extension_id string
}

func registerDefault(client http.Client) (ic InvocationClient, err error) {
	rc := RegistrationClient{
		extension_name: "extension_golang",
		version:        VERSION,
		base_url:       os.Getenv(AWS_LAMBDA_RUNTIME_API),
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
