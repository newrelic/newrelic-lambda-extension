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
