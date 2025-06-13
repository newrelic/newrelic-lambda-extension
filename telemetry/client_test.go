package telemetry

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/lambda/logserver"
	"github.com/newrelic/newrelic-lambda-extension/util"
	"github.com/stretchr/testify/assert"
)

const (
	clientTestingTimeout = 800 * time.Millisecond
	testARN              = "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go"
)

func TestClientSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPost)

		assert.Equal(t, r.Header.Get("Content-Encoding"), "gzip")
		assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
		assert.Equal(t, r.Header.Get("User-Agent"), "newrelic-lambda-extension")
		assert.Equal(t, r.Header.Get("X-License-Key"), "a mock license key")

		reqBytes, err := io.ReadAll(r.Body)
		if len(reqBytes) > 1000000 {
			w.WriteHeader(413)
			w.Write([]byte(""))
			return
		}

		assert.NoError(t, err)
		defer util.Close(r.Body)
		assert.NotEmpty(t, reqBytes)

		reqBody, err := util.Uncompress(reqBytes)
		assert.NoError(t, err)
		assert.NotEmpty(t, reqBody)

		var reqData RequestData
		assert.NoError(t, json.Unmarshal(reqBody, &reqData))
		assert.NotEmpty(t, reqData)

		w.WriteHeader(200)
		w.Write([]byte(""))
	}))

	defer srv.Close()

	client := NewWithHTTPClient(srv.Client(), "", "a mock license key", srv.URL, srv.URL, &Batch{}, false, clientTestingTimeout)

	ctx := context.Background()
	bytes := []byte("valid example payload")
	err, successCount := client.SendTelemetry(ctx, "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes})

	assert.NoError(t, err)
	assert.Equal(t, 1, successCount)

	client = New("", "mock license key", srv.URL, srv.URL, &Batch{}, false, clientTestingTimeout)
	assert.NotNil(t, client)
}

func TestClientSendPayloadTooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPost)

		assert.Equal(t, r.Header.Get("Content-Encoding"), "gzip")
		assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
		assert.Equal(t, r.Header.Get("User-Agent"), "newrelic-lambda-extension")
		assert.Equal(t, r.Header.Get("X-License-Key"), "a mock license key")

		reqBytes, err := io.ReadAll(r.Body)
		if len(reqBytes) > 1000000 {
			w.WriteHeader(413)
			w.Write([]byte(""))
			return
		}

		assert.NoError(t, err)
		defer util.Close(r.Body)
		assert.NotEmpty(t, reqBytes)

		reqBody, err := util.Uncompress(reqBytes)
		assert.NoError(t, err)
		assert.NotEmpty(t, reqBody)

		var reqData RequestData
		assert.NoError(t, json.Unmarshal(reqBody, &reqData))
		assert.NotEmpty(t, reqData)

		w.WriteHeader(200)
		w.Write([]byte(""))
	}))

	defer srv.Close()

	client := NewWithHTTPClient(srv.Client(), "", "a mock license key", srv.URL, srv.URL, &Batch{}, false, clientTestingTimeout)

	ctx := context.Background()
	bytes := []byte(payloadTooLarge)
	err, successCount := client.SendTelemetry(ctx, "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes})

	assert.NoError(t, err)
	assert.Equal(t, 0, successCount)

	client = New("", "mock license key", srv.URL, srv.URL, &Batch{}, false, clientTestingTimeout)
	assert.NotNil(t, client)
}

func TestClientSendPayloadTooLargeSplit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPost)

		assert.Equal(t, r.Header.Get("Content-Encoding"), "gzip")
		assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
		assert.Equal(t, r.Header.Get("User-Agent"), "newrelic-lambda-extension")
		assert.Equal(t, r.Header.Get("X-License-Key"), "a mock license key")

		reqBytes, err := io.ReadAll(r.Body)
		if len(reqBytes) > 1000000 {
			w.WriteHeader(413)
			w.Write([]byte(""))
			return
		}

		assert.NoError(t, err)
		defer util.Close(r.Body)
		assert.NotEmpty(t, reqBytes)

		reqBody, err := util.Uncompress(reqBytes)
		assert.NoError(t, err)
		assert.NotEmpty(t, reqBody)

		var reqData RequestData
		assert.NoError(t, json.Unmarshal(reqBody, &reqData))
		assert.NotEmpty(t, reqData)

		w.WriteHeader(200)
		w.Write([]byte(""))
	}))

	defer srv.Close()

	client := NewWithHTTPClient(srv.Client(), "", "a mock license key", srv.URL, srv.URL, &Batch{}, false, clientTestingTimeout)

	ctx := context.Background()
	bytes := []byte(payloadTooLarge)
	err, successCount := client.SendTelemetry(ctx, "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes, []byte("valid example payload")})

	assert.NoError(t, err)
	assert.Equal(t, 1, successCount)

	client = New("", "mock license key", srv.URL, srv.URL, &Batch{}, false, clientTestingTimeout)
	assert.NotNil(t, client)
}

