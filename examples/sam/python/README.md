# Instrumented Python Lambda

This is a "Hello, World" style Lambda function in Python, instrumented 
with the New Relic Agent.

This example is both instructive, and a diagnostic tool: if you can
deploy this Lambda function, and see its events in NR One, you'll
know that all the telemetry plumbing is connected correctly. 

## Building and deploying

### Prerequisites

- The [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html)
- [Docker](https://docs.docker.com/get-docker/)
- The [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
- [newrelic-lambda](https://github.com/newrelic/newrelic-lambda-cli#installation) CLI tool

Make sure you've run the `newrelic-lambda integrations install` command in your
AWS Region, and included the `--enable-license-key-secret` flag.

To build and deploy your application for the first time, run the following in your shell:

```bash
sam build --use-container
sam deploy --guided
```

The first command will build the source of your application. The second command will package and deploy your application to AWS, with a series of prompts:

* **Stack Name**: The name of the stack to deploy to CloudFormation. This should be unique to your account and region, and a good starting point would be something matching your project name.
* **AWS Region**: The AWS region you want to deploy your app to.
* **Confirm changes before deploy**: If set to yes, any change sets will be shown to you before execution for manual review. If set to no, the AWS SAM CLI will automatically deploy application changes.
* **Allow SAM CLI IAM role creation**: Many AWS SAM templates, including this example, create AWS IAM roles required for the AWS Lambda function(s) included to access AWS services. By default, these are scoped down to minimum required permissions. To deploy an AWS CloudFormation stack which creates or modifies IAM roles, the `CAPABILITY_IAM` value for `capabilities` must be provided. If permission isn't provided through this prompt, to deploy this example you must explicitly pass `--capabilities CAPABILITY_IAM` to the `sam deploy` command.
* **Save arguments to samconfig.toml**: If set to yes, your choices will be saved to a configuration file inside the project, so that in the future you can just re-run `sam deploy` without parameters to deploy changes to your application.

At this point, you can invoke the function. As provided, the example
function doesn't pay attention to its invocation event. If everything
has gone well, each invocation gets reported to New Relic, and its
telemetry appears in NR One.

## Code Structure

Now is also a good time to look at the structure of the example code.


```bash
.
├── README.md                   <-- This instructions file
├── newrelic_example_python     <-- Source code for a lambda function
│   ├── app.py                  <-- Lambda function code
├── tests                       <-- Unit tests
└── template.yaml
```

### template.yaml

This function is deployed using a SAM template, which is a CloudFormation
template with some extra syntactic sugar for Lambda functions. In it, we
tell CloudFormation where to find lambda function code, what layers to use, and
what IAM policies to add to the Lambda function's execution role. We also set
environment variables that are available to the handler function. 

### app.py

Lambda functions written in Python are Python modules. The runtime loads them
just like any python module, and then invokes the handler function for each 
invocation event. New Relic publishes a Lambda Layer that wraps your handler
function, and initializes the New Relic agent, allowing us to collect telemetry.

There are a couple examples here of how you might add custom events and attributes
to the default telemetry.

Since Python is a dynamic, interpreted language, the Agent can inject instrumentation
into the various client libraries you might be using in your function. This happens 
once, during cold start, and provides rich, detailed instrumentation out of the box, 
with minimal developer effort.


## Cleanup

To delete the sample application that you created, use the AWS CLI. Assuming you used your project name for the stack name, you can run the following:

```bash
sam delete --stack-name "newrelic_example_python"
```