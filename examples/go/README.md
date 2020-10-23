# Instrumented Go Lambda

This is a "Hello, World" style Lambda function in Go, instrumented 
with the New Relic Agent.

This example is both instructive, and a diagnostic tool: if you can
deploy this Lambda function, and see its events in NR One, you'll
know that all the telemetry plumbing is connected correctly. 

## Building and deploying

### Prerequisites

- The [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html)
- [Go](https://golang.org/doc/install)
- [newrelic-lambda](https://github.com/newrelic/newrelic-lambda-cli#installation) CLI tool

Make sure you've run the `newrelic-lambda install` command in your
AWS Region, and included the `--enable-license-key-secret` flag.

### deploy script

From a command prompt, in this directory, run

    ./deploy.sh <accountId> <region>
    
where `<accountId>` is your New Relic account ID and `<region>` is your AWS Region, like "us-west-2".

This will compile, package and deploy the CloudFormation stack for
this example function.

At this point, you can invoke the function. As provided, the example
function doesn't pay attention to its invocation event. If everything
has gone well, each invocation gets reported to New Relic, and its
telemetry appears in NR One.

## Code Structure

Now is also a good time to look at the structure of the example code.

### template.yaml

This function is deployed using a SAM template, which is a CloudFormation
template with some extra syntactic sugar for Lambda functions. In it, we
tell CloudFormation where to find the handler zip, what layers to use, and
what IAM policies to add to the Lambda function's execution role. 

### main.go

Lambda functions written in Go are whole executable processes. They start at
`main()` just like any Go program. The included Lambda libraries do the work
of interfacing with the AWS Lambda service, to fetch new events and send 
responses. The New Relic Agent wraps the Lambda library, and collects telemetry.

There are a couple examples here of how you might add custom events and attributes
to the default telemetry.

Since Go is a compiled language, the Agent can't automatically modify the various 
database and network clients your function might be using, so you'll want to
manually wrap interesting parts of your function code in 
[segments](https://docs.newrelic.com/docs/agents/go-agent/instrumentation/instrument-go-segments).
