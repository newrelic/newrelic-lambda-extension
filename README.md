[![Community Project header](https://github.com/newrelic/opensource-website/raw/master/src/images/categories/Community_Project.png)](https://opensource.newrelic.com/oss-category/#community-project)

# newrelic-lambda-extension [![Build Status](https://circleci.com/gh/newrelic/newrelic-lambda-extension.svg?style=svg)](https://circleci.com/gh/newrelic/newrelic-lambda-extension) [![Coverage](https://codecov.io/gh/newrelic/newrelic-lambda-extension/branch/main/graph/badge.svg?token=T73UEDVA5K)](https://codecov.io/gh/newrelic/newrelic-lambda-extension)

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

After deploying your AWS Lambda function with one of the layer ARNs from the
link above you should begin seeing telemetry data in New Relic.

See below for details on supported New Relic agents.

## Supported Configurations

AWS's [Extension API supports](https://docs.aws.amazon.com/lambda/latest/dg/runtimes-extensions-api.html) only a subset 
of all their runtimes. Notably absent as of this writing are Node JS before 10, Python before 3.7, Go (all versions), 
Dotnet before 3.1, and the older "java8" runtime, though "java8.al2" is supported.

For Go lambdas, we suggest using "provided" or "provided.al2". The Go example's deploy script contains compiler flags
that produce a suitable self-hosting Go executable. See the [Custom runtime](https://docs.aws.amazon.com/lambda/latest/dg/runtimes-custom.html)
docs for more details on this feature. 

All of our layers include the extension, and the latest Agent version for the Layer's runtime. The latest 
layer version ARNs for your runtime and region are available [here](https://layers.newrelic-external.com/). The 
`NewRelicLambdaExtension` layer is suitable for Go, Java and Dotnet.

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

## Disabling Extension

The New Relic Lambda Extension is enabled by default. To disable it, after adding or
updating the Lambda layer, set the `NEW_RELIC_LAMBDA_EXTENSION_ENABLED` environment
variable to `false`.

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
