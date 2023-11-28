package telemetry

import (
	"io"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/newrelic/newrelic-lambda-extension/util"
)

const (
	telemetryNamedPipePath       = "/tmp/newrelic-telemetry"
	telemetryNamedPipeRetries    = 10
	telemetryNamedPipeRetryDelay = 10 * time.Millisecond
)

func InitTelemetryChannel() (chan []byte, error) {
	_ = os.Remove(telemetryNamedPipePath)

	err := syscall.Mkfifo(telemetryNamedPipePath, 0666)
	if err != nil {
		return nil, err
	}

	// verify that the special file is visible in the file system
	// before we try to open it, to avoid a race condition
	var tries int
	for {
		_, err := os.Stat(telemetryNamedPipePath)
		if err == nil {
			break
		}
		if tries < telemetryNamedPipeRetries {
			tries++
			time.Sleep(telemetryNamedPipeRetryDelay)
		} else {
			log.Panic("failed to create telemetry pipe ", err)
		}
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
	bytes, err := io.ReadAll(telemetryPipe)
	if err != nil {
		log.Panic("failed to read telemetry pipe", err)
	}

	return bytes
}
