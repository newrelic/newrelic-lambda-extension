//go:build !race
// +build !race

package logserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/config"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogServer(t *testing.T) {
	logs, err := startInternal("localhost")
	assert.NoError(t, err)

	testEvents := []api.LogEvent{
		{
			Time: time.Now(),
			Type: "platform.report",
			Record: map[string]interface{}{
				"metrics": map[string]float64{
					"durationMs":       25.3,
					"billedDurationMs": 100.0,
					"memorySizeMB":     128.0,
					"maxMemoryUsedMB":  73.5,
					"initDurationMs":   202.0,
				},
				"requestId": "testRequestId",
			},
		},
	}

	testEventBytes, err := json.Marshal(testEvents)
	assert.NoError(t, err)

	realEndpoint := fmt.Sprintf("http://localhost:%d", logs.Port())
	req, err := http.NewRequest("POST", realEndpoint, bytes.NewBuffer(testEventBytes))
	assert.NoError(t, err)

	client := http.Client{}
	res, err := client.Do(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, http.NoBody, res.Body)

	logLines := logs.PollPlatformChannel()

	assert.Equal(t, 1, len(logLines))
	assert.Equal(t, "REPORT RequestId: testRequestId\tDuration: 25.30 ms\tBilled Duration: 100 ms\tMemory Size: 128 MB\tMax Memory Used: 74 MB\tInit Duration: 202.00 ms", string(logLines[0].Content))

	assert.Nil(t, logs.Close())
}

func TestFunctionLogs(t *testing.T) {
	logs, err := startInternal("localhost")
	assert.NoError(t, err)

	testEvents := []api.LogEvent{
		{
			Time: time.Now().Add(-100 * time.Millisecond),
			Type: "platform.start",
			Record: map[string]interface{}{
				"requestId": "testRequestId",
			},
		},
		{
			Time:   time.Now().Add(-50 * time.Millisecond),
			Type:   "function",
			Record: "log line 1",
		},
	}

	testEventBytes, err := json.Marshal(testEvents)
	assert.NoError(t, err)

	realEndpoint := fmt.Sprintf("http://localhost:%d", logs.Port())
	req, err := http.NewRequest("POST", realEndpoint, bytes.NewBuffer(testEventBytes))
	assert.NoError(t, err)

	client := http.Client{}
	go func() {
		res, err := client.Do(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)
		assert.Equal(t, http.NoBody, res.Body)
	}()

	logLines, _ := logs.AwaitFunctionLogs()

	assert.Equal(t, 1, len(logLines))
	assert.Equal(t, "log line 1", string(logLines[0].Content))
	assert.Equal(t, "testRequestId", logLines[0].RequestID)

	testEvents2 := []api.LogEvent{
		{
			Time:   time.Now().Add(500 * time.Millisecond),
			Type:   "function",
			Record: "log line 2",
		},
	}

	testEventBytes, err = json.Marshal(testEvents2)
	assert.NoError(t, err)

	req, err = http.NewRequest("POST", realEndpoint, bytes.NewBuffer(testEventBytes))
	assert.NoError(t, err)

	go func() {
		res, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)
		assert.Equal(t, http.NoBody, res.Body)
	}()

	logLines2, _ := logs.AwaitFunctionLogs()

	assert.Equal(t, 1, len(logLines2))
	assert.Equal(t, "log line 2", string(logLines2[0].Content))
	assert.Equal(t, "testRequestId", logLines2[0].RequestID)

	testRequestId := "abcdef01-a2b3-4321-cd89-0123456789ab"

	testEvents3 := []api.LogEvent{
		{
			Time:   time.Now().Add(600 * time.Millisecond),
			Type:   "platform.start",
			Record: "RequestId: " + testRequestId,
		},
		{
			Time:   time.Now().Add(700 * time.Millisecond),
			Type:   "function",
			Record: "log line 3, for testing start line record as string",
		},
	}

	testEventBytes, err = json.Marshal(testEvents3)
	assert.NoError(t, err)

	req, err = http.NewRequest("POST", realEndpoint, bytes.NewBuffer(testEventBytes))
	assert.NoError(t, err)

	go func() {
		res, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)
		assert.Equal(t, http.NoBody, res.Body)
	}()

	logLines3, _ := logs.AwaitFunctionLogs()

	assert.Equal(t, 1, len(logLines3))
	assert.Equal(t, "log line 3, for testing start line record as string", string(logLines3[0].Content))
	assert.Equal(t, testRequestId, logLines3[0].RequestID)

	platformMetricString := "REPORT RequestId: " + testRequestId + "\tDuration: 25.30 ms\tBilled Duration: 100 ms\tMemory Size: 128 MB\tMax Memory Used: 74 MB\tInit Duration: 202.00 ms"

	testEvents4 := []api.LogEvent{
		{
			Time:   time.Now().Add(800 * time.Millisecond),
			Type:   "platform.report",
			Record: platformMetricString,
		},
		{
			Time:   time.Now().Add(900 * time.Millisecond),
			Type:   "function",
			Record: "log line 4, testing platform metrics as string",
		},
	}

	testEventBytes, err = json.Marshal(testEvents4)
	assert.NoError(t, err)

	req, err = http.NewRequest("POST", realEndpoint, bytes.NewBuffer(testEventBytes))
	assert.NoError(t, err)

	go func() {
		res, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)
		assert.Equal(t, http.NoBody, res.Body)
	}()

	logLines4, _ := logs.AwaitFunctionLogs()

	assert.Equal(t, 1, len(logLines4))
	assert.Equal(t, "log line 4, testing platform metrics as string", string(logLines4[0].Content))
	assert.Equal(t, testRequestId, logLines4[0].RequestID)

	assert.Nil(t, logs.Close())
}

