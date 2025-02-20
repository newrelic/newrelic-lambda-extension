# Instrumented Node.js Lambda

This is a "Hello, World" style Lambda function in Node.js, instrumented 
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

## Deploy the sample application

The Serverless Application Model Command Line Interface (SAM CLI) is an extension of the AWS CLI that adds functionality for building and testing Lambda applications. It uses Docker to run your functions in an Amazon Linux environment that matches Lambda. It can also emulate your application's build environment and API.

To use the SAM CLI, you need the following tools.

* SAM CLI - [Install the SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
* Node.js - [Install Node.js 22](https://nodejs.org/en/), including the NPM package management tool.
* Docker - [Install Docker community edition](https://hub.docker.com/search/?type=edition&offering=community)

To build and deploy your application for the first time, run the following in your shell:

```bash
sam build
sam deploy --guided
```

The first command will build the source of your application. The second command will package and deploy your application to AWS, with a series of prompts:

* **Stack Name**: The name of the stack to deploy to CloudFormation. This should be unique to your account and region, and a good starting point would be something matching your project name.
* **AWS Region**: The AWS region you want to deploy your app to.
* **Confirm changes before deploy**: If set to yes, any change sets will be shown to you before execution for manual review. If set to no, the AWS SAM CLI will automatically deploy application changes.
* **Allow SAM CLI IAM role creation**: Many AWS SAM templates, including this example, create AWS IAM roles required for the AWS Lambda function(s) included to access AWS services. By default, these are scoped down to minimum required permissions. To deploy an AWS CloudFormation stack which creates or modifies IAM roles, the `CAPABILITY_IAM` value for `capabilities` must be provided. If permission isn't provided through this prompt, to deploy this example you must explicitly pass `--capabilities CAPABILITY_IAM` to the `sam deploy` command.
* **Save arguments to samconfig.toml**: If set to yes, your choices will be saved to a configuration file inside the project, so that in the future you can just re-run `sam deploy` without parameters to deploy changes to your application.


## Code Structure

Now is also a good time to look at the structure of the example code.

```bash
.
├── README.md                   <-- This instructions file
├── newrelic-example-node       <-- Source code for a lambda function
│   ├── app.js                  <-- Lambda function code
│   └── test                    <-- Unit tests
└── template.yaml
```


### template.yaml

This function is deployed using a SAM template, which is a CloudFormation
template with some extra syntactic sugar for Lambda functions. In it, we
tell CloudFormation where to find lambda function code, what layers to use, and
what IAM policies to add to the Lambda function's execution role. We also set
environment variables that are available to the handler function. 

### app.js

Lambda functions written in JavaScript are Node.js modules. The runtime loads them
just like any Node module, and then invokes the handler function for each 
invocation event. New Relic publishes a Lambda Layer that wraps your handler
function, and initializes the New Relic agent, allowing us to collect telemetry.

There are a couple examples here of how you might add custom events and attributes
to the default telemetry.

Since JavaScript is a dynamic, interpreted language, the Agent can inject instrumentation
into the various client libraries you might be using in your function. This happens 
once, during cold start, and provides rich, detailed instrumentation out of the box, 
with minimal developer effort.

## Unit tests

Tests are defined in the `hello-world/tests` folder in this project. Use NPM to install the [Mocha test framework](https://mochajs.org/) and run unit tests.

```bash
newrelic-example-node$ cd hello-world
hello-world$ npm install
hello-world$ npm run test
```

## Cleanup

To delete the sample application that you created, use the AWS CLI. Assuming you used your project name for the stack name, you can run the following:

```bash
sam delete --stack-name 
```

## Resources

See the [AWS SAM developer guide](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/what-is-sam.html) for an introduction to SAM specification, the SAM CLI, and serverless application concepts.

Next, you can use AWS Serverless Application Repository to deploy ready to use Apps that go beyond hello world samples and learn how authors developed their applications: [AWS Serverless Application Repository main page](https://aws.amazon.com/serverless/serverlessrepo/)

