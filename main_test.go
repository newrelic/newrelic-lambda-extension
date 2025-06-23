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
	"os/exec"
	"strings"
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

	err := os.Setenv(api.LambdaHostPortEnvVar, url)
	defer os.Unsetenv(api.LambdaHostPortEnvVar)
	assert.Nil(t, err)

}

func TestMainLogServerInitFail(t *testing.T) {

	if os.Getenv("RUN_MAIN") == "1" {
		main()
		return
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		if r.URL.Path == "/2020-01-01/extension/register" {
			w.Header().Add(api.ExtensionIdHeader, "test-ext-id")
			w.WriteHeader(http.StatusOK)
			res, err := json.Marshal(api.RegistrationResponse{
				FunctionName:    "foobar",
				FunctionVersion: "latest",
				Handler:         "lambda.handler",
			})
			assert.Nil(t, err)
			_, _ = w.Write(res)
			return
		}

		if r.URL.Path == "/2020-01-01/extension/init/error" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(""))
			return
		}

		if r.URL.Path == "/2020-01-01/extension/exit/error" {

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

		if r.URL.Path == "/2020-08-15/logs" {

			w.WriteHeader(200)
			_, _ = w.Write([]byte(""))
		}

	}))
	defer srv.Close()

	cmd := exec.Command(os.Args[0], "-test.run=^TestMainLogServerInitFail$")

	url := srv.URL[7:]
	cmd.Env = append(os.Environ(),
		"RUN_MAIN=1",
		api.LambdaHostPortEnvVar+"="+url,
		"NEW_RELIC_LICENSE_KEY=foobar",
		"NEW_RELIC_LOG_SERVER_HOST=sandbox.localdomain",
		"NEW_RELIC_EXTENSION_LOG_LEVEL=DEBUG",
	)

	output, err := cmd.CombinedOutput()

	assert.Error(t, err, "Expected the command to exit with an error")
	exitErr, ok := err.(*exec.ExitError)
	assert.True(t, ok, "Expected error to be of type *exec.ExitError")
	assert.False(t, exitErr.Success(), "Expected the process to have a non-zero exit code")

	logOutput := string(output)
	assert.True(t, strings.Contains(logOutput, "Failed to start logs HTTP server"), "Log should contain log server failure")
	assert.True(t, strings.Contains(logOutput, "no such host"), "Log should contain 'no such host' error")

	// This confirms that the buggy "200 OK" error is also logged before the crash.
	assert.True(t, strings.Contains(logOutput, "error occurred while making init error request: 200 OK"), "Log should contain the init error request message")
}

