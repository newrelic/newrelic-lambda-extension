//go:build !race
// +build !race

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"

	"github.com/stretchr/testify/assert"
)

// TODO: These tests are very repetitive. Helpers would be useful here.

func TestMainRegisterFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		if r.URL.Path == "/2020-01-01/extension/register" {
			w.Header().Add(api.ExtensionIdHeader, "test-ext-id")
			w.WriteHeader(400)
			_, _ = w.Write(nil)
		}
	}))
	defer srv.Close()

	url := srv.URL[7:]

	_ = os.Setenv(api.LambdaHostPortEnvVar, url)
	defer os.Unsetenv(api.LambdaHostPortEnvVar)

	assert.Panics(t, main)
}

func TestMainLogServerInitFail(t *testing.T) {
	var (
		registerRequestCount    int
		initErrorRequestCount   int
		exitErrorRequestCount   int
		logRegisterRequestCount int
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		if r.URL.Path == "/2020-01-01/extension/register" {
			registerRequestCount++

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
			initErrorRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-01-01/extension/exit/error" {
			exitErrorRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-08-15/logs" {
			logRegisterRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}
	}))
	defer srv.Close()

	url := srv.URL[7:]

	_ = os.Setenv(api.LambdaHostPortEnvVar, url)
	defer os.Unsetenv(api.LambdaHostPortEnvVar)

	_ = os.Setenv("NEW_RELIC_LICENSE_KEY", "foobar")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY")

	// Shouldn't be able to bind to this locally
	_ = os.Setenv("NEW_RELIC_LOG_SERVER_HOST", "sandbox.localdomain")
	defer os.Unsetenv("NEW_RELIC_LOG_SERVER_HOST")

	_ = os.Setenv("NEW_RELIC_EXTENSION_LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("NEW_RELIC_EXTENSION_LOG_LEVEL")

	assert.Panics(t, main)

	assert.Equal(t, 1, registerRequestCount)
	assert.Equal(t, 1, initErrorRequestCount)
	assert.Equal(t, 0, exitErrorRequestCount)
	assert.Equal(t, 0, logRegisterRequestCount)
}

func TestMainLogServerRegisterFail(t *testing.T) {
	var (
		registerRequestCount    int
		initErrorRequestCount   int
		exitErrorRequestCount   int
		logRegisterRequestCount int
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		if r.URL.Path == "/2020-01-01/extension/register" {
			registerRequestCount++

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
			initErrorRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-01-01/extension/exit/error" {
			exitErrorRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-08-15/logs" {
			logRegisterRequestCount++

			w.WriteHeader(400)
			_, _ = w.Write(nil)
		}
	}))
	defer srv.Close()

	url := srv.URL[7:]

	_ = os.Setenv(api.LambdaHostPortEnvVar, url)
	defer os.Unsetenv(api.LambdaHostPortEnvVar)

	_ = os.Setenv("NEW_RELIC_LICENSE_KEY", "foobar")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY")

	_ = os.Setenv("NEW_RELIC_LOG_SERVER_HOST", "localhost")
	defer os.Unsetenv("NEW_RELIC_LOG_SERVER_HOST")

	_ = os.Setenv("NEW_RELIC_EXTENSION_LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("NEW_RELIC_EXTENSION_LOG_LEVEL")

	assert.Panics(t, main)

	assert.Equal(t, 1, registerRequestCount)
	assert.Equal(t, 1, initErrorRequestCount)
	assert.Equal(t, 0, exitErrorRequestCount)
	assert.Equal(t, 1, logRegisterRequestCount)
}

