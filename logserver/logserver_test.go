package logserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"testing"
	"time"
)

func Test_Logserver(t *testing.T) {
	logs, err := Start()
	if err != nil {
		log.Println("Failed to start logs HTTP server", err)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	testEvents := []api.LogEvent{
		{
			Time: time.Now(),
			Type: "platform.extension",
			Record: map[string]string{
				"foo": "bar",
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
}
