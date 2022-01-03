module github.com/newrelic/newrelic-lambda-extension

go 1.14

require (
	github.com/aws/aws-lambda-go v1.19.1 // indirect
	github.com/aws/aws-sdk-go v1.34.21
	github.com/google/uuid v1.1.2
	github.com/newrelic/go-agent/v3 v3.9.0
	github.com/newrelic/go-agent/v3/integrations/nrlambda v1.2.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v1.3.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.3.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.3.0
	go.opentelemetry.io/otel/sdk v1.3.0
	go.opentelemetry.io/otel/trace v1.3.0
	golang.org/x/mod v0.4.2
	google.golang.org/genproto v0.0.0-20200910191746-8ad3c7ee2cd1 // indirect
)
