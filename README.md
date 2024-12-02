[![Community Plus header](https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Plus.png)](https://opensource.newrelic.com/oss-category/#community-plus)

# newrelic-lambda-extension [![Build Status](https://github.com/newrelic/newrelic-lambda-extension/actions/workflows/build-release-assets.yml/badge.svg)](https://github.com/newrelic/newrelic-lambda-extension/actions/workflows/build-release-assets.yml) [![Coverage](https://codecov.io/gh/newrelic/newrelic-lambda-extension/branch/main/graph/badge.svg?token=T73UEDVA5K)](https://codecov.io/gh/newrelic/newrelic-lambda-extension)

An AWS Lambda extension to collect, enhance, and transport telemetry data from your AWS Lambda functions to New Relic without requiring an external transport such as CloudWatch Logs or Kinesis.

This lightweight AWS Lambda Extension runs alongside your AWS Lambda functions and automatically handles the collection and transport of telemetry data from
supported New Relic serverless agents. The extension requires a telemetry payload from a New Relic agent. Conditions that delay or prevent that payload from being written may result in longer-than-expected invocation durations.

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
# build extension for x86_64 architecture
make dist-x86_64

# build extension for arm64 architecture
make dist-arm64
```

This creates the extension binary in `./extensions/newrelic-lambda-extension`. The binary is compiled for Amazon Linux, which is likely different from the platform you're working on.

## Deploying

To publish the extension to your AWS account, run the following command:

```sh
    make publish
```

This packages the extension, and publishes a new layer version in your AWS account. Be sure that the AWS CLI is configured correctly. You can use the usual [AWS CLI environment variables](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html) to control the account and region for the CLI.

## Startup Checks

This Lambda Extension will perform a series of checks on initialization to help customer troubleshoot configuration. Should any of
these checks fail, the extension wil attempt to output recommendations to both
CloudWatch Logs and New Relic Logs. If you have any issues using this extension, be sure
to check your logs for messages starting with `Startup check warning:` for
troubleshooting recommendations. It is recommended to ignore all Extension checks after the lambda is successfully instrumented as mentioned [here](#extension-environment-variables).

Startup checks include:
| Check name | Description |
|--------|-----------|
| *agent* | New Relic agent version check |
| *handler* | Lambda handler configuration check |
| *sanity* | Lambda environment variable, SSM, & Secrets Manager checks for license key |
| *vendor* | If Vendored New Relic agent added along with layer check |

## Logging

The New Relic Lambda Extension can also send your function's logs to New Relic. If you use the Lambda Extension, you can avoid the CloudWatch Logs ingest charge for telemetry gathered by New Relic. For additional Lambda Extension environment variables, you can refer to the [docs](https://docs.newrelic.com/docs/serverless-function-monitoring/aws-lambda-monitoring/instrument-lambda-function/introduction-lambda/#extension). You can use the following extension environment variable for logging

| Environment variable | Default value | Options | Description |
|--------|-----------|-------------|-------------|
| `NEW_RELIC_EXTENSION_SEND_FUNCTION_LOGS` | `false` | `true` , `false` | Send function logs to New Relic. |
| `NEW_RELIC_EXTENSION_SEND_EXTENSION_LOGS` | `false` | `true` , `false` | Send extension logs in addition to the function logs to New Relic. |
| `NEW_RELIC_EXTENSION_LOGS_ENABLED` | `true` | `true` , `false` | Enable or disable `[NR_EXT]` log lines |
| `NR_TAGS` |  | | Specify tags to be added to all log events. **Optional**. Each tag is composed of a colon-delimited key and value. Multiple key-value pairs are semicolon-delimited; for example, env:prod;team:myTeam. |
| `NR_ENV_DELIMITER` | | | Some users in UTF-8 environments might face difficulty in defining strings of `NR_TAGS` delimited by the semicolon `;` character. Use `NR_ENV_DELIMITER`, to set custom delimiter for `NR_TAGS`. |

## Extension Environment variables

The New Relic Lambda Extension offers various features, which can be utilised by using the Lambda environment variables. These include:
| Environment variable | Default value | Options | Description |
|--------|-----------|-------------|-------------|
|`NEW_RELIC_IGNORE_EXTENSION_CHECKS`| `false` | `all` , `agent`, `handler`, `sanity`, `vendor` | Ignore selected Extension Checks by using a comma-separated value, e.g., `agent,handler`, to ignore agent and handler checks. Use `all` to ignore all the Extension Checks as mentioned [here](#startup-checks). It is recommended to ignore all Extension checks after the lambda is successfully instrumented. |
|`NEW_RELIC_DATA_COLLECTION_TIMEOUT`| `10s` | Time such as `5s`. Valid time units are "ms", "s"| Reduce time the Extension waits for sending telemetry.|
|`NEW_RELIC_LAMBDA_EXTENSION_ENABLED`| `false` | `true` , `false` | Disable the Extension. It is enabled by default |


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
