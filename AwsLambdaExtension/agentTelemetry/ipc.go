package agentTelemetry

import (
	"io"
	"os"
	"syscall"
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
		l.Panic("failed to open telemetry pipe", err)
	}

	defer func(telemetryPipe *os.File) {
		err := telemetryPipe.Close()
		if err != nil {
			l.Warn(err)
		}
	}(telemetryPipe)

	// When the write side closes, we get an EOF.
	bytes, err := io.ReadAll(telemetryPipe)
	if err != nil {
		l.Panic("failed to read telemetry pipe", err)
	}

	return bytes
}
