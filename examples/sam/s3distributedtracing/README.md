# Distributed Tracing with S3 Example

Here, we demonstrate trace context propagation for Distributed Tracing using S3 object metadata.

- We start with a static HTML page, served by a Python Lambda function, and instrumented using the New Relic Browser Agent.
- In response to a form submission (POST), the Python Lambda splits the message into words and uploads each as an S3 object, propagating the trace context in the S3 object metadata.
- A Node.js Lambda function is triggered by S3 object creation events. It retrieves the Trace Context from the S3 object metadata and logs the content.

## Building and deploying

### Prerequisites

- The [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html)
- [Docker](https://docs.docker.com/get-docker/)
- [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
- [newrelic-lambda](https://github.com/newrelic/newrelic-lambda-cli#installation) CLI tool

From a command prompt, in this directory, run:

    ./deploy.sh <accountId> <trustedAccountId> <browserAppId> <browserAgentId> <browserLicenseKey> <newRelicLicenseKey> <region>

## Verifying Distributed Tracing

To verify that distributed tracing is working correctly and that the **Trace ID** remains identical throughout the chain, you can check the CloudWatch logs for both Lambda functions.

1. **Invoke the application**: Open the `PythonS3WriterApi` URL (found in the SAM stack outputs) in your browser, enter a message, and click **Submit**.
2. **Check Python logs**: In the CloudWatch logs for the `PythonS3Writer` function, look for a log entry like:
    `New Relic Trace Details - Trace ID: <trace_id>, Span ID: <span_id>`
3. **Check Node.js logs**: In the CloudWatch logs for the `NodeS3Reader` function, look for the corresponding log entry:
    `New Relic Trace Details - Trace ID: <trace_id>, Span ID: <span_id>`
4. **Compare Trace IDs**: Confirm that the `<trace_id>` is identical in both log entries. This confirms that the trace context was successfully propagated from the browser to the Python writer, and then to the Node.js reader via S3 metadata.

## Key Differences from SQS Example

This example demonstrates the same distributed tracing pattern as the SQS-based example but uses S3:

- **Storage Medium**: Uses S3 object metadata instead of SQS message attributes to store trace context.
- **Metadata Limits**: S3 object metadata has a limit of **2 KB** for user-defined metadata. While this is plenty for New Relic distributed tracing headers, it is significantly smaller than the **256 KB** allowed for SNS or SQS message attributes.
- **Event Triggering**: S3 bucket notifications trigger the Node.js function.
- **Metadata Retrieval**: Node.js function uses `s3.getObject()` to retrieve both content and metadata.
