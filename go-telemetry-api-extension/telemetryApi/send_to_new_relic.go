package telemetryApi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"net/http"
)

func sendDataToNR(ctx context.Context, logEntries []interface{}, d *Dispatcher) error {

	/*
		logAttributes := map[string]string{}
		logAttributes[HostnameAttributeKey] = "a host name"
		logAttributes["mykey"] = "my value"

		payload := NewLogPayload(logAttributes)

		for _, logLine := range logEntries{
			//  do some processing and add line to payload
			payload.AddLogLine(time.Now().UnixMilli(), "debug", "message")
		}

		bodyBytes := payload.Marshal()
	*/

	bodyBytes, _ := json.Marshal(map[string]string{"message": fmt.Sprintf("%v", logEntries)})
	req, err := http.NewRequestWithContext(ctx, "POST", d.postUri, bytes.NewBuffer(bodyBytes))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", d.licenseKey)
	_, err = d.httpClient.Do(req)

	return err
}
