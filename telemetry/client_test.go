package telemetry

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/util"
	"github.com/stretchr/testify/assert"
)

func TestClientSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, http.MethodPost)

		assert.Equal(t, r.Header.Get("Content-Encoding"), "gzip")
		assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
		assert.Equal(t, r.Header.Get("User-Agent"), "newrelic-lambda-extension")
		assert.Equal(t, r.Header.Get("X-License-Key"), "a mock license key")

		reqBytes, err := ioutil.ReadAll(r.Body)
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

	client := NewWithHTTPClient(srv.Client(), "", "a mock license key", srv.URL, srv.URL, &Batch{}, false)

	ctx := context.Background()
	bytes := []byte("foobar")
	err, successCount := client.SendTelemetry(ctx, "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes})

	assert.NoError(t, err)
	assert.Equal(t, 1, successCount)

	client = New("", "mock license key", srv.URL, srv.URL, &Batch{}, false)
	assert.NotNil(t, client)
}

func TestClientSendRetry(t *testing.T) {
	var count int32 = 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if atomic.LoadInt32(&count) == 0 {
			time.Sleep(300 * time.Millisecond)
		} else {
			assert.Equal(t, r.Method, http.MethodPost)

			assert.Equal(t, r.Header.Get("Content-Encoding"), "gzip")
			assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
			assert.Equal(t, r.Header.Get("User-Agent"), "newrelic-lambda-extension")
			assert.Equal(t, r.Header.Get("X-License-Key"), "a mock license key")

			reqBytes, err := ioutil.ReadAll(r.Body)
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
	httpClient.Timeout = 200 * time.Millisecond
	client := NewWithHTTPClient(httpClient, "", "a mock license key", srv.URL, srv.URL, &Batch{}, false)

	ctx := context.Background()
	bytes := []byte("foobar")
	err, successCount := client.SendTelemetry(ctx, "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes})

	assert.NoError(t, err)
	assert.Equal(t, 1, successCount)
	assert.Equal(t, int32(2), atomic.LoadInt32(&count))
}

func TestClientSendOutOfRetries(t *testing.T) {
	var count int32 = 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		time.Sleep(300 * time.Millisecond)
	}))

	defer srv.Close()

	httpClient := srv.Client()
	httpClient.Timeout = 200 * time.Millisecond
	client := NewWithHTTPClient(httpClient, "", "a mock license key", srv.URL, srv.URL, &Batch{}, false)

	ctx := context.Background()
	bytes := []byte("foobar")
	err, successCount := client.SendTelemetry(ctx, "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes})

	assert.NoError(t, err)
	assert.Equal(t, 0, successCount)
	assert.Equal(t, int32(retries), atomic.LoadInt32(&count))
}

func TestClientUnreachableEndpoint(t *testing.T) {
	httpClient := &http.Client{
		Timeout: time.Millisecond * 1,
	}

	client := NewWithHTTPClient(httpClient, "", "a mock license key", "http://10.123.123.123:12345", "http://10.123.123.123:12345", &Batch{}, false)

	ctx := context.Background()
	bytes := []byte("foobar")
	err, successCount := client.SendTelemetry(ctx, "arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes})

	assert.Nil(t, err)
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
