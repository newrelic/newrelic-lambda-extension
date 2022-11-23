package telemetryApi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
        "errors"
	"net/http"

//	"github.com/pkg/errors"
)

func sendDataToNR(ctx context.Context, logEntries []interface{}, d *Dispatcher) (error) {

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
