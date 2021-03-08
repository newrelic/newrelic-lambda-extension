package telemetry

import (
	"io/ioutil"
	"log"
	"os"
	"syscall"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

const telemetryNamedPipePath = "/tmp/newrelic-telemetry"

func InitTelemetryChannel() (chan []byte, error) {
	_ = os.Remove(telemetryNamedPipePath)

	err := syscall.Mkfifo(telemetryNamedPipePath, 0666)
	if err != nil {
		return nil, err
	}

	telemetryChan := make(chan []byte)

	go func() {
		for {
			telemetryChan <- pollForTelemetry()
		}
	}()

	return telemetryChan, nil
}

func pollForTelemetry() []byte {
	// Opening a pipe will block, until the write side has been opened as well
	telemetryPipe, err := os.OpenFile(telemetryNamedPipePath, os.O_RDONLY, 0)
	if err != nil {
		log.Panic("failed to open telemetry pipe", err)
	}

	defer util.Close(telemetryPipe)

	// When the write side closes, we get an EOF.
	bytes, err := ioutil.ReadAll(telemetryPipe)
	if err != nil {
		log.Panic("failed to read telemetry pipe", err)
	}

	return bytes
}
