# Instrumented Ruby Lambda

This is a "Hello, World" style Lambda function in Ruby, instrumented 
with the New Relic agent.

This example is both instructive and a diagnostic tool: if you can
deploy this Lambda function and see its events in NR One, you'll
know that all the telemetry plumbing is connected correctly. 

## Building and deploying

### Prerequisites

- The [AWS CLI v2](https://aws.amazon.com/cli/)
- [Docker](https://docs.docker.com/get-docker/)
- The [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)

Make sure you've run the `newrelic-lambda integrations install` command in your
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

### main.tf

This function is deployed using this Terraform script. In it, we
tell Terraform where to find lambda function code, what layers to use, and
what IAM policies to add to the Lambda function's execution role. We also set
environment variables that are available to the handler function. 

### app.rb

Lambda functions written in Ruby are involve a Ruby method at a mininum and can
optionally be found within a class and/or module based namespace. The runtime
loads the Ruby code, and then invokes the handler function method for each 
invocation event. New Relic publishes a Lambda Layer that wraps your handler
function and initializes the New Relic agent, allowing us to collect telemetry.

There are a couple examples here of how you might add custom events and attributes
to the default telemetry.

Since Ruby is a dynamic, interpreted language, the Agent can inject instrumentation
into the various client libraries you might be using in your function. This happens 
once, during cold start, and provides rich, detailed instrumentation out of the box, 
with minimal developer effort.