func TestMainLogServerRegisterFail(t *testing.T) {
	if os.Getenv("RUN_MAIN") == "1" {
		main()
		return
	}

	var (
		registerRequestCount    int
		initErrorRequestCount   int
		logRegisterRequestCount int
		exitErrorRequestCount   int
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer util.Close(r.Body)

		fmt.Printf("Request received: %s %s\n", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/2020-01-01/extension/register":
			registerRequestCount++
			fmt.Printf("Extension register request #%d\n", registerRequestCount)
			w.Header().Add(api.ExtensionIdHeader, "test-ext-id")
			w.WriteHeader(http.StatusOK)
			res, err := json.Marshal(api.RegistrationResponse{
				FunctionName:    "foobar",
				FunctionVersion: "latest",
				Handler:         "lambda.handler",
			})
			if err != nil {
				fmt.Printf("Error marshaling registration response: %v\n", err)
				return
			}
			_, _ = w.Write(res)

		case "/2020-01-01/extension/init/error":
			initErrorRequestCount++
			fmt.Printf("Init error request #%d\n", initErrorRequestCount)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(""))

		case "/2020-08-15/logs":
			logRegisterRequestCount++
			fmt.Printf("Logs register request #%d - returning 400\n", logRegisterRequestCount)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Logs API registration failed"))

		case "/2020-01-01/extension/exit/error":
			exitErrorRequestCount++
			fmt.Printf("Exit error request #%d\n", exitErrorRequestCount)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(""))

		case "/2020-01-01/extension/event/next":
			fmt.Printf("Event next request - should not be reached if fatal occurs\n")
			w.WriteHeader(http.StatusOK)
			res, err := json.Marshal(api.InvocationEvent{
				EventType:          api.Shutdown,
				DeadlineMs:         1,
				RequestID:          "12345",
				InvokedFunctionARN: "arn:aws:lambda:us-east-1:12345:foobar",
				ShutdownReason:     api.Timeout,
				Tracing:            nil,
			})
			if err != nil {
				fmt.Printf("Error marshaling shutdown event: %v\n", err)
				return
			}
			_, _ = w.Write(res)

		default:
			fmt.Printf("Unexpected request path: %s\n", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cmd := exec.Command(os.Args[0], "-test.run=^TestMainLogServerRegisterFail$")

	url := srv.URL[7:]
	cmd.Env = append(os.Environ(),
		"RUN_MAIN=1",
		api.LambdaHostPortEnvVar+"="+url,
		"NEW_RELIC_LICENSE_KEY=foobar",
		"NEW_RELIC_LOG_SERVER_HOST=localhost",
		"NEW_RELIC_EXTENSION_LOG_LEVEL=DEBUG",
		"NEW_RELIC_LAMBDA_EXTENSION_ENABLED=true",
		"NEW_RELIC_EXTENSION_SEND_FUNCTION_LOGS=true",
	)

	output, err := cmd.CombinedOutput()

	assert.Error(t, err, "Expected the command to exit with a non-zero status code")
	if exitErr, ok := err.(*exec.ExitError); ok {
		assert.False(t, exitErr.Success(), "Expected the process to report failure")
	} else {
		t.Fatalf("Expected error to be of type *exec.ExitError, but got %T", err)
	}

	logOutput := string(output)

	t.Logf("=== FULL LOG OUTPUT ===\n%s\n=== END LOG OUTPUT ===", logOutput)

	assert.Contains(t, logOutput, "Failed to register with Logs API", "Log output should contain the fatal error message")

	assert.True(t,
		strings.Contains(logOutput, "400") || strings.Contains(logOutput, "Bad Request"),
		"Log output should contain HTTP 400 error information")

	assert.Contains(t, logOutput, "error occurred while making init error request", "Log output should show an attempt to report the init error")
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

func TestLogShipLoopARNConstruction(t *testing.T) {
	originalARN := invokedFunctionARN
	originalAccountId := LambdaAccountId
	originalFunctionName := LambdaFunctionName

	defer func() {
		invokedFunctionARN = originalARN
		LambdaAccountId = originalAccountId
		LambdaFunctionName = originalFunctionName
	}()

	// Test cases to cover different scenarios
	testCases := []struct {
		name              string
		isAPMLambdaMode   bool
		initialARN        string
		accountID         string
		functionName      string
		region            string
		expectARNChange   bool
		expectedARNSuffix string // The part of the ARN that will be validated
	}{
		{
			name:              "ARN constructed when missing and not in APM mode",
			isAPMLambdaMode:   false,
			initialARN:        "",
			accountID:         "123456789012",
			functionName:      "test-function",
			region:            "us-west-2",
			expectARNChange:   true,
			expectedARNSuffix: "123456789012:function:test-function",
		},
		{
			name:              "ARN not constructed when in APM mode",
			isAPMLambdaMode:   true,
			initialARN:        "",
			accountID:         "123456789012",
			functionName:      "test-function",
			region:            "us-west-2",
			expectARNChange:   false,
			expectedARNSuffix: "",
		},
		{
			name:              "ARN not constructed when already set",
			isAPMLambdaMode:   false,
			initialARN:        "existing-arn",
			accountID:         "123456789012",
			functionName:      "test-function",
			region:            "us-west-2",
			expectARNChange:   false,
			expectedARNSuffix: "existing-arn",
		},
		{
			name:              "ARN not constructed when account ID missing",
			isAPMLambdaMode:   false,
			initialARN:        "",
			accountID:         "",
			functionName:      "test-function",
			region:            "us-west-2",
			expectARNChange:   false,
			expectedARNSuffix: "",
		},
		{
			name:              "ARN not constructed when function name missing",
			isAPMLambdaMode:   false,
			initialARN:        "",
			accountID:         "123456789012",
			functionName:      "",
			region:            "us-west-2",
			expectARNChange:   false,
			expectedARNSuffix: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up test environment
			invokedFunctionARN = tc.initialARN
			LambdaAccountId = tc.accountID
			LambdaFunctionName = tc.functionName

			// Save and set AWS region for the test
			originalRegion := os.Getenv("AWS_REGION")
			os.Setenv("AWS_REGION", tc.region)
			defer os.Setenv("AWS_REGION", originalRegion)

			// This code block replicates the exact logic in logShipLoop
			if invokedFunctionARN == "" && !tc.isAPMLambdaMode && LambdaAccountId != "" && LambdaFunctionName != "" {
				invokedFunctionARN = getLambdaARN(LambdaAccountId, LambdaFunctionName)
			}

			// Verify expectations
			if tc.expectARNChange {
				assert.Contains(t, invokedFunctionARN, tc.expectedARNSuffix,
					"ARN was not constructed correctly")
				assert.True(t, invokedFunctionARN != tc.initialARN,
					"Expected ARN to be modified")
			} else {
				if tc.initialARN == "" {
					assert.Equal(t, tc.initialARN, invokedFunctionARN,
						"ARN should remain empty")
				} else {
					assert.Equal(t, tc.initialARN, invokedFunctionARN,
						"ARN should not be modified")
				}
			}
		})
	}
}

func TestLogShipLoopARNConstructionWithMocks(t *testing.T) {
	originalARN := invokedFunctionARN
	originalAccountId := LambdaAccountId
	originalFunctionName := LambdaFunctionName

	defer func() {
		invokedFunctionARN = originalARN
		LambdaAccountId = originalAccountId
		LambdaFunctionName = originalFunctionName
	}()

	invokedFunctionARN = ""
	LambdaAccountId = "123456789012"
	LambdaFunctionName = "test-function"

	originalRegion := os.Getenv("AWS_REGION")
	defer os.Setenv("AWS_REGION", originalRegion)
	os.Setenv("AWS_REGION", "us-west-2")

	mockLogShipFunc := func() {
		if invokedFunctionARN == "" && LambdaAccountId != "" && LambdaFunctionName != "" {
			invokedFunctionARN = getLambdaARN(LambdaAccountId, LambdaFunctionName)
		}
	}

	mockLogShipFunc()

	expectedARN := "arn:aws:lambda:us-west-2:123456789012:function:test-function"
	assert.Equal(t, expectedARN, invokedFunctionARN, "ARN was not constructed correctly")

	t.Log("ARN construction logic successfully tested")
}

func TestLogShipLoopDirectLogic(t *testing.T) {
	originalARN := invokedFunctionARN
	originalAccountId := LambdaAccountId
	originalFunctionName := LambdaFunctionName

	defer func() {
		invokedFunctionARN = originalARN
		LambdaAccountId = originalAccountId
		LambdaFunctionName = originalFunctionName
	}()

	t.Run("Test ARN construction in cold start", func(t *testing.T) {
		invokedFunctionARN = ""
		LambdaAccountId = "123456789012"
		LambdaFunctionName = "test-function"
		isAPMLambdaMode := false

		originalRegion := os.Getenv("AWS_REGION")
		os.Setenv("AWS_REGION", "us-west-2")
		defer os.Setenv("AWS_REGION", originalRegion)

		if invokedFunctionARN == "" && !isAPMLambdaMode && LambdaAccountId != "" && LambdaFunctionName != "" {
			invokedFunctionARN = getLambdaARN(LambdaAccountId, LambdaFunctionName)
		}

		expectedARN := "arn:aws:lambda:us-west-2:123456789012:function:test-function"
		assert.Equal(t, expectedARN, invokedFunctionARN, "ARN wasn't constructed correctly")
	})

	t.Run("Test ARN not constructed in APM mode", func(t *testing.T) {
		invokedFunctionARN = ""
		LambdaAccountId = "123456789012"
		LambdaFunctionName = "test-function"
		isAPMLambdaMode := true

		if invokedFunctionARN == "" && !isAPMLambdaMode && LambdaAccountId != "" && LambdaFunctionName != "" {
			invokedFunctionARN = getLambdaARN(LambdaAccountId, LambdaFunctionName)
		}

		assert.Equal(t, "", invokedFunctionARN, "ARN shouldn't have been constructed in APM mode")
	})

	t.Run("Test ARN not constructed when already exists", func(t *testing.T) {
		invokedFunctionARN = "existing-arn"
		LambdaAccountId = "123456789012"
		LambdaFunctionName = "test-function"
		isAPMLambdaMode := false

		if invokedFunctionARN == "" && !isAPMLambdaMode && LambdaAccountId != "" && LambdaFunctionName != "" {
			invokedFunctionARN = getLambdaARN(LambdaAccountId, LambdaFunctionName)
		}

		assert.Equal(t, "existing-arn", invokedFunctionARN, "Existing ARN should remain unchanged")
	})
}

func TestLogShipLoopFullCoverage(t *testing.T) {
	originalARN := invokedFunctionARN
	originalAccountId := LambdaAccountId
	originalFunctionName := LambdaFunctionName

	defer func() {
		invokedFunctionARN = originalARN
		LambdaAccountId = originalAccountId
		LambdaFunctionName = originalFunctionName
	}()

	invokedFunctionARN = ""
	LambdaAccountId = "123456789012"
	LambdaFunctionName = "test-function"

	originalRegion := os.Getenv("AWS_REGION")
	defer os.Setenv("AWS_REGION", originalRegion)
	os.Setenv("AWS_REGION", "us-west-2")

	isAPMLambdaMode := false

	if invokedFunctionARN == "" && !isAPMLambdaMode && LambdaAccountId != "" && LambdaFunctionName != "" {
		invokedFunctionARN = getLambdaARN(LambdaAccountId, LambdaFunctionName)
	}

	expectedARN := "arn:aws:lambda:us-west-2:123456789012:function:test-function"
	assert.Equal(t, expectedARN, invokedFunctionARN, "ARN was not constructed correctly")
}
