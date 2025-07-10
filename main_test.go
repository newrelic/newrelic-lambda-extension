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

var testDoneChan chan struct{}

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

func TestGetLambdaARN(t *testing.T) {
	originalRegion := os.Getenv("AWS_REGION")
	originalDefaultRegion := os.Getenv("AWS_DEFAULT_REGION")

	defer func() {
		os.Setenv("AWS_REGION", originalRegion)
		os.Setenv("AWS_DEFAULT_REGION", originalDefaultRegion)
	}()

	tests := []struct {
		name             string
		awsAccountId     string
		awsLambdaName    string
		awsRegion        string
		awsDefaultRegion string
		expectedARN      string
	}{
		{
			name:             "Standard case with AWS_REGION set",
			awsAccountId:     "123456789012",
			awsLambdaName:    "my-function",
			awsRegion:        "us-east-1",
			awsDefaultRegion: "",
			expectedARN:      "arn:aws:lambda:us-east-1:123456789012:function:my-function",
		},
		{
			name:             "AWS_REGION not set, fallback to AWS_DEFAULT_REGION",
			awsAccountId:     "123456789012",
			awsLambdaName:    "my-function",
			awsRegion:        "",
			awsDefaultRegion: "us-west-2",
			expectedARN:      "arn:aws:lambda:us-west-2:123456789012:function:my-function",
		},
		{
			name:             "Both AWS_REGION and AWS_DEFAULT_REGION set, AWS_REGION takes precedence",
			awsAccountId:     "123456789012",
			awsLambdaName:    "my-function",
			awsRegion:        "us-east-1",
			awsDefaultRegion: "us-west-2",
			expectedARN:      "arn:aws:lambda:us-east-1:123456789012:function:my-function",
		},
		{
			name:             "Neither region environment variable set",
			awsAccountId:     "123456789012",
			awsLambdaName:    "my-function",
			awsRegion:        "",
			awsDefaultRegion: "",
			expectedARN:      "arn:aws:lambda::123456789012:function:my-function",
		},
		{
			name:             "Function name with hyphens and underscores",
			awsAccountId:     "987654321098",
			awsLambdaName:    "my-complex_function-name",
			awsRegion:        "eu-west-1",
			awsDefaultRegion: "",
			expectedARN:      "arn:aws:lambda:eu-west-1:987654321098:function:my-complex_function-name",
		},
		{
			name:             "Different AWS region",
			awsAccountId:     "555666777888",
			awsLambdaName:    "test-function",
			awsRegion:        "ap-southeast-1",
			awsDefaultRegion: "",
			expectedARN:      "arn:aws:lambda:ap-southeast-1:555666777888:function:test-function",
		},
		{
			name:             "Empty account ID",
			awsAccountId:     "",
			awsLambdaName:    "my-function",
			awsRegion:        "us-east-1",
			awsDefaultRegion: "",
			expectedARN:      "arn:aws:lambda:us-east-1::function:my-function",
		},
		{
			name:             "Empty function name",
			awsAccountId:     "123456789012",
			awsLambdaName:    "",
			awsRegion:        "us-east-1",
			awsDefaultRegion: "",
			expectedARN:      "arn:aws:lambda:us-east-1:123456789012:function:",
		},
		{
			name:             "Long function name",
			awsAccountId:     "123456789012",
			awsLambdaName:    "very-long-function-name-that-might-be-used-in-some-cases",
			awsRegion:        "us-east-1",
			awsDefaultRegion: "",
			expectedARN:      "arn:aws:lambda:us-east-1:123456789012:function:very-long-function-name-that-might-be-used-in-some-cases",
		},
		{
			name:             "GovCloud region",
			awsAccountId:     "123456789012",
			awsLambdaName:    "gov-function",
			awsRegion:        "us-gov-west-1",
			awsDefaultRegion: "",
			expectedARN:      "arn:aws:lambda:us-gov-west-1:123456789012:function:gov-function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("AWS_REGION", tt.awsRegion)
			os.Setenv("AWS_DEFAULT_REGION", tt.awsDefaultRegion)

			result := getLambdaARN(tt.awsAccountId, tt.awsLambdaName)

			if result != tt.expectedARN {
				t.Errorf("getLambdaARN() = %v, want %v", result, tt.expectedARN)
			}
		})
	}
}

func TestGetLambdaARN_EnvironmentVariableHandling(t *testing.T) {
	originalRegion := os.Getenv("AWS_REGION")
	originalDefaultRegion := os.Getenv("AWS_DEFAULT_REGION")

	defer func() {
		os.Setenv("AWS_REGION", originalRegion)
		os.Setenv("AWS_DEFAULT_REGION", originalDefaultRegion)
	}()

	os.Setenv("AWS_REGION", "primary-region")
	os.Setenv("AWS_DEFAULT_REGION", "fallback-region")

	result := getLambdaARN("123456789012", "test-function")
	expected := "arn:aws:lambda:primary-region:123456789012:function:test-function"

	if result != expected {
		t.Errorf("Expected AWS_REGION to take precedence. Got %v, want %v", result, expected)
	}

	os.Setenv("AWS_REGION", "")
	os.Setenv("AWS_DEFAULT_REGION", "fallback-region")

	result = getLambdaARN("123456789012", "test-function")
	expected = "arn:aws:lambda:fallback-region:123456789012:function:test-function"

	if result != expected {
		t.Errorf("Expected fallback to AWS_DEFAULT_REGION. Got %v, want %v", result, expected)
	}
}
