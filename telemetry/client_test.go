package telemetry

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

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

	client := NewWithHTTPClient(srv.Client(), "", "a mock license key", &srv.URL)

	bytes := []byte("foobar")
	err := client.SendTelemetry("fakeArn", [][]byte{bytes})

	assert.NoError(t, err)
}
