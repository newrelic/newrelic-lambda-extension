package api

const (
	Invoke              = "INVOKE"
	Shutdown            = "SHUTDOWN"
	Version             = "2020-01-01"
	AwsLambdaRuntimeApi = "AWS_LAMBDA_RUNTIME_API"
)

type InvocationEvent struct {
	EventType          string            `json:"eventType"`
	DeadlineMs         int               `json:"deadlineMs"`
	RequestId          string            `json:"requestId"`
	InvokedFunctionArn string            `json:"invokedFunctionArn"`
	Tracing            map[string]string `json:"tracing"`
}

type RegistrationRequest struct {
	Events            []string `json:"events"`
	ConfigurationKeys []string `json:"configurationKeys"`
}
