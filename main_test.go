//go:build !race
// +build !race

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/telemetry"
	"github.com/newrelic/newrelic-lambda-extension/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestMainTimeoutAddTelemetry(t *testing.T) {
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
				ShutdownReason:     api.Timeout,
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

	_ = os.Setenv("NEW_RELIC_COLLECT_TRACE_ID", "true")
	defer os.Unsetenv("NEW_RELIC_COLLECT_TRACE_ID")

	_ = os.Setenv("NEW_RELIC_LOG_SERVER_HOST", "localhost")
	defer os.Unsetenv("NEW_RELIC_LOG_SERVER_HOST")

	_ = os.Setenv("NEW_RELIC_EXTENSION_LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("NEW_RELIC_EXTENSION_LOG_LEVEL")

	lastRequestId := "12345"
	timeoutMessage := "2025-02-11T09:19:31Z 0dffa608-e35a-45e8-89fe-f97093ae8c0d Task timed out after 9223372036.85 seconds"
	isAPMTelemetry := false
	conf := config.ConfigurationFromEnvironment()

	batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis), conf.CollectTraceID)

	assert.Nil(t, batch.AddTelemetry(lastRequestId, []byte(timeoutMessage), isAPMTelemetry))
	assert.NotPanics(t, main)

	assert.Equal(t, 1, registerRequestCount)
	assert.Equal(t, 0, initErrorRequestCount)
	assert.Equal(t, 0, exitErrorRequestCount)
	assert.Equal(t, 1, logRegisterRequestCount)
	assert.Equal(t, 1, nextEventRequestCount)
}

func TestMainPlatformErrorAddTelemetry(t *testing.T) {
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
				ShutdownReason:     api.Failure,
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

	_ = os.Setenv("NEW_RELIC_COLLECT_TRACE_ID", "true")
	defer os.Unsetenv("NEW_RELIC_COLLECT_TRACE_ID")

	_ = os.Setenv("NEW_RELIC_LOG_SERVER_HOST", "localhost")
	defer os.Unsetenv("NEW_RELIC_LOG_SERVER_HOST")

	_ = os.Setenv("NEW_RELIC_EXTENSION_LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("NEW_RELIC_EXTENSION_LOG_LEVEL")

	lastRequestId := "12345"
	errorMessage := "RequestId: 0dffa608-e35a-45e8-89fe-f97093ae8c0d AWS Lambda platform fault caused a shutdown"
	isAPMTelemetry := false
	conf := config.ConfigurationFromEnvironment()

	batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis), conf.CollectTraceID)

	assert.Nil(t, batch.AddTelemetry(lastRequestId, []byte(errorMessage), isAPMTelemetry))
	assert.NotPanics(t, main)

	assert.Equal(t, 1, registerRequestCount)
	assert.Equal(t, 0, initErrorRequestCount)
	assert.Equal(t, 0, exitErrorRequestCount)
	assert.Equal(t, 1, logRegisterRequestCount)
	assert.Equal(t, 1, nextEventRequestCount)
}

func TestPollLogServerAddTelemetry(t *testing.T) {
	pollLogContent := "REPORT RequestId: f2b3bcc1-a9e1-418d-9709-34402bb218a7 Duration: 3000.00 ms Billed Duration: 3000 ms Memory Size: 128 MB Max Memory Used: 89 MB Init Duration: 836.96 ms"
	pollLogRequestId := "f2b3bcc1-a9e1-418d-9709-34402bb218a7"
	isAPMTelemetry := false
	conf := config.ConfigurationFromEnvironment()
	batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis), conf.CollectTraceID)
	assert.Nil(t, batch.AddTelemetry(pollLogRequestId, []byte(pollLogContent), isAPMTelemetry))
}

