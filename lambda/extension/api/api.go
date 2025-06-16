// Package api contains types and constants for interacting with the AWS Lambda Extension API.
package api

import (
	"fmt"
	"time"
)

// LifecycleEvent represents lifecycle events that the extension can express interest in
type LifecycleEvent string
type ShutdownReason string
type LogEventType string

const (
	Invoke   LifecycleEvent = "INVOKE"
	Shutdown LifecycleEvent = "SHUTDOWN"

	Spindown ShutdownReason = "spindown"
	Timeout  ShutdownReason = "timeout"
	Failure  ShutdownReason = "failure"

	Platform  LogEventType = "platform"
	Function               = "function"
	Extension              = "extension"

	Version        string = "2020-01-01"
	LogsApiVersion        = "2020-08-15"

	LambdaHostPortEnvVar = "AWS_LAMBDA_RUNTIME_API"

	ExtensionNameHeader      = "Lambda-Extension-Name"
	ExtensionFeatureHeader   = "Lambda-Extension-Accept-Feature"
	ExtensionIdHeader        = "Lambda-Extension-Identifier"
	ExtensionErrorTypeHeader = "Lambda-Extension-Function-Error-Type"

	LogBufferDefaultBytes   uint32 = 1024 * 1024
	LogBufferDefaultItems   uint32 = 10_000
	LogBufferDefaultTimeout uint32 = 500
)

type InvocationEvent struct {
	// Either INVOKE or SHUTDOWN.
	EventType LifecycleEvent `json:"eventType"`
	// The instant that the invocation times out, as epoch milliseconds
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
	AccountId 		string `json:"accountId"`
}

type LogSubscription struct {
	Buffering   BufferingCfg   `json:"buffering"`
	Destination DestinationCfg `json:"destination"`
	Types       []LogEventType `json:"types"`
}

func NewLogSubscription(bufferingCfg BufferingCfg, destinationCfg DestinationCfg, types []LogEventType) *LogSubscription {
	return &LogSubscription{
		Buffering:   bufferingCfg,
		Destination: destinationCfg,
		Types:       types,
	}
}

func DefaultLogSubscription(types []LogEventType, port uint16) *LogSubscription {
	endpoint := formatLogsEndpoint(port)

	return NewLogSubscription(
		BufferingCfg{
			MaxBytes:  LogBufferDefaultBytes,
			MaxItems:  LogBufferDefaultItems,
			TimeoutMs: LogBufferDefaultTimeout,
		},
		DestinationCfg{
			URI:      endpoint,
			Protocol: "HTTP",
		},
		types,
	)
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

type LogEvent struct {
	Time   time.Time   `json:"time"`
	Type   string      `json:"type"`
	Record interface{} `json:"record"`
}
