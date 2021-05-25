package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"

	"github.com/stretchr/testify/assert"
)

func TestMainRegisterFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		if r.URL.Path == "/2020-01-01/extension/register" {
			w.Header().Add(api.ExtensionIdHeader, "test-ext-id")
			w.WriteHeader(500)
			_, _ = w.Write([]byte(""))
		}

	}))
	defer srv.Close()

	url := srv.URL[7:]

	_ = os.Setenv(api.LambdaHostPortEnvVar, url)
	defer os.Unsetenv(api.LambdaHostPortEnvVar)

	assert.Panics(t, main)
}

func TestMain(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		util.Logln(r.URL.Path)

		defer util.Close(r.Body)

		if r.URL.Path == "/2020-01-01/extension/register" {
			w.Header().Add(api.ExtensionIdHeader, "test-ext-id")
			w.WriteHeader(200)
			res, err := json.Marshal(api.RegistrationResponse{
				FunctionName:    "foobar",
				FunctionVersion: "latest",
				Handler:         "lambda.handler",
			})
			assert.Nil(t, err)
			_, _ = w.Write(res)
		}

		if r.URL.Path == "/2020-01-01/extension/init/error" {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-01-01/extension/exit/error" {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-08-15/logs" {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-01-01/extension/event/next" {
			w.WriteHeader(200)
			res, err := json.Marshal(api.InvocationEvent{
				EventType:          api.Shutdown,
				DeadlineMs:         1,
				RequestID:          "12345",
				InvokedFunctionARN: "arn:aws:lambda:us-east-1:12345:foobar",
				ShutdownReason:     api.Timeout,
				Tracing:            nil,
			})
			assert.Nil(t, err)
			_, _ = w.Write(res)
		}
	}))
	defer srv.Close()

	url := srv.URL[7:]

	_ = os.Setenv(api.LambdaHostPortEnvVar, url)
	defer os.Unsetenv(api.LambdaHostPortEnvVar)

	_ = os.Setenv("NEW_RELIC_LICENSE_KEY", "foobar")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY")

	_ = os.Setenv("NEW_RELIC_LOG_SERVER_HOST_OVERRIDE", "localhost")
	defer os.Unsetenv("NEW_RELIC_LOG_SERVER_HOST_OVERRIDE")

	_ = os.Setenv("NEW_RELIC_EXTENSION_LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("NEW_RELIC_EXTENSION_LOG_LEVEL")

	assert.NotPanics(t, main)
}
