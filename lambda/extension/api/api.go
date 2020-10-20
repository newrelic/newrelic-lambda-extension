// Package api contains types and constants for interacting with the AWS Lambda Extension API.
package api

import "fmt"

// LifecycleEvent represents lifecycle events that the extension can express interest in
type LifecycleEvent string

const (
	Invoke   LifecycleEvent = "INVOKE"
	Shutdown LifecycleEvent = "SHUTDOWN"

	Version = "2020-01-01"
	LogsApiVersion = "2020-08-15"

	LambdaHostPortEnvVar = "AWS_LAMBDA_RUNTIME_API"

	ExtensionNameHeader = "Lambda-Extension-Name"
	ExtensionIdHeader   = "Lambda-Extension-Identifier"
	ExtensionErrorTypeHeader   = "Lambda-Extension-Function-Error-Type"
)

type InvocationEvent struct {
	// Either INVOKE or SHUTDOWN.
	EventType LifecycleEvent `json:"eventType"`
	// The time at which the event will timeout, as milliseconds since the epoch.
	DeadlineMs int64 `json:"deadlineMs"`
	// The AWS Request ID, for INVOKE events.
	RequestID string `json:"requestId"`
	// The ARN of the function being invoked, for INVOKE events.
	InvokedFunctionARN string `json:"invokedFunctionArn"`
	// XRay trace ID, for INVOKE events.
	Tracing map[string]string `json:"tracing"`
}

type RegistrationRequest struct {
	Events            []LifecycleEvent `json:"events"`
}

type RegistrationResponse struct {
	FunctionName    string            `json:"functionName"`
	FunctionVersion string            `json:"functionVersion"`
	Handler         string            `json:"handler"`
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

func DefaultLogSubscription(types []LogEventType, endpoint string) LogSubscription {
	return LogSubscription{
		Buffering: BufferingCfg{
			MaxBytes:  1024 * 1024,
			MaxItems:  10000,
			TimeoutMs: 10000,
		},
		Destination: DestinationCfg{
			URI:      endpoint,
			Protocol: "HTTP",
			Encoding: "JSON",
			Method:   "POST",
		},
		Types: types,
	}
}

func FormatLogsEndpoint(port uint16, path string) string {
	return fmt.Sprintf("http://sandbox:%d/%s", port, path)
}

type BufferingCfg struct {
	MaxBytes  uint32 `json:"maxBytes"`
	MaxItems  uint32 `json:"maxItems"`
	TimeoutMs uint32 `json:"timeoutMs"`
}

type DestinationCfg struct {
	URI      string `json:"URI"`
	Protocol string `json:"protocol"`
	Encoding string `json:"encoding"`
	Method   string `json:"method"`
	//Port uint16 `json:"port"` //Not used by us
}

type LogEventType string

const (
	Platform  LogEventType = "platform"
	Function               = "function"
	Extension              = "extension"
)