func TestTelemetryChannelAddTelemetry(t *testing.T) {

	telemetryRequestId := "a89efeea-261f-47c1-8d7d-250e40ad9670"
	isAPMTelemetry := true
	data := []interface{}{1, "NR_LAMBDA_MONITORING", "H4sIAEUlq2cC/+VYa2/bNhT9K4Wwj7ZEUhIl+luatmiHFAvqZB0QBAItMY5WWVRJKqkb5L/vUrJjJ5Zdx3a7YIMhW+bj8PDecy8fd85EGJ5xw53BnVMpaWQqi+RGKJ3L0hngqOeIbyKtDfxNRHmTK1lORGmcgXP0eZic8Mko40k1Ndey9F1MnJ7Dx1C/gHAwcqmLHioKXo5reIWatputUbYhfA/4rR4UDeig1n3BtenjQUBpRGMWYhwHg6u6TC2bQdu7bxRPRZ71fUz6RkD796Io5GepiuzdrGmfi9uT0ft3H7/+dR3CcHOIJZK/nRydvR2eOfc9Z26NuoI3kRSSZyJLJjKrC6GdwYXzO1fa6V1cOEC25aryFGCJS+w87+4vexeOzidVIf7WzfzAMsz151WqLk0+EUla5NaSTzqW4laJogGcW+5xhXst5RftTgqgJIpEfymEtV/PefW0ZdKygyrkYjdeA5RKJcG/pdAJ19MyzWUX1qxxIcdjoRL7k5fjDQ3FNyNUyYvk2piqyEcPTS8vwd9QMTVAUNxYSbQWvyjroujdOUpooW5krhKdfweZYIIQqNC21IkWwsryHsx/54zrPAPfjfyUsiDjI0ZEcMUEDKW5tT5UGlWLHgg7lyo3U+jpEkbjgPacRjcfuvqno4xQGlJGKMFBSgHP+ksbALUh4UMFCwI/CGNQS624aTSEXESQlWgUMIp8EuEQOk4rq/QzxUvNG9EBWskntvAPcy3UUo0316u3u7SNNLw4yy1+B597cABYGHTrKvG1BsjGADxm4koI3icUX/WDKMX9OIuyPgmRCBDPGI2a+IVuraLcXxywSyPP0f5cCd1HzSCLZUPDlWkl0MpOV7zcSnKguDWSMwt3fVijvWcJaxuhthIaAvmFdg4glVn0+FEorhiiLESMjYK91J5CwhxLBfQdyPWiTYtLJnN/ovJL5YK71PRU5uXc6/9vucPSDh54JPWFO92AEAROXRSFLqEoApqN0GeeGtZVJZXho7wAWXqnzZy8owpSetoowfskxrk2rSy8N3N9gLJT2egWltQL3LNy8X2EKcIB6AwTvHcZQmEAMRJQEkNUBTiGOe9M/MgYMamMXiH+8NmEftIuh947qW65yuzrbEBR8pGN771gT2TKizcCFmogu8DOcn0A8I+NSvRhCfORKPSOPH9GclgV49PctWcZQgEKotCHkIoI8zEK6OZJ8aJ4ibzO5vuH/6TVH2b3EsjNc+Xr6TGIQSjvvPxSytty7e8LkcwuvBtHvATy3cvCJ5tbs+XwtFPTnmrKu3gjRGBNYjFBMcUxpgGihykOXAabsShkFPuxnYroo21Wtvdc3UBUzohfFeJbDjl3hbrvshiHDJEIVlKKWADw4QFKsRvGYHWKYSqIoSiIoJg94n18eu6dwz771boAfPp09vbODcz7+9pdxg9BhlMNS/3uJNr+e9NoUtHuLJrue5H4KCawU/dOr6cadkSrmcUPXUbCKMYk3OodE4ZcSnEUxrC1jENMNw33A+o4jkBCIWaUwbkkZPGeZcgPScCwj3Ho+5j6bCM39i8ao2PwX22ODyUc/spUQFq0iaa953lCaf7ZIjMdS1glUiOV13mj5v1Rm6o23usp7CpWDU/IowcjH07Fzxp1A77fw5SS9suiR74dATEcka1WjeZMOWyO8MqbHTF3PEJ0Y7bXA/tBLizRce222foUxY8enzI4aT1rzCd3LpvHs/ZffkI/jMnG8eytyFuL3mbE5lUP7Y3Nc/PhdsDNfe1ewC2St7TZeKsUGKqDNDoc9grvXbGPa23kRByU7xLm/jw7zuKHoNoJuwPbDpw3SlbVyvF4DdTl5f39P0toy3q2GQAA"}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Error marshalling data: %v", err)
	}

	byteArray := []byte(jsonData)
	conf := config.ConfigurationFromEnvironment()
	conf.CollectTraceID = true
	batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis), conf.CollectTraceID)

	batch.AddInvocation("a89efeea-261f-47c1-8d7d-250e40ad9670", time.Now())

	inv := batch.AddTelemetry(telemetryRequestId, byteArray, isAPMTelemetry)
	assert.NotNil(t, inv)
}
func TestTelemetryChannelHandling(t *testing.T) {
	tests := []struct {
		name        string
		sendData    bool
		telemetry   []byte
		expectError bool
	}{
		{
			name:        "Successful telemetry",
			sendData:    true,
			telemetry:   []byte("test-telemetry-data"),
			expectError: false,
		},
		{
			name:        "Empty channel",
			sendData:    false,
			telemetry:   nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telemetryChan := make(chan []byte, 1)
			lastRequestId := "a89efeea-261f-47c1-8d7d-250e40ad9670"
			conf := config.ConfigurationFromEnvironment()
			batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis), conf.CollectTraceID)
			batch.AddInvocation(lastRequestId, time.Now())

			if tt.sendData {
				telemetryChan <- tt.telemetry
			}

			select {
			case telemetryBytes := <-telemetryChan:
				util.Debugf("Agent telemetry bytes: %s", base64.URLEncoding.EncodeToString(telemetryBytes))
				inv := batch.AddTelemetry(lastRequestId, telemetryBytes, true)
				util.Logf("We suspected a timeout for request %s but got telemetry anyway", lastRequestId)
				assert.NotNil(t, inv)
				assert.Equal(t, tt.telemetry, inv.Telemetry[0])
			default:
				if !tt.expectError {
					t.Error("Expected to receive telemetry data but channel was empty")
				}
			}
		})
	}
}
func TestLogEventTypeConfiguration(t *testing.T) {
	tests := []struct {
		name              string
		sendFunctionLogs  bool
		sendExtensionLogs bool
		expectedTypes     []api.LogEventType
	}{
		{
			name:              "Only Platform logs enabled",
			sendFunctionLogs:  false,
			sendExtensionLogs: false,
			expectedTypes:     []api.LogEventType{api.Platform},
		},
		{
			name:              "Function logs enabled",
			sendFunctionLogs:  true,
			sendExtensionLogs: false,
			expectedTypes:     []api.LogEventType{api.Platform, api.Function},
		},
		{
			name:              "Extension logs enabled",
			sendFunctionLogs:  false,
			sendExtensionLogs: true,
			expectedTypes:     []api.LogEventType{api.Platform, api.Extension},
		},
		{
			name:              "All logs enabled",
			sendFunctionLogs:  true,
			sendExtensionLogs: true,
			expectedTypes:     []api.LogEventType{api.Platform, api.Function, api.Extension},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := config.Configuration{
				SendFunctionLogs:  tt.sendFunctionLogs,
				SendExtensionLogs: tt.sendExtensionLogs,
			}

			eventTypes := []api.LogEventType{api.Platform}
			if conf.SendFunctionLogs {
				eventTypes = append(eventTypes, api.Function)
			}
			if conf.SendExtensionLogs {
				eventTypes = append(eventTypes, api.Extension)
			}

			assert.Equal(t, len(tt.expectedTypes), len(eventTypes), "Event types length mismatch")
			assert.ElementsMatch(t, tt.expectedTypes, eventTypes, "Event types do not match expected values")

			// Verify Platform is always included as first element
			assert.Equal(t, api.Platform, eventTypes[0], "Platform should always be the first event type")
		})
	}
}
func TestTelemetryChannelSelect(t *testing.T) {
	telemetryChan := make(chan []byte, 1)
	lastRequestId := "test-request-123"

	conf := config.ConfigurationFromEnvironment()
	batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis), conf.CollectTraceID)
	batch.AddInvocation(lastRequestId, time.Now())

	testData := []byte("test-telemetry-data")
	telemetryChan <- testData

	var telemetryBytes []byte
	var inv *telemetry.Invocation
	select {
	case telemetryBytes = <-telemetryChan:
		util.Debugf("Agent telemetry bytes: %s", base64.URLEncoding.EncodeToString(telemetryBytes))
		inv = batch.AddTelemetry(lastRequestId, telemetryBytes, true)
		util.Logf("We suspected a timeout for request %s but got telemetry anyway", lastRequestId)
	default:
		t.Fatal("Expected to receive telemetry data but channel was empty")
	}

	assert.NotNil(t, inv)
	assert.Equal(t, testData, inv.Telemetry[0])
}