func TestClientSendRetry(t *testing.T) {
	var count int32 = 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if atomic.LoadInt32(&count) == 0 {
			time.Sleep(100 * time.Millisecond)
		} else {
			assert.Equal(t, r.Method, http.MethodPost)

			assert.Equal(t, r.Header.Get("Content-Encoding"), "gzip")
			assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
			assert.Equal(t, r.Header.Get("User-Agent"), "newrelic-lambda-extension")
			assert.Equal(t, r.Header.Get("X-License-Key"), "a mock license key")

			reqBytes, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			defer util.Close(r.Body)
			assert.NotEmpty(t, reqBytes)

			reqBody, err := util.Uncompress(reqBytes)
			assert.NoError(t, err)
			assert.NotEmpty(t, reqBody)

			var reqData RequestData
			assert.NoError(t, json.Unmarshal(reqBody, &reqData))
			assert.NotEmpty(t, reqData)

			w.WriteHeader(200)
			w.Write([]byte(""))
		}
		atomic.AddInt32(&count, 1)
	}))

	defer srv.Close()

	httpClient := srv.Client()
	httpClient.Timeout = 50 * time.Millisecond
	client := NewWithHTTPClient(httpClient, "", "a mock license key", srv.URL, srv.URL, &Batch{}, false, clientTestingTimeout)

	ctx := context.Background()
	bytes := []byte("foobar")
	err, successCount := client.SendTelemetry(ctx, "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes})

	assert.NoError(t, err)
	assert.Equal(t, 1, successCount)
	assert.Equal(t, int32(2), atomic.LoadInt32(&count))
}

func TestClientSendServerTimeout(t *testing.T) {
	util.ConfigLogger(true, true)
	var count int32 = 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if atomic.LoadInt32(&count) == 0 {
			time.Sleep(800 * time.Millisecond)
		} else {
			assert.Equal(t, r.Method, http.MethodPost)

			assert.Equal(t, r.Header.Get("Content-Encoding"), "gzip")
			assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
			assert.Equal(t, r.Header.Get("User-Agent"), "newrelic-lambda-extension")
			assert.Equal(t, r.Header.Get("X-License-Key"), "a mock license key")

			reqBytes, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			defer util.Close(r.Body)
			assert.NotEmpty(t, reqBytes)

			reqBody, err := util.Uncompress(reqBytes)
			assert.NoError(t, err)
			assert.NotEmpty(t, reqBody)

			var reqData RequestData
			assert.NoError(t, json.Unmarshal(reqBody, &reqData))
			assert.NotEmpty(t, reqData)

			w.WriteHeader(200)
			w.Write([]byte(""))
		}
		atomic.AddInt32(&count, 1)
	}))

	defer srv.Close()

	clientTimeout := 300 * time.Millisecond

	httpClient := srv.Client()
	httpClient.Timeout = httpClientTimeout
	client := NewWithHTTPClient(httpClient, "", "a mock license key", srv.URL, srv.URL, &Batch{}, false, clientTimeout)

	ctx := context.Background()
	bytes := []byte("foobar")
	startTime := time.Now()
	err, successCount := client.SendTelemetry(ctx, "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes, bytes, bytes, bytes})
	endSend := time.Since(startTime)
	t.Logf("time to send: %s", endSend.String())

	assert.NoError(t, err)
	assert.Equal(t, 0, successCount)
	assert.Equal(t, int32(0), atomic.LoadInt32(&count))

	if endSend > clientTimeout+100*time.Millisecond {
		t.Errorf("took longer exceeded client timeout within a margin of error to fail sending the payload; took %s, expected %s", endSend.String(), clientTimeout.String())
	}
}

func TestClientSendAttemptFailsRetry(t *testing.T) {
	util.ConfigLogger(true, true)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusGatewayTimeout)
	}))

	defer srv.Close()

	clientTimeout := 2000 * time.Millisecond

	httpClient := srv.Client()
	httpClient.Timeout = 100 * time.Millisecond
	client := NewWithHTTPClient(httpClient, "", "a mock license key", srv.URL, srv.URL, &Batch{}, false, clientTimeout)

	ctx := context.Background()
	bytes := []byte("foobar")
	startTime := time.Now()
	err, successCount := client.SendTelemetry(ctx, "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes, bytes, bytes, bytes})
	endSend := time.Since(startTime)
	t.Logf("time to send: %s", endSend.String())

	assert.NoError(t, err)
	assert.Equal(t, 0, successCount)

	if endSend > clientTimeout+100*time.Millisecond {
		t.Errorf("took longer exceeded client timeout within a margin of error to fail sending the payload; took %s, expected %s", endSend.String(), clientTimeout.String())
	}
}

