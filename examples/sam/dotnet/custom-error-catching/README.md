# Instrumented Dotnet Lambda

This is a "Hello, World" style Lambda function in .NET, instrumented 
with the New Relic .NET Agent AWS Lambda layer.

This example is both instructive, and a diagnostic tool: if you can
deploy this Lambda function, and see its events in NR One, you'll
know that all the telemetry plumbing is connected correctly. 

# Custom Error Monitoring With NewRelic.Agent.Api

Lambdas have several conditions where they can never go through an invocation error and should always be handled cleanly.
This occurs with integrations into ALBs, API Gateways, batch-working SQS Queues(s), etc...

ALB/API Gateway expects a response body containing a 500 code, notifying the user of an error.

The Lambda should never completely error out when batch-working multiple messages from single or multiple queues. Instead it should return which items it successfully worked. 

 The SQS auto-redrives anything not included in the response. If the Lambda errors out, SQS retrieves all messages regardless of how many were completed.

These are a few examples where Lambda invocation errors will result in degraded performance and errors in event flows. 

All invocation errors cause a cold start.

### Lambdas Execution Environment

Lambda's has a very unique life cycle that is exampled in the following [documentation](https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtime-environment.html)

TLDR:
* INIT Phase:
A cold start is when a lambda performs the init process of the life cycle, where it will initialize Extension, the RunTime, including the Java process that invokes your function, and finalize initializing the user's code.

* Invoke Phase:
The lambda runtime invokes the user's code until the Shutdown Phase. 
The Lambda is considered `hot` and can respond quickly. The INIT Phase does not happen again unless there is a failed invocation.

* Shutdown Phase:
Reaping of the instance will occur when no invocations occur for 15 minutes. The creation of new instance to meet scaling demands will result in a cold start. The shutdown phase may not happen if the autoscaling and provision settings do not allow it.

* Invoke With Error:
When invocation errors occur, the Lambda virtual environment resets the runtime and shuts down the extensions. The following invocation forces a Coldish Start with a new INIT Phase in an already provisioned instance.

#### How Do Errors Affect The Life Cycle

When errors occur during a Lambda's invocation, it always causes a `Invoke With Error`; the following invocation causes a new `INIT Phase` for that Lambda's virtual environment or provisioning.

This will always result in a coldish start. The provisioned virtual environment will start the INIT Phase again.

If enough errors occur for a single provisioned Lambda's, AWS will reap that provisioned virtual environment and create a new one for the next invocation. 

From experience, AWS monitors how often a Lambda is erroring out and will auto-descale them regardless of scaling or provisioning rules.

New Relic's Lambda monitoring can monitor errors without the Lambda erroring out, ensuring performance and monitoring needs are met. 

### Custom Error Notification With New Relic's Nuget NewRelic.Agent.Api

An instrumented Lambda, layer, layerless or containerized, can utilize NewRelic.Agent.Api Nuget, which can alter an invocation's instrumentation metadata.

After installing `<PackageReference Include="NewRelic.Agent.Api" Version="10.26.0" />`

This Nuget is not the same as `NewRelic.Agent` which is used to instrument Lambda's without New Relic's Lambda layer. `NewRelic.Agent.Api` is compatible with `NewRelic.Agent`!

In the `function.cs` is the Lambda responding to an Api Gateway; the source really does not matter. To simulate an unhandled exception that bubbled all the way up, there is a try-catch that will handle it.
This try catch will respond with a 500 response to Api Gateway and use the following line:

`NewRelic.Api.Agent.NewRelic.NoticeError(e);`

The Lambda will exit successfully and avoid performance issues with the following invocation. However, this invocation is now marked as an error regardless of the Cloudwatch data. This invocation is tracked through New Relic's `Error Triage` and error metrics.

Now, you can monitor Lambda using New Relic native dashboards, metrics, and alerts to inform reports about internal errors without incurring performance penalties for invocation errors.

#### Custom Error/Invocation Tracking and the NewRelic.Agent.Api

`NoticedError` is callable anytime during a lambda invocation and can be used deeper in the call stack to avoid the need of bubbling the exception to the top.

Also, you can pass in custom exceptions to improve the feedback in `Error Triage` with the following:

Just some sudo code deep inside a repository client

```C#

    public async Task<JokeResponse> GetDadJokeFromQueueID(string id) {

        try {
            var response = await client.getDadJokes(id);

            return new () {
                id = id,
                joke = response.joke,
                success = true,
            }

        } catch(e) {
            `NewRelic.Api.Agent.NewRelic.NoticeError(e);`
        }

    }
```

Tracking failed business logic is trackable without throwing an exception and still show show up on the `Error Triage` dashboard

```C#

    public async Task<boolean> ValidIdDoesNotExist (string id) {

        var response = await client.doesIdExist(id);
       
        if(response.existing == true) { // meaning it is an existing id
            // an Exception is an object is optional
            NewRelic.Api.Agent.NewRelic.NoticeError(new ExistingIDException($"ID: {id} is already existing"));`
            return false;
        } else {
            return response.existing;
        }
    }
```

Now New Relic can respond to any number of problems that also does not incurr an invocation error. This will help with less time sorting through logs for problems
but directly using New Relic Lambda's built in `Dashboards` and `Error Triage` reducing the frustraction and creating custom alerts based on log messages.

Invocation/Transcation can be found with the Exception Class and the error message. To increase searchablity of `Error Triage` or `Invocations` you can add custom attributes:

```C#
    public async Task<boolean> ValidIdDoesNotExist (IdLookUp idLookup) {

        var response = await client.doesIdExist(idLookup.id);
       
        if(response.existing == true) { // meaning it is an existing id
            // an Exception is an object is optional
            var agent = NewRelic.Api.Agent.NewRelic.GetAgent();

            var transaction = agent.CurrentTransaction;

            transaction.AddCustomAttribute("correlation-id", idLookup.correlationId);

            NewRelic.Api.Agent.NewRelic.NoticeError(new ExistingIDException($"ID: {idLookup.id} is already existing"));`
            return false;
        } else {
            return response.existing;
        }
    }

```

Now any invocation in `Error Triage` and `Invocations` can be found with a corrlection id or any additional metadata. 

This will greatly enhanced the usablity of the builtin New Relic's Lambda Dashboards which can be directly access through New Relic's Workloads and feed better monitoring into New Relic's Workflows!

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

### Function.cs

Lambda functions written in .NET are C# classes. The runtime loads them
just like any C# class, and then invokes the handler function for each 
invocation event.

The New Relic .NET Agent is used to instrument your AWS Lambda.  In most cases, 
the agent automatically instruments your AWS Lambda function handler.  The layer 
used in this example includes both the agent and the required New Relic Lambda 
Extension.  When instrumenting an AWS Lambda, the .NET Agent relies on the Lambda 
Extension to send telemetry to New Relic.