func TestMainShutdown(t *testing.T) {
	var (
		registerRequestCount    int
		initErrorRequestCount   int
		exitErrorRequestCount   int
		logRegisterRequestCount int
		nextEventRequestCount   int
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		util.Logln("Path: ", r.URL.Path)
		defer util.Close(r.Body)

		if r.URL.Path == "/2020-01-01/extension/register" {
			registerRequestCount++

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
			initErrorRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-01-01/extension/exit/error" {
			exitErrorRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-08-15/logs" {
			logRegisterRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-01-01/extension/event/next" {
			nextEventRequestCount++

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

	_ = os.Setenv("NEW_RELIC_LOG_SERVER_HOST", "localhost")
	defer os.Unsetenv("NEW_RELIC_LOG_SERVER_HOST")

	_ = os.Setenv("NEW_RELIC_EXTENSION_LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("NEW_RELIC_EXTENSION_LOG_LEVEL")

	assert.NotPanics(t, main)

	assert.Equal(t, 1, registerRequestCount)
	assert.Equal(t, 0, initErrorRequestCount)
	assert.Equal(t, 0, exitErrorRequestCount)
	assert.Equal(t, 1, logRegisterRequestCount)
	assert.Equal(t, 1, nextEventRequestCount)
}

func TestMainNoLicenseKey(t *testing.T) {
	var (
		registerRequestCount    int
		initErrorRequestCount   int
		exitErrorRequestCount   int
		logRegisterRequestCount int
		nextEventRequestCount   int
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		util.Logln("Path: ", r.URL.Path)
		defer util.Close(r.Body)

		if r.URL.Path == "/2020-01-01/extension/register" {
			registerRequestCount++

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
			initErrorRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-01-01/extension/exit/error" {
			exitErrorRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-08-15/logs" {
			logRegisterRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-01-01/extension/event/next" {
			nextEventRequestCount++

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

	_ = os.Setenv("NEW_RELIC_EXTENSION_LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("NEW_RELIC_EXTENSION_LOG_LEVEL")

	assert.NotPanics(t, main)

	assert.Equal(t, 1, registerRequestCount)
	assert.Equal(t, 0, initErrorRequestCount)
	assert.Equal(t, 0, exitErrorRequestCount)
	assert.Equal(t, 0, logRegisterRequestCount)
	assert.Equal(t, 1, nextEventRequestCount)
}

func TestMainExtensionDisabled(t *testing.T) {
	var (
		registerRequestCount    int
		initErrorRequestCount   int
		exitErrorRequestCount   int
		logRegisterRequestCount int
		nextEventRequestCount   int
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		util.Logln("Path: ", r.URL.Path)
		defer util.Close(r.Body)

		if r.URL.Path == "/2020-01-01/extension/register" {
			registerRequestCount++

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
			initErrorRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-01-01/extension/exit/error" {
			exitErrorRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-08-15/logs" {
			logRegisterRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-01-01/extension/event/next" {
			nextEventRequestCount++

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

	_ = os.Setenv("NEW_RELIC_LAMBDA_EXTENSION_ENABLED", "false")
	defer os.Unsetenv("NEW_RELIC_LAMBDA_EXTENSION_ENABLED")

	_ = os.Setenv("NEW_RELIC_EXTENSION_LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("NEW_RELIC_EXTENSION_LOG_LEVEL")

	assert.NotPanics(t, main)

	assert.Equal(t, 1, registerRequestCount)
	assert.Equal(t, 0, initErrorRequestCount)
	assert.Equal(t, 0, exitErrorRequestCount)
	assert.Equal(t, 0, logRegisterRequestCount)
	assert.Equal(t, 1, nextEventRequestCount)
}

func TestMainTimeout(t *testing.T) {
	var (
		registerRequestCount    int
		initErrorRequestCount   int
		exitErrorRequestCount   int
		logRegisterRequestCount int
		nextEventRequestCount   int
	)

	ctx, cancel := context.WithCancel(context.Background())
	overrideContext(ctx)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		if r.URL.Path == "/2020-01-01/extension/register" {
			registerRequestCount++

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
			initErrorRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-01-01/extension/exit/error" {
			exitErrorRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-08-15/logs" {
			logRegisterRequestCount++

			w.WriteHeader(200)
			_, _ = w.Write(nil)

		}

		if r.URL.Path == "/2020-01-01/extension/event/next" {
			nextEventRequestCount++

			w.WriteHeader(200)
			res, err := json.Marshal(api.InvocationEvent{
				EventType:          api.Invoke,
				DeadlineMs:         1000,
				RequestID:          "12345",
				InvokedFunctionARN: "arn:aws:lambda:us-east-1:12345:foobar",
				ShutdownReason:     "",
				Tracing:            nil,
			})
			assert.Nil(t, err)
			_, _ = w.Write(res)

			cancel()
		}
	}))
	defer srv.Close()

	url := srv.URL[7:]

	_ = os.Setenv(api.LambdaHostPortEnvVar, url)
	defer os.Unsetenv(api.LambdaHostPortEnvVar)

	_ = os.Setenv("NEW_RELIC_LICENSE_KEY", "foobar")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY")

	_ = os.Setenv("NEW_RELIC_LOG_SERVER_HOST", "localhost")
	defer os.Unsetenv("NEW_RELIC_LOG_SERVER_HOST")

	_ = os.Setenv("NEW_RELIC_EXTENSION_LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("NEW_RELIC_EXTENSION_LOG_LEVEL")

	assert.NotPanics(t, main)

	assert.Equal(t, 1, registerRequestCount)
	assert.Equal(t, 0, initErrorRequestCount)
	assert.Equal(t, 0, exitErrorRequestCount)
	assert.Equal(t, 1, logRegisterRequestCount)
	assert.Equal(t, 1, nextEventRequestCount)
}

func TestMainTimeoutUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(200*time.Millisecond))
	defer cancel()
	overrideContext(ctx)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		if r.URL.Path == "/2020-01-01/extension/register" {
			w.Header().Add(api.ExtensionIdHeader, "test-ext-id")
			w.WriteHeader(200)
			res, err := json.Marshal(api.RegistrationResponse{
				FunctionName:    "foobar",
				FunctionVersion: "$latest",
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
			_, _ = w.Write(nil)
		}

		if r.URL.Path == "/2020-08-15/logs" {
			w.WriteHeader(200)
			_, _ = w.Write(nil)
		}

		if r.URL.Path == "/2020-01-01/extension/event/next" {
			time.Sleep(25 * time.Millisecond)

			w.WriteHeader(200)
			res, err := json.Marshal(api.InvocationEvent{
				EventType:          api.Invoke,
				DeadlineMs:         100,
				RequestID:          "12345",
				InvokedFunctionARN: "arn:aws:lambda:us-east-1:12345:foobar",
				ShutdownReason:     "",
				Tracing:            nil,
			})
			assert.Nil(t, err)
			_, _ = w.Write(res)
		}

		if r.URL.Path == "/aws/lambda/v1" {
			time.Sleep(5 * time.Second)

			w.WriteHeader(200)
			_, _ = w.Write(nil)
		}
	}))
	defer srv.Close()

	url := srv.URL[7:]

	_ = os.Setenv(api.LambdaHostPortEnvVar, url)
	defer os.Unsetenv(api.LambdaHostPortEnvVar)

	_ = os.Setenv("NEW_RELIC_LICENSE_KEY", "foobar")
	defer os.Unsetenv("NEW_RELIC_LICENSE_KEY")

	_ = os.Setenv("NEW_RELIC_LOG_SERVER_HOST", "localhost")
	defer os.Unsetenv("NEW_RELIC_LOG_SERVER_HOST")

	_ = os.Setenv("NEW_RELIC_EXTENSION_LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("NEW_RELIC_EXTENSION_LOG_LEVEL")

	_ = os.Setenv("NEW_RELIC_TELEMETRY_ENDPOINT", fmt.Sprintf("%s/aws/lambda/v1", srv.URL))
	defer os.Unsetenv("NEW_RELIC_TELEMETRY_ENDPOINT")

	_ = os.Remove("/tmp/newrelic-telemetry")

	go func() {
		pipeOpened := false

		for {
			select {
			case <-ctx.Done():
				return
			default:
				if _, err := os.Stat("/tmp/newrelic-telemetry"); os.IsNotExist(err) {
					if pipeOpened {
						return
					} else {
						continue
					}
				} else {
					pipeOpened = true
				}

				pipe, err := os.OpenFile("/tmp/newrelic-telemetry", os.O_WRONLY, 0)
				assert.Nil(t, err)
				defer pipe.Close()

				pipe.WriteString("foobar\n")
				pipe.Close()
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	assert.NotPanics(t, main)
}

func TestMainTimeoutNoPipeWrite(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(200*time.Millisecond))
	defer cancel()
	overrideContext(ctx)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		if r.URL.Path == "/2020-01-01/extension/register" {
			w.Header().Add(api.ExtensionIdHeader, "test-ext-id")
			w.WriteHeader(200)
			res, err := json.Marshal(api.RegistrationResponse{
				FunctionName:    "foobar",
				FunctionVersion: "$latest",
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
			_, _ = w.Write(nil)
		}

		if r.URL.Path == "/2020-08-15/logs" {
			w.WriteHeader(200)
			_, _ = w.Write(nil)
		}

		if r.URL.Path == "/2020-01-01/extension/event/next" {
			time.Sleep(25 * time.Millisecond)

			w.WriteHeader(200)
			res, err := json.Marshal(api.InvocationEvent{
				EventType:          api.Invoke,
				DeadlineMs:         100,
				RequestID:          "12345",
				InvokedFunctionARN: "arn:aws:lambda:us-east-1:12345:foobar",
				ShutdownReason:     "",
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

	_ = os.Setenv("NEW_RELIC_LOG_SERVER_HOST", "localhost")
	defer os.Unsetenv("NEW_RELIC_LOG_SERVER_HOST")

	_ = os.Setenv("NEW_RELIC_EXTENSION_LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("NEW_RELIC_EXTENSION_LOG_LEVEL")

	_ = os.Remove("/tmp/newrelic-telemetry")

	go func() {
		pipeOpened := false

		for {
			select {
			case <-ctx.Done():
				return
			default:
				if _, err := os.Stat("/tmp/newrelic-telemetry"); os.IsNotExist(err) {
					if pipeOpened {
						return
					} else {
						continue
					}
				} else {
					pipeOpened = true
				}

				pipe, err := os.OpenFile("/tmp/newrelic-telemetry", os.O_WRONLY, 0)
				assert.Nil(t, err)
				defer pipe.Close()

				time.Sleep(200 * time.Millisecond)
				pipe.Close()
			}
		}
	}()

	assert.NotPanics(t, main)
}

func overrideContext(ctx context.Context) {
	rootCtx = ctx
}
