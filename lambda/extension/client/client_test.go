package client

import (
	"encoding/json"
	"github.com/newrelic/lambda-extension/util"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/newrelic/lambda-extension/lambda/extension/api"

	"github.com/stretchr/testify/assert"
)

var exePath, _ = os.Executable()
var exeName = filepath.Base(exePath)

func TestNew(t *testing.T) {
	_ = os.Setenv(api.LambdaHostPortEnvVar, "127.0.0.1:8123")
	defer os.Unsetenv(api.LambdaHostPortEnvVar)
	client := New(http.Client{})

	assert.Equal(t, exeName, client.extensionName)
}

func TestRegistrationClient_GetRegisterURL(t *testing.T) {
	_ = os.Setenv(api.LambdaHostPortEnvVar, "127.0.0.1:8123")
	defer os.Unsetenv(api.LambdaHostPortEnvVar)
	client := New(http.Client{})
	assert.Equal(t, "http://127.0.0.1:8123/2020-01-01/extension/register", client.getRegisterURL())
}

func TestRegistrationClient_RegisterDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPost)

		assert.NotEmpty(t, r.Header.Get(api.ExtensionNameHeader))

		reqBytes, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		defer util.Close(r.Body)
		assert.NotEmpty(t, reqBytes)

		var reqData api.RegistrationRequest
		assert.NoError(t, json.Unmarshal(reqBytes, &reqData))
		assert.Equal(t, []api.LifecycleEvent{api.Invoke, api.Shutdown}, reqData.Events)

		w.Header().Add(api.ExtensionIdHeader, "test-ext-id")
		w.WriteHeader(200)
		respBytes, _ := json.Marshal(api.RegistrationResponse{})
		_, _ = w.Write(respBytes)
	}))

	defer srv.Close()

	url := srv.URL[7:]
	_ = os.Setenv(api.LambdaHostPortEnvVar, url)
	defer os.Unsetenv(api.LambdaHostPortEnvVar)
	client := New(*srv.Client())
	invocationClient, rr, err := client.RegisterDefault()

	assert.NoError(t, err)
	assert.Equal(t, "test-ext-id", invocationClient.extensionId)
	assert.NotNil(t, rr)
}

func TestInvocationClient_NextEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodGet)

		assert.NotEmpty(t, r.Header.Get(api.ExtensionIdHeader))
		defer util.Close(r.Body)

		w.WriteHeader(200)
		respBytes, _ := json.Marshal(api.InvocationEvent{
			EventType:          api.Invoke,
			DeadlineMs:         1234,
			RequestID:          "5678",
			InvokedFunctionARN: "arn:aws:test",
			Tracing:            nil,
		})
		_, _ = w.Write(respBytes)
	}))

	defer srv.Close()

	url := srv.URL[7:]
	client := InvocationClient{
		version:     api.Version,
		baseUrl:     url,
		httpClient:  *srv.Client(),
		extensionId: "test-ext-id",
	}
	invocationEvent, err := client.NextEvent()

	assert.NoError(t, err)
	assert.NotNil(t, invocationEvent)
}
