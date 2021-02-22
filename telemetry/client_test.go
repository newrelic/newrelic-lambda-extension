package telemetry

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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

	client := NewWithHTTPClient(srv.Client(), "", "a mock license key", &srv.URL, &srv.URL)

	bytes := []byte("foobar")
	err, successCount := client.SendTelemetry("arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes})

	assert.NoError(t, err)
	assert.Equal(t, 1, successCount)
}

func TestClientSendRetry(t *testing.T) {
	count := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if count == 0 {
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
		count += 1
	}))

	defer srv.Close()

	httpClient := srv.Client()
	httpClient.Timeout = 200 * time.Millisecond
	client := NewWithHTTPClient(httpClient, "", "a mock license key", &srv.URL, &srv.URL)

	bytes := []byte("foobar")
	err, successCount := client.SendTelemetry("arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes})

	assert.NoError(t, err)
	assert.Equal(t, 1, successCount)
	assert.Equal(t, 2, count)
}

func TestClientSendOutOfRetries(t *testing.T) {
	count := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count += 1
		time.Sleep(300 * time.Millisecond)
	}))

	defer srv.Close()

	httpClient := srv.Client()
	httpClient.Timeout = 200 * time.Millisecond
	client := NewWithHTTPClient(httpClient, "", "a mock license key", &srv.URL, &srv.URL)

	bytes := []byte("foobar")
	err, successCount := client.SendTelemetry("arn:aws:lambda:us-east-1:1234:function:newrelic-example-go", [][]byte{bytes})

	assert.NoError(t, err)
	assert.Equal(t, 0, successCount)
	assert.Equal(t, retries, count)
}
