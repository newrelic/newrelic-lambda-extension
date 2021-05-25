package logserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/stretchr/testify/assert"
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
