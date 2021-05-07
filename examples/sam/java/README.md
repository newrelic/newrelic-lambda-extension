# Instrumented Java Lambda

This is a "Hello, World" style Lambda function in Java 8, instrumented 
with the New Relic OpenTracing Java SDK.

This example is both instructive, and a diagnostic tool: if you can
deploy this Lambda function, and see its events in NR One, you'll
know that all the telemetry plumbing is connected correctly. 

## Building and deploying

### Prerequisites

- The [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html)
- [Docker](https://docs.docker.com/get-docker/)
- The [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
- [newrelic-lambda](https://github.com/newrelic/newrelic-lambda-cli#installation) CLI tool

Make sure you've run the `newrelic-lambda install` command in your
AWS Region, and included the `--enable-license-key-secret` flag.

### deploy script

From a command prompt, in this directory, run

    ./deploy.sh <accountId> <region>
    
where `<accountId>` is your New Relic account ID, and  `<region>` 
is your AWS Region, like "us-west-2".

This will package and deploy the CloudFormation stack for this example 
function.

At this point, you can invoke the function. As provided, the example
function doesn't pay attention to its invocation event. If everything
has gone well, each invocation gets reported to New Relic, and its
telemetry appears in NR One.

## Code Structure

Now is also a good time to look at the structure of the example code.

### template.yaml

This function is deployed using a SAM template, which is a CloudFormation
template with some extra syntactic sugar for Lambda functions. In it, we
tell CloudFormation where to find lambda function code, what layers to use, and
what IAM policies to add to the Lambda function's execution role. We also set
environment variables that are available to the handler function. 

### App.java and Stream.java

Lambda functions written in Java are Java classes. The runtime loads them
just like any Java class, and then invokes the handler function for each 
invocation event.

New Relic's Java instrumentation is based on the OpenTracing API standard. 
Libraries that implement the standard create spans using the OpenTracing API.
New Relic gathers these traces, and acts as an "exporter", serializing them
and sending them to the New Relic collector.

If you're familiar with the logging ecosystem in Java, the idea behind 
OpenTracing and OpenTelemetry is a similar one: standardize the API that 
libraries use, and let applications choose the concrete implementation 
responsible for managing the data itself. 

We have two examples, implementing the AWS RequestHandler and the AWS RequestStreamHandler.
Both examples are instrumented with the published New Relic Lambda Layer that wraps your handler
function, and initializes the New Relic agent, allowing us to collect telemetry. 

Additionally, there are a couple examples in App.java demonstrating how you might use the OpenTracing API in
your own code to create custom span data.