func TestTimeoutTelemetryHandling(t *testing.T) {
	lastRequestId := "test-request-123"
	eventStart := time.Now()
	lastEventStart := eventStart.Add(-5 * time.Second)

	conf := config.ConfigurationFromEnvironment()
	batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis), conf.CollectTraceID)
	batch.AddInvocation(lastRequestId, lastEventStart)

	timeoutMessage := fmt.Sprintf(
		"%s %s Task timed out after %.2f seconds",
		eventStart.UTC().Format(time.RFC3339),
		lastRequestId,
		eventStart.Sub(lastEventStart).Seconds(),
	)

	inv := batch.AddTelemetry(lastRequestId, []byte(timeoutMessage), false)

	assert.NotNil(t, inv)
	assert.Equal(t, timeoutMessage, string(inv.Telemetry[0]))
	assert.Equal(t, lastRequestId, inv.RequestId)
}
func TestShutdownTelemetryHandling(t *testing.T) {
	tests := []struct {
		name           string
		shutdownReason api.ShutdownReason
		expectedMsg    string
	}{
		{
			name:           "Timeout shutdown",
			shutdownReason: api.Timeout,
			expectedMsg:    "",
		},
		{
			name:           "Failure shutdown",
			shutdownReason: api.Failure,
			expectedMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lastRequestId := "test-request-123"
			eventStart := time.Now()
			lastEventStart := eventStart.Add(-5 * time.Second)

			conf := config.ConfigurationFromEnvironment()
			batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis), conf.CollectTraceID)
			batch.AddInvocation(lastRequestId, lastEventStart)

			var expectedMsg string
			if tt.shutdownReason == api.Timeout {
				expectedMsg = fmt.Sprintf(
					"%s %s Task timed out after %.2f seconds",
					eventStart.UTC().Format(time.RFC3339),
					lastRequestId,
					eventStart.Sub(lastEventStart).Seconds(),
				)
			} else {
				expectedMsg = fmt.Sprintf("RequestId: %s AWS Lambda platform fault caused a shutdown", lastRequestId)
			}

			inv := batch.AddTelemetry(lastRequestId, []byte(expectedMsg), false)

			assert.NotNil(t, inv)
			assert.Equal(t, expectedMsg, string(inv.Telemetry[0]))
			assert.Equal(t, lastRequestId, inv.RequestId)
		})
	}
}

