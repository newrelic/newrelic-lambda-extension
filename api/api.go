// Package api contains types and constants for interacting with the AWS Lambda Extension API.
package api

// LifecycleEvent represents lifecycle events that the extension can express interest in
type LifecycleEvent string

const (
	Invoke   LifecycleEvent = "INVOKE"
	Shutdown LifecycleEvent = "SHUTDOWN"

	Version = "2020-01-01"

	LambdaHostPortEnvVar = "AWS_LAMBDA_RUNTIME_API"

	ExtensionNameHeader = "Lambda-Extension-Name"
	ExtensionIdHeader   = "Lambda-Extension-Identifier"
)

type InvocationEvent struct {
	// Either INVOKE or SHUTDOWN.
	EventType LifecycleEvent `json:"eventType"`
	// The time at which the event will timeout, as milliseconds since the epoch.
	DeadlineMs int64 `json:"deadlineMs"`
	// The AWS Request ID, for INVOKE events.
	RequestId string `json:"requestId"`
	// The ARN of the function being invoked, for INVOKE events.
	InvokedFunctionArn string `json:"invokedFunctionArn"`
	// XRay trace ID, for INVOKE events.
	Tracing map[string]string `json:"tracing"`
}

type RegistrationRequest struct {
	Events            []LifecycleEvent `json:"events"`
	ConfigurationKeys []string         `json:"configurationKeys"`
}

type RegistrationResponse struct {
	FunctionName    string            `json:"functionName"`
	FunctionVersion string            `json:"functionVersion"`
	Handler         string            `json:"handler"`
	Configuration   map[string]string `json:"configuration"`
}
