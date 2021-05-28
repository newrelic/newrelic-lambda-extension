package client

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"

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
	rc := RegistrationClient{}
	ctx := context.Background()
	ic, res, err := rc.RegisterDefault(ctx)
	assert.Nil(t, ic)
	assert.Nil(t, res)
	assert.Error(t, err)

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
	invocationClient, rr, err := client.RegisterDefault(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "test-ext-id", invocationClient.extensionId)
	assert.NotNil(t, rr)
	assert.NotEmpty(t, invocationClient.getInitErrorURL())
	assert.NotEmpty(t, invocationClient.getExitErrorURL())
	assert.NotEmpty(t, invocationClient.getLogRegistrationURL())
}

func TestRegistrationClient_RegisterError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		w.Header().Add(api.ExtensionIdHeader, "test-ext-id")
		w.WriteHeader(400)
		_, _ = w.Write(nil)
	}))
	defer srv.Close()

	url := srv.URL[7:]

	_ = os.Setenv(api.LambdaHostPortEnvVar, url)
	defer os.Unsetenv(api.LambdaHostPortEnvVar)

	client := New(*srv.Client())
	ctx := context.Background()
	ic, rr, err := client.RegisterDefault(ctx)

	assert.Nil(t, ic)
	assert.Nil(t, rr)
	assert.Error(t, err)
}

func TestRegistrationClient_RegisterPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		w.Header().Add(api.ExtensionIdHeader, "test-ext-id")
		w.WriteHeader(500)
		_, _ = w.Write(nil)
	}))
	defer srv.Close()

	url := srv.URL[7:]

	_ = os.Setenv(api.LambdaHostPortEnvVar, url)
	defer os.Unsetenv(api.LambdaHostPortEnvVar)

	client := New(*srv.Client())
	ctx := context.Background()

	assert.Panics(t, func() {
		client.RegisterDefault(ctx)
	})
}

func TestInvocationClient_LogRegister(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPut)

		assert.NotEmpty(t, r.Header.Get(api.ExtensionIdHeader))
		defer util.Close(r.Body)

		w.WriteHeader(200)
		_, _ = w.Write(nil)
	}))
	defer srv.Close()

	url := srv.URL[7:]

	client := InvocationClient{
		version:     api.Version,
		baseUrl:     url,
		httpClient:  *srv.Client(),
		extensionId: "test-ext-id",
	}

	eventTypes := []api.LogEventType{api.Platform}
	subscriptionRequest := api.DefaultLogSubscription(eventTypes, 12345)

	ctx := context.Background()
	err := client.LogRegister(ctx, subscriptionRequest)

	assert.NoError(t, err)
}

func TestInvocationClient_LogRegisterError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		w.WriteHeader(400)
		_, _ = w.Write(nil)
	}))
	defer srv.Close()

	url := srv.URL[7:]

	client := InvocationClient{
		version:     api.Version,
		baseUrl:     url,
		httpClient:  *srv.Client(),
		extensionId: "test-ext-id",
	}

	eventTypes := []api.LogEventType{api.Platform}
	subscriptionRequest := api.DefaultLogSubscription(eventTypes, 12345)

	ctx := context.Background()
	err := client.LogRegister(ctx, subscriptionRequest)

	assert.Error(t, err)
}

func TestInvocationClient_LogRegisterEPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		w.WriteHeader(500)
		_, _ = w.Write(nil)
	}))
	defer srv.Close()

	url := srv.URL[7:]

	client := InvocationClient{
		version:     api.Version,
		baseUrl:     url,
		httpClient:  *srv.Client(),
		extensionId: "test-ext-id",
	}

	eventTypes := []api.LogEventType{api.Platform}
	subscriptionRequest := api.DefaultLogSubscription(eventTypes, 12345)

	ctx := context.Background()
	assert.Panics(t, func() {
		client.LogRegister(ctx, subscriptionRequest)
	})
}