func TestSendFunctionLogsEmpty(t *testing.T) {
	util.ConfigLogger(true, true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, r.Method, http.MethodPost)

		assert.Equal(t, r.Header.Get("Content-Encoding"), "gzip")
		assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
		assert.Equal(t, r.Header.Get("User-Agent"), "newrelic-lambda-extension")
		assert.Equal(t, r.Header.Get("X-License-Key"), "a mock license key")

		reqBytes, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		defer util.Close(r.Body)
		assert.NotEmpty(t, reqBytes)

		reqBody, err := util.Uncompress(reqBytes)
		assert.NoError(t, err)
		assert.NotEmpty(t, reqBody)

		var reqData []DetailedFunctionLog
		assert.NoError(t, json.Unmarshal(reqBody, &reqData))
		assert.NotEmpty(t, reqData)

		w.WriteHeader(200)
		w.Write([]byte(""))
	}))
	defer srv.Close()

	clientTimeout := 1000 * time.Millisecond

	httpClient := srv.Client()
	httpClient.Timeout = 200 * time.Millisecond
	client := NewWithHTTPClient(httpClient, "", "a mock license key", srv.URL, srv.URL, &Batch{}, false, clientTimeout)

	// empty log bundle
	logLines := []logserver.LogLine{}
	var isAPMLambdaMode bool
	startSendLogs := time.Now()
	err := client.SendFunctionLogs(context.Background(), testARN, logLines, isAPMLambdaMode)
	sendDuration := time.Since(startSendLogs)

	if err != nil {
		t.Error(err)
	}

	if sendDuration > clientTimeout+10*time.Millisecond {
		t.Errorf("expected sending logs to take a maximum of %s, but took %s", clientTimeout.String(), sendDuration.String())
	}
}

func TestSendFunctionLogsSendingTimeout(t *testing.T) {
	util.ConfigLogger(true, true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer srv.Close()

	clientTimeout := 2000 * time.Millisecond

	httpClient := srv.Client()
	httpClient.Timeout = 100 * time.Millisecond
	client := NewWithHTTPClient(httpClient, "", "a mock license key", srv.URL, srv.URL, &Batch{}, false, clientTimeout)

	logLines := []logserver.LogLine{
		{
			Time:      time.Now(),
			RequestID: "test-request-1",
			Content:   []byte("test content"),
		},
		{
			Time:      time.Now(),
			RequestID: "test-request-2",
			Content:   []byte("test content"),
		},
		{
			Time:      time.Now(),
			RequestID: "test-request-3",
			Content:   []byte("test content"),
		},
	}

	startSendLogs := time.Now()
	var isAPMLambdaMode bool
	err := client.SendFunctionLogs(context.Background(), testARN, logLines, isAPMLambdaMode)
	sendDuration := time.Since(startSendLogs)

	if err != nil {
		t.Error(err)
	}

	if sendDuration > clientTimeout+100*time.Millisecond {
		t.Errorf("expected sending logs to take a maximum of %s, but took %s", clientTimeout.String(), sendDuration.String())
	}
}
func TestClientReachesDataTimeout(t *testing.T) {
	startTime := time.Now()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(400 * time.Millisecond)
	}))

	defer srv.Close()

	httpClient := srv.Client()
	httpClient.Timeout = 100 * time.Millisecond
	client := NewWithHTTPClient(httpClient, "", "a mock license key", srv.URL, srv.URL, &Batch{}, false, clientTestingTimeout)

	ctx := context.Background()
	bytes := []byte("foobar")
	err, successCount := client.SendTelemetry(ctx, "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes})
	assert.LessOrEqual(t, int(time.Since(startTime)), int(clientTestingTimeout+250*time.Millisecond))
	assert.NoError(t, err)
	assert.Equal(t, 0, successCount)
}

func TestClientUnreachableEndpoint(t *testing.T) {
	httpClient := &http.Client{
		Timeout: time.Millisecond * 1,
	}

	client := NewWithHTTPClient(httpClient, "", "a mock license key", "http://10.123.123.123:12345", "http://10.123.123.123:12345", &Batch{}, false, clientTestingTimeout)

	ctx := context.Background()
	bytes := []byte("foobar")
	err, successCount := client.SendTelemetry(ctx, "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes})

	assert.Nil(t, err)
	assert.Equal(t, 0, successCount)
}

func TestClientGetsHTTPError(t *testing.T) {
	startTime := time.Now()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))

	defer srv.Close()

	httpClient := srv.Client()
	httpClient.Timeout = 100 * time.Millisecond
	client := NewWithHTTPClient(httpClient, "", "a mock license key", srv.URL, srv.URL, &Batch{}, false, clientTestingTimeout)

	ctx := context.Background()
	bytes := []byte("foobar")
	err, successCount := client.SendTelemetry(ctx, "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes})
	assert.Less(t, int(time.Since(startTime)), int(clientTestingTimeout)) // should exit as soon as a non-timeout error occurs without retrying
	assert.NoError(t, err)
	assert.Equal(t, 0, successCount)
}