func TestLogServerStart(t *testing.T) {
	logs, err := Start(&config.Configuration{LogServerHost: "localhost"})
	assert.NoError(t, err)
	assert.Nil(t, logs.Close())
}

func TestLogServerCloseShutdownFlag(t *testing.T) {
	logServer, err := startInternal("localhost")
	require.NoError(t, err)
	require.NotNil(t, logServer)

	logServer.shutdownLock.RLock()
	initialShutdownState := logServer.isShuttingDown
	logServer.shutdownLock.RUnlock()
	assert.False(t, initialShutdownState, "LogServer should not be in shutdown state initially")

	err = logServer.Close()
	assert.NoError(t, err)

	logServer.shutdownLock.RLock()
	finalShutdownState := logServer.isShuttingDown
	logServer.shutdownLock.RUnlock()
	assert.True(t, finalShutdownState, "LogServer should be in shutdown state after Close()")
}

func TestLogServerHandlerDuringShutdown(t *testing.T) {
	logServer, err := startInternal("localhost")
	require.NoError(t, err)
	require.NotNil(t, logServer)

	recorder := httptest.NewRecorder()

	logEvent := []api.LogEvent{
		{
			Time:   time.Now(),
			Type:   "function",
			Record: "test log event",
		},
	}

	jsonData, err := json.Marshal(logEvent)
	require.NoError(t, err)

	request := httptest.NewRequest("POST", "/", bytes.NewBuffer(jsonData))

	logServer.shutdownLock.Lock()
	logServer.isShuttingDown = true
	logServer.shutdownLock.Unlock()

	logServer.handler(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)

	select {
	case logs := <-logServer.functionLogChan:
		t.Fatalf("Expected no logs to be processed, but got: %v", logs)
	default:
	}

	logServer.Close()
}

func SendFunctionLogsContinuously(logServer *LogServer, t *testing.T) {
	for i := 0; i < 5000; i++ {
		logEvent := []api.LogEvent{
			{
				Time:   time.Now(),
				Type:   "function",
				Record: fmt.Sprintf("test log event %d", i),
			},
		}

		jsonData, err := json.Marshal(logEvent)
		require.NoError(t, err)

		request := httptest.NewRequest("POST", "/", bytes.NewBuffer(jsonData))
		recorder := httptest.NewRecorder()
		logServer.handler(recorder, request)
	}
}

func TestLogServerShutdownDuringRequests(t *testing.T) {
	logServer, err := startInternal("localhost")
	require.NoError(t, err)
	require.NotNil(t, logServer)

	done := make(chan struct{})

	go func() {
		for {
			_, more := logServer.AwaitFunctionLogs()
			if !more {
				return
			}
		}
	}()

	go SendFunctionLogsContinuously(logServer, t)
	time.Sleep(10 * time.Millisecond)
	err = logServer.Close()
	assert.NoError(t, err, "Server should close without errors")
	close(done)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out")
	}
}
