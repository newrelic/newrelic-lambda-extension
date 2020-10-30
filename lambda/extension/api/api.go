// Package api contains types and constants for interacting with the AWS Lambda Extension API.
package api

import (
	"fmt"
	"time"
)

// LifecycleEvent represents lifecycle events that the extension can express interest in
type LifecycleEvent string
type ShutdownReason string

const (
	Invoke   LifecycleEvent = "INVOKE"
	Shutdown LifecycleEvent = "SHUTDOWN"

	Spindown ShutdownReason = "spindown"
	Timeout ShutdownReason = "timeout"
	Failure ShutdownReason = "failure"

	Version        = "2020-01-01"
	LogsApiVersion = "2020-08-15"

	LambdaHostPortEnvVar = "AWS_LAMBDA_RUNTIME_API"

	ExtensionNameHeader      = "Lambda-Extension-Name"
	ExtensionIdHeader        = "Lambda-Extension-Identifier"
	ExtensionErrorTypeHeader = "Lambda-Extension-Function-Error-Type"
)

type InvocationEvent struct {
	// Either INVOKE or SHUTDOWN.
	EventType LifecycleEvent `json:"eventType"`
	// The time left on the invocation, in microseconds.
	DeadlineMs int64 `json:"deadlineMs"`
	// The AWS Request ID, for INVOKE events.
	RequestID string `json:"requestId"`
	// The ARN of the function being invoked, for INVOKE events.
	InvokedFunctionARN string `json:"invokedFunctionArn"`
	// XRay trace ID, for INVOKE events.
	Tracing map[string]string `json:"tracing"`
	// The reason for termination, if this is a shutdown event
	ShutdownReason ShutdownReason `json:"shutdownReason"`
}

type RegistrationRequest struct {
	Events []LifecycleEvent `json:"events"`
}

type RegistrationResponse struct {
	FunctionName    string `json:"functionName"`
	FunctionVersion string `json:"functionVersion"`
	Handler         string `json:"handler"`
}

type LogSubscription struct {
	Buffering   BufferingCfg   `json:"buffering"`
	Destination DestinationCfg `json:"destination"`
	Types       []LogEventType `json:"types"`
}

func NewLogSubscription(bufferingCfg BufferingCfg, destinationCfg DestinationCfg, types []LogEventType) LogSubscription {
	return LogSubscription{
		Buffering:   bufferingCfg,
		Destination: destinationCfg,
		Types:       types,
	}
}

func DefaultLogSubscription(types []LogEventType, port uint16) LogSubscription {
	endpoint := formatLogsEndpoint(port)

	return LogSubscription{
		Buffering: BufferingCfg{
			MaxBytes:  256 * 1024,
			MaxItems:  1000,
			TimeoutMs: 100,
		},
		Destination: DestinationCfg{
			URI:      endpoint,
			Protocol: "HTTP",
		},
		Types: types,
	}
}

func formatLogsEndpoint(port uint16) string {
	return fmt.Sprintf("http://sandbox:%d", port)
}

type BufferingCfg struct {
	MaxBytes  uint32 `json:"maxBytes"`
	MaxItems  uint32 `json:"maxItems"`
	TimeoutMs uint32 `json:"timeoutMs"`
}

type DestinationCfg struct {
	URI      string `json:"URI"`
	Protocol string `json:"protocol"`
	//Port uint16 `json:"port"` //Not used by us
}

type LogEventType string

const (
	Platform  LogEventType = "platform"
	Function               = "function"
	Extension              = "extension"
)

type LogEvent struct {
	Time   time.Time   `json:"time"`
	Type   string      `json:"type"`
	Record interface{} `json:"record"`
}