func TestInvocationClient_InitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPost)

		assert.NotEmpty(t, r.Header.Get(api.ExtensionIdHeader))
		defer util.Close(r.Body)

		w.WriteHeader(202)
		_, _ = w.Write(nil)
	}))
	defer srv.Close()

	url := srv.URL[7:]

	client := InvocationClient{
		version:     api.Version,
		baseUrl:     url,
		httpClient:  *srv.Client(),
		extensionId: "test-ext-id",
	}

	ctx := context.Background()
	err := client.InitError(ctx, "foo.bar", errors.New("something went wrong"))

	assert.NoError(t, err)
}

func TestInvocationClient_InitErrorError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		w.WriteHeader(400)
		_, _ = w.Write(nil)
	}))
	defer srv.Close()

	url := srv.URL[7:]

	client := InvocationClient{
		version:     api.Version,
		baseUrl:     url,
		httpClient:  *srv.Client(),
		extensionId: "test-ext-id",
	}

	ctx := context.Background()
	err := client.InitError(ctx, "foo.bar", errors.New("something went wrong"))

	assert.Error(t, err)
}

func TestInvocationClient_InitErrorPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		w.WriteHeader(500)
		_, _ = w.Write(nil)
	}))
	defer srv.Close()

	url := srv.URL[7:]

	client := InvocationClient{
		version:     api.Version,
		baseUrl:     url,
		httpClient:  *srv.Client(),
		extensionId: "test-ext-id",
	}

	ctx := context.Background()
	assert.Panics(t, func() {
		client.InitError(ctx, "foo.bar", errors.New("something went wrong"))
	})
}

func TestInvocationClient_ExitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPost)

		assert.NotEmpty(t, r.Header.Get(api.ExtensionIdHeader))
		defer util.Close(r.Body)

		w.WriteHeader(202)
		_, _ = w.Write(nil)
	}))
	defer srv.Close()

	url := srv.URL[7:]

	client := InvocationClient{
		version:     api.Version,
		baseUrl:     url,
		httpClient:  *srv.Client(),
		extensionId: "test-ext-id",
	}

	ctx := context.Background()
	err := client.ExitError(ctx, "foo.bar", errors.New("something went wrong"))

	assert.NoError(t, err)
}

func TestInvocationClient_ExitErrorError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		w.WriteHeader(400)
		_, _ = w.Write(nil)
	}))
	defer srv.Close()

	url := srv.URL[7:]

	client := InvocationClient{
		version:     api.Version,
		baseUrl:     url,
		httpClient:  *srv.Client(),
		extensionId: "test-ext-id",
	}

	ctx := context.Background()
	err := client.ExitError(ctx, "foo.bar", errors.New("something went wrong"))

	assert.Error(t, err)
}

func TestInvocationClient_ExitErrorPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		w.WriteHeader(500)
		_, _ = w.Write(nil)
	}))
	defer srv.Close()

	url := srv.URL[7:]

	client := InvocationClient{
		version:     api.Version,
		baseUrl:     url,
		httpClient:  *srv.Client(),
		extensionId: "test-ext-id",
	}

	ctx := context.Background()
	assert.Panics(t, func() {
		client.ExitError(ctx, "foo.bar", errors.New("something went wrong"))
	})
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
	ctx := context.Background()
	invocationEvent, err := client.NextEvent(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, invocationEvent)
}

func TestInvocationClient_NextEventError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		w.WriteHeader(400)
		_, _ = w.Write(nil)
	}))
	defer srv.Close()

	url := srv.URL[7:]

	client := InvocationClient{
		version:     api.Version,
		baseUrl:     url,
		httpClient:  *srv.Client(),
		extensionId: "test-ext-id",
	}

	ctx := context.Background()
	event, err := client.NextEvent(ctx)

	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestInvocationClient_NextEventPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		w.WriteHeader(500)
		_, _ = w.Write(nil)
	}))
	defer srv.Close()

	url := srv.URL[7:]

	client := InvocationClient{
		version:     api.Version,
		baseUrl:     url,
		httpClient:  *srv.Client(),
		extensionId: "test-ext-id",
	}

	ctx := context.Background()
	assert.Panics(t, func() {
		client.NextEvent(ctx)
	})
}