func TestGetInfraEndpointURL(t *testing.T) {
	assert.Equal(t, "barbaz", getInfraEndpointURL("foobar", "barbaz"))
	assert.Equal(t, InfraEndpointUS, getInfraEndpointURL("us license key", ""))
	assert.Equal(t, InfraEndpointEU, getInfraEndpointURL("eu license key", ""))
}

func TestGetLogEndpointURL(t *testing.T) {
	assert.Equal(t, "barbaz", getLogEndpointURL("foobar", "barbaz"))
	assert.Equal(t, LogEndpointUS, getLogEndpointURL("us mock license key", ""))
	assert.Equal(t, LogEndpointEU, getLogEndpointURL("eu mock license key", ""))
}

func TestGetNewRelicTags(t *testing.T) {

	tests := []struct {
		name         string
		common       map[string]interface{}
		expected     map[string]interface{}
		envTags      string
		envDelimiter string
	}{
		{
			name: "Add New Relic tags to common",
			common: map[string]interface{}{
				"plugin":    "testPlugin",
				"faas.arn":  "arn:aws:lambda:us-east-1:123456789012:function:testFunction",
				"faas.name": "testFunction",
			},
			expected: map[string]interface{}{
				"plugin":    "testPlugin",
				"faas.arn":  "arn:aws:lambda:us-east-1:123456789012:function:testFunction",
				"faas.name": "testFunction",
				"env":       "prod",
				"team":      "myTeam",
			},
			envTags:      "env:prod;team:myTeam",
			envDelimiter: ";",
		},
		{
			name: "Add New Relic tags to common if no delimiter is set",
			common: map[string]interface{}{
				"plugin":    "testPlugin",
				"faas.arn":  "arn:aws:lambda:us-east-1:123456789012:function:testFunction",
				"faas.name": "testFunction",
			},
			expected: map[string]interface{}{
				"plugin":    "testPlugin",
				"faas.arn":  "arn:aws:lambda:us-east-1:123456789012:function:testFunction",
				"faas.name": "testFunction",
				"env":       "prod",
				"team":      "myTeam",
			},
			envTags: "env:prod;team:myTeam",
		},
		{
			name: "No New Relic tags to common if delimiter is incorrect",
			common: map[string]interface{}{
				"plugin":    "testPlugin",
				"faas.arn":  "arn:aws:lambda:us-east-1:123456789012:function:testFunction",
				"faas.name": "testFunction",
			},
			expected: map[string]interface{}{
				"plugin":    "testPlugin",
				"faas.arn":  "arn:aws:lambda:us-east-1:123456789012:function:testFunction",
				"faas.name": "testFunction",
			},
			envTags:      "env:prod;team:myTeam",
			envDelimiter: ",",
		},
		{
			name: "No New Relic tags to add",
			common: map[string]interface{}{
				"plugin":    "testPlugin",
				"faas.arn":  "arn:aws:lambda:us-east-1:123456789012:function:testFunction",
				"faas.name": "testFunction",
			},
			expected: map[string]interface{}{
				"plugin":    "testPlugin",
				"faas.arn":  "arn:aws:lambda:us-east-1:123456789012:function:testFunction",
				"faas.name": "testFunction",
			},
			envTags:      "",
			envDelimiter: "",
		},
		{
			name: "No New Relic tags to add when enTags and envDelimiter are undeclared",
			common: map[string]interface{}{
				"plugin":    "testPlugin",
				"faas.arn":  "arn:aws:lambda:us-east-1:123456789012:function:testFunction",
				"faas.name": "testFunction",
			},
			expected: map[string]interface{}{
				"plugin":    "testPlugin",
				"faas.arn":  "arn:aws:lambda:us-east-1:123456789012:function:testFunction",
				"faas.name": "testFunction",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("NR_TAGS", tt.envTags)
			defer os.Unsetenv("NR_TAGS")
			_ = os.Setenv("NR_ENV_DELIMITER", tt.envDelimiter)
			defer os.Unsetenv("NR_ENV_DELIMITER")

			common := make(map[string]interface{}, len(tt.common))
			for k, v := range tt.common {
				common[k] = v
			}

			getNewRelicTags(common)

			for k, v := range tt.expected {
				if common[k] != v {
					t.Errorf("expected common[%q] to be %v, but got %v", k, v, common[k])
				}
			}
			for k := range common {
				if _, ok := tt.expected[k]; !ok {
					t.Errorf("unexpected key %q in common map", k)
				}
			}
		})
	}
}
