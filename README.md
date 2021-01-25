[![Community Project header](https://github.com/newrelic/opensource-website/raw/master/src/images/categories/Community_Project.png)](https://opensource.newrelic.com/oss-category/#community-project)

# newrelic-lambda-extension [![Build Status](https://circleci.com/gh/newrelic/newrelic-lambda-extension.svg?style=svg)](https://circleci.com/gh/newrelic/newrelic-lambda-extension)

An AWS Lambda extension to collect, enhance, and transport telemetry data from your AWS Lambda functions to New Relic without requiring an external transport such as CloudWatch Logs or Kinesis.

This lightweight AWS Lambda Extension runs alongside your AWS Lambda functions and automatically handles the collection and transport of telemetry data from
supported New Relic serverless agents.

## Installation

To install the extension, simply include the layer with your instrumented
Lambda function. The current layer ARN can be found [here][3].

[3]: https://layers.newrelic-external.com

**Note:** This extension is included with all New Relic AWS Lambda layers going forward.

You'll also need to make the New Relic license key available to the extension. Use the [New Relic Lambda CLI][4]
to install the managed secret, and then add the permission for the secret to your Lambda execution role.

[4]: https://github.com/newrelic/newrelic-lambda-cli

    newrelic-lambda integrations install \
        --nr-account-id <account id> \
        --nr-api-key <api key> \
        --linked-account-name <linked account name> \
        --enable-license-key-secret

Each of the example functions in the `examples` directory has the appropriate license key secret permission. 

The New Relic Lambda Extension is disabled by default. To enable it, after adding or
updating the Lambda layer, set the `NEW_RELIC_LAMBDA_EXTENSION_ENABLED` environment
variable to any value.

After deploying your AWS Lambda function with one of the layer ARNs from the
link above you should begin seeing telemetry data in New Relic.

See below for details on supported New Relic agents.

## Supported Agents

1. Node Agent, version [v6.13.1](https://github.com/newrelic/node-newrelic/releases/tag/v6.13.1), via layer versions
 NewRelicNodeJS12X:20, NewRelicNodeJS10X:22, NewRelicNodeJS810:20 
2. Python Agent, version [v5.18.0.148](https://github.com/newrelic/newrelic-python-agent/releases/tag/v5.18.0.148) via layer versions NewRelicPython38:18, NewRelicPython37:22, NewRelicPython36:21, NewRelicPython27:21
3. Go Agent, version [v3.9.0](https://github.com/newrelic/go-agent/releases/tag/v3.9.0)
4. Java
   - `com.newrelic.opentracing:newrelic-java-lambda` [v2.1.2](https://github.com/newrelic/newrelic-lambda-tracer-java/releases/tag/v2.1.2)
   - `com.newrelic.opentracing:java-aws-lambda` [v2.1.0](https://github.com/newrelic/java-aws-lambda/releases/tag/v2.1.0)
5. Dotnet: [v1.1.0](https://github.com/newrelic/newrelic-dotnet-agent/releases/tag/AwsLambdaOpenTracer_v1.1.0)

Note that future agent layers (for Node and Python) will include the extension. To test with a different extension version, make sure that the layer for the version you want to run is **after** the agent, so that it overwrites the packaged extension. 

For other runtimes, be sure to include the latest `NewRelicLambdaExtension` layer.

## Building

Use the included `Makefile` to compile the extension. 

```sh
make dist
```

This creates the extension binary in `./extensions/newrelic-lambda-extension`. The binary is compiled for Amazon Linux, which is likely different from the platform you're working on.

## Deploying

To publish the extension to your AWS account, run the following command:

```sh
    make publish
```

This packages the extension, and publishes a new layer version in your AWS account. Be sure that the AWS CLI is configured correctly. You can use the usual [AWS CLI environment variables](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html) to control the account and region for the CLI.

## Startup Checks

This Lambda Extension will perform a series of checks on initialization. Should any of
these checks fail, the extension wil attempt to output troubleshooting recommendations to both
CloudWatch Logs and New Relic Logs. If you have any issues using this extension, be sure
to check your logs for messages starting with `Startup check failed:` for
troubleshooting recommendations.

Startup checks include:

* New Relic agent version checks
* Lambda handler configuration checks
* Lambda environment variable checks
* Vendored New Relic agent checks

## Testing

To test locally, acquire the AWS extension test harness first. Then:

>TODO: Link to the AWS SDK that has the test harness, assuming it gets published.

1. (Optional) Use the `newrelic-lambda` CLI to create the license key managed secret in your AWS account and region.
2. Build the docker container for sample function code. Give it the tag `lambda_ext`.
   - Be sure to include your lambda function in the container.
3. Start up your container.

   - Using AWS Secret Manager
   
            export AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id --profile default)
            export AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key --profile default)
            export AWS_SESSION_TOKEN=$(aws configure get aws_session_token --profile default)
    
            docker run --rm -v $(pwd)/extensions:/opt/extensions -p 9001:8080 \
                -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY -e AWS_SESSION_TOKEN \
                lambda_ext:latest \
                -h function.handler -c '{}' -t 60000

   - Or, setting the license key directly
   
            docker run --rm \
                -v $(pwd)/extensions:/opt/extensions \
                -p 9001:8080 \
                lambda_ext:latest \
                -h function.handler -c '{"NEW_RELIC_LICENSE_KEY": "your-license-key-here"}' -t 60000

4. To invoke the sample lambda run: 
   
       curl -XPOST 'http://localhost:9001/2015-03-31/functions/function.handler/invocations' \
           -d 'invoke-payload'

5. Finally, you can exercise the container shutdown lifecycle event with:

        curl -XPOST 'http://localhost:9001/test/shutdown' \
            -d '{"timeoutMs": 5000 }'

## Support

New Relic hosts and moderates an online forum where customers can interact with New Relic employees as well as other customers to get help and share best practices. Like all official New Relic open source projects, there's a related Community topic in the New Relic Explorers Hub. You can find this project's topic/threads [in the Explorers Hub](https://discuss.newrelic.com/t/new-relic-lambda-extension/111715).

## Contributing

We encourage your contributions to improve `newrelic-lambda-extension`! Keep in mind when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA one time per project.

If you have any questions, or to execute our corporate CLA, required if your contribution is on behalf of a company,  please drop us an email at opensource@newrelic.com.

## License
`newrelic-lambda-extension` is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License. The `newrelic-lambda-extension` also uses source code from third-party libraries. You can find full details on which libraries are used and the terms under which they are licensed in the [third-party notices document](THIRD_PARTY_NOTICES.md).