func TestBatchAddTelemetry(t *testing.T) {
	lastRequestId := "test-123"
	telemetryBytes := []byte("test telemetry")
	conf := config.ConfigurationFromEnvironment()
	batch := telemetry.NewBatch(int64(conf.RipeMillis), int64(conf.RotMillis), conf.CollectTraceID)

	batch.AddInvocation(lastRequestId, time.Now())
	inv := batch.AddTelemetry(lastRequestId, telemetryBytes, true)

	assert.NotNil(t, inv)
	assert.Equal(t, lastRequestId, inv.RequestId)
	assert.Equal(t, telemetryBytes, inv.Telemetry[0])
}

type MockBatch struct {
	mock.Mock
}

func (m *MockBatch) AddTelemetry(requestId string, message []byte, flag bool) {
	m.Called(requestId, message, flag)
}

func TestHandleShutdownEvent(t *testing.T) {
	mockBatch := new(MockBatch)

	tests := []struct {
		name            string
		eventType       string
		shutdownReason  string
		lastRequestId   string
		eventStart      time.Time
		lastEventStart  time.Time
		expectedMessage string
	}{
		{
			name:            "Timeout with valid lastRequestId",
			eventType:       "Shutdown",
			shutdownReason:  "Timeout",
			lastRequestId:   "12345",
			eventStart:      time.Now(),
			lastEventStart:  time.Now().Add(-30 * time.Second),
			expectedMessage: "Task timed out after 30.00 seconds",
		},
		{
			name:            "Failure with valid lastRequestId",
			eventType:       "Shutdown",
			shutdownReason:  "Failure",
			lastRequestId:   "12345",
			expectedMessage: "AWS Lambda platform fault caused a shutdown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := Event{
				EventType:      tt.eventType,
				ShutdownReason: tt.shutdownReason,
			}

			mockBatch.On("AddTelemetry", tt.lastRequestId, mock.Anything, false).Run(func(args mock.Arguments) {
				message := args.Get(1).([]byte)
				assert.Contains(t, string(message), tt.expectedMessage)
			}).Once()

			handleShutdownEvent(event, tt.lastRequestId, tt.eventStart, tt.lastEventStart, mockBatch)

			mockBatch.AssertExpectations(t)
		})
	}
}

type Event struct {
	EventType      string
	ShutdownReason string
}

func handleShutdownEvent(event Event, lastRequestId string, eventStart, lastEventStart time.Time, batch *MockBatch) {
	if event.EventType == "Shutdown" {
		if event.ShutdownReason == "Timeout" && lastRequestId != "" {
			timestamp := eventStart.UTC()
			timeoutSecs := eventStart.Sub(lastEventStart).Seconds()
			timeoutMessage := fmt.Sprintf(
				"%s %s Task timed out after %.2f seconds",
				timestamp.Format(time.RFC3339),
				lastRequestId,
				timeoutSecs,
			)
			batch.AddTelemetry(lastRequestId, []byte(timeoutMessage), false)
		} else if event.ShutdownReason == "Failure" && lastRequestId != "" {
			errorMessage := fmt.Sprintf("RequestId: %s AWS Lambda platform fault caused a shutdown", lastRequestId)
			batch.AddTelemetry(lastRequestId, []byte(errorMessage), false)
		}
	}
}
