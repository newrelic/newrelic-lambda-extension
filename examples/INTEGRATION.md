# Enable serverless monitoring for AWS Lambda

[Serverless monitoring for AWS Lambda](https://docs.newrelic.com/docs/introduction-new-relic-monitoring-aws-lambda)
offers in-depth performance monitoring for your Lambda functions, with very low latency. This document
explains how to enable this feature and get started using it.

**Note** Use of this feature may result in Amazon Web Services charges. For more information, see 
[Requirements](https://docs.newrelic.com/docs/introduction-new-relic-monitoring-aws-lambda#requirements).

## How Lambda monitoring works

TODO: revise diagram

When our Lambda monitoring is enabled, this is how data moves from your Lambda function to New Relic:

1. You configure your Lambda function to include our Lambda Layer for the runtime you've chosen.
2. As your code runs, the New Relic Agent or SDK gathers telemetry about the invocation and its execution.
3. Just before execution finishes, the New Relic Agent or SDK sends the telemetry it has gathered to the New Relic
 Lambda Extension, included with the Layer. 
4. The extension sends the telemetry on to the New Relic collector, along with additional information from the Lambda
 platform.

## Requirements

For requirements and compatibility information, including **potential impact on your AWS billing**, see [Lambda
monitoring requirements](https://docs.newrelic.com/docs/serverless-function-monitoring/aws-lambda-monitoring/get-started/introduction-new-relic-monitoring-aws-lambda#requirements).

## Enable procedure overview

There are a few things that have to happen to let New Relic gather telemetry from your Lambda functions.

1. Link your AWS account with your New Relic account.
2. Store your New Relic license key in AWS Secret Manager.
3. Instrument your individual Lambda functions.

There are several ways to accomplish each of these steps, and we know your needs aren't the same as someone else's. 
This guide will focus primarily on the most straightforward path to success, with details on how all the parts fit
together. 
 
## Prerequisites
 
1. Be sure you have the [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html) installed
 and configured.

    `aws sts get-caller-identity` will print your user, role, and AWS account ID if the AWS CLI is installed and
     configured correctly. 

2. You'll need sufficient access to your AWS and New Relic accounts to perform the integration.
    - In New Relic, You must be a user or [admin](https://docs.newrelic.com/docs/accounts/accounts/roles-permissions/users-roles#roles) 
    with an infrastructure manager [Add-on role](https://docs.newrelic.com/docs/accounts/accounts/roles-permissions/add-roles-permissions).
    - In AWS, you'll need permissions for creating IAM resources (Role and Policy), creating managed secrets, and
     creating and updating Lambda functions. These resources are created via CloudFormation stacks, so you'll need 
     permissions to create those. Also, all the examples require that you create an S3 bucket, so you'll need those
     permissions as well.
2. Ensure that you have Python 3.3 (or later) installed.
3. Also install the [newrelic-lambda](https://github.com/newrelic/newrelic-lambda-cli#installation) CLI tool.  This
 tools automates the account linking and license key storage steps, and can help instrument your existing lambda
 functions as well.
 
       pip install newrelic-lambda-cli
 
    Note that you may need to use `pip3` instead of `pip` if your system uses Python 2.x by default. This is common on
    MacOS.

## Linking accounts and instrumenting your first Lambda function

1. Gather three values:
    1. Your New Relic account ID
    2. Your Personal [New Relic API Key](https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys#user-api-key) 
    (not your REST API key)
    3. Decide on a name for the link between your New Relic and AWS accounts. This name will appear in the
     Infrastructure UI, where you can manage the link in the future.
2. Using those values in the appropriate places, run this command:

       newrelic-lambda integrations install --nr-account-id YOUR_NR_ACCOUNT_ID \
           --linked-account-name YOUR_LINKED_ACCOUNT_NAME \
           --nr-api-key YOUR_NR_API_KEY \
           --enable-license-key-secret
3. Clone this repository

       git clone https://github.com/newrelic/newrelic-lambda-extension.git

4. Select an example to deploy, instrument, and run:
   - [Node.js](https://github.com/newrelic/newrelic-lambda-extension/tree/main/examples/node)
   - [Python](https://github.com/newrelic/newrelic-lambda-extension/tree/main/examples/python)
   - [Go](https://github.com/newrelic/newrelic-lambda-extension/tree/main/examples/go)
   - [Java](https://github.com/newrelic/newrelic-lambda-extension/tree/main/examples/java)
   - [.NET](https://github.com/newrelic/newrelic-lambda-extension/tree/main/examples/dotnet)
 
   You will need to follow the directions in the README for your chosen example carefully. It's also a good idea to
   read through the example code, and the SAM template, to better understand how each language integrates with New
   Relic.
5. Find your new Lambda function in New Relic, and verify that it is instrumented, and sending telemetry.

Our examples are based on the AWS SAM CLI. There are many other tools available for managing and deploying Lambda
functions. New Relic offers a plugin for the Serverless Framework, and the CLI can modify your existing Lambda
functions to add instrumentation. 

Integrating the necessary Lambda Layer and function permission is straightforward in whatever AWS resource management
tool you choose. 

After you've gotten the example to work for you, you can clean up by deleting the CloudFormation stack, using either
the AWS Console, or the AWS CLI: `aws cloudformation delete-stack --stack-name <stack-name>` 

### What does the account link do?

When you link your AWS account to New Relic, you're granting permission to us to create an inventory of your AWS
account, and gather CloudWatch metrics for your Lambda functions. Resources in your AWS account then show up as
entities in the New Relic Entity Explorer. We also decorate the telemetry from the Agent with configuration information
gathered via this link. This is the "basic" Lambda monitoring that our advanced monitoring builds upon.

### Why store the License Key in Secrets Manager?

Your New Relic License Key identifies and authenticates you to New Relic, allowing us to associate your telemetry with
your New Relic account. Each function that sends telemetry needs access to this value, and it needs to be managed 
securely. Secrets Manager solves these problems.

### What's in the New Relic Lambda Layers?

First, the layer for your runtime contains the New Relic Lambda Extension. This executable acts as a "sidecar" for your
Lambda function. The extension sends the telemetry to New Relic, and interacts with the Lambda platform directly, to 
enhance the data we gather, while minimizing the impact of instrumentation on your application's performance.

Second, for Node.js and Python, the layer contains the New Relic Agent code, and a "wrapper" for your Lambda handler.

For other runtimes, we take an SDK approach, providing you with the tools to instrument your code, while taking
advantage of emerging standards like OpenTracing and OpenTelemetry to gather telemetry your libraries and frameworks
are already producing.

## Frequent Complications

### Multiple AWS regions and accounts

The `newrelic-lambda` CLI should be run once per region, with the `--aws-region` parameter. Use the same linked
account name, and the tool will detect that the account link has been created already. The license key secret needs
to be created in each region.

Similarly, several AWS accounts can be linked to a New Relic account. Give each account a different linked account
name. The `--aws-profile` argument to the CLI tool will select the named profile. The tool uses the same configuration
as the AWS CLI.

### Logs in Context

You can [stream CloudWatch logs](TODO) (for Lambda functions, or any other AWS services) to New Relic.

 

## For more help

If you need more help, check out these support and learning resources:

- Browse the Explorers Hub  to get help from the community and join in discussions.
- Find answers on our sites and learn how to use our support portal.
- Run New Relic Diagnostics, our troubleshooting tool for Linux, Windows, and macOS.
- Review New Relic's data security and licenses documentation.
