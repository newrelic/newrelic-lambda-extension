# New Relic Telemetry API Extension

The New Relic Telemetry API Extension collects telemetry data from both AWS Lambda Telemetry API and New Relic Agents and sends it to New Relic. Please note that Telemetry API collects all logs for an invocation, so when using an agent do not forward logs or they will be duplicated. If you are using a New Relic Agent, be sure to follow its [lambda/serverless installation documentation](https://docs.newrelic.com/docs/serverless-function-monitoring/aws-lambda-monitoring/enable-lambda-monitoring/instrument-example/). The use of this layer does not require a New Relic agent to be present, and can be configured to send telemetry without one. The following environment variables are available to configure this extension:

| Environment Variable | Required | Description |
| --- | --- | --- |
| NEW_RELIC_ACCOUNT_ID | **always** | The account ID of the New Relic account you want to send data to |
| NEW_RELIC_LICENSE_KEY | *conditional* | A plaintext New Relic license key. Either this or the value retrieved from NEW_RELIC_LICENSE_KEY_SECRET must contain a valid New Relic license key or the application will exit. If a plaintext license key is provided, it will override the license key retrieved from an AWS Secret. |
| NEW_RELIC_LICENSE_KEY_SECRET | *conditional* | The name of an AWS Secrets Manager Secret containing a New Relic license key. If no plaintext license key is provided, the value of this variable must be set to the name of an AWS Secret containing a valid New Relic license key. |
| NEW_RELIC_EXTENSION_AGENT_DATA_COLLECTION_ENABLED | optional | Setting this to "false" will prevent the extension from collecting data from New Relic Agents. This will not prevent the agent from running. |
| NEW_RELIC_EXTENSION_AGENT_DATA_BATCH_SIZE | optional | The number of invocations to store before sending Agent Data to New Relic. If your lamba function gets invoked at a high frequency, increasing this number will improve the performance of the extension and avoid dropped data and improve performance. Default: 1 |
| NEW_RELIC_EXTENSION_TELEMETRY_API_BATCH_SIZE | optional | The number of Telemetry API events and logs to batch before sending them to New Relic. If your application invokes frequently, increase this number to avoid data getting dropped and to improve performance. Default: 1 | 
| NEW_RELIC_EXTENSION_DATA_COLLECTION_TIMEOUT |  optional | A valid time.Duration string for how long the extension should wait to attempt to send agent data to New Relic in the event of a timeout/retry loop scenario. Example: 1s, 1500ms; Default: 10s |
| NEW_RELIC_EXTENSION_COLLECTOR_OVERRIDE | optional | An override for the New Relic collection endpoint you want to send data to. By default, this will be detected based on the region of your New Relic license key. |
| NEW_RELIC_EXTENSION_LOG_LEVEL | optional | The log level of the New Relic Telemetry API Extension. For more verbose logs, set to "debug". For error logs only, set to "error". |

## Installation

To install the extension, simply include the layer with your instrumented
Lambda function. The current layer ARN can be found [here][3].

[3]: https://layers.newrelic-external.com

**Note:** This extension is included with all New Relic AWS Lambda layers going forward.

## Building Locally

Use the `make build-arm64` or `make build-amd64` commands to build local version of this binary. Make sure that you have docker enabled and conifgured properly. This runs the build command in a linux docker container to prevent errors caused by MacOS filesystem artifacts from occuring.

## Support

New Relic hosts and moderates an online forum where customers can interact with New Relic employees as well as other customers to get help and share best practices. Like all official New Relic open source projects, there's a related Community topic in the New Relic Explorers Hub. You can find this project's topic/threads [in the Explorers Hub](https://discuss.newrelic.com/t/new-relic-lambda-extension/111715).

## Contributing

We encourage your contributions to improve `newrelic-lambda-extension`! Keep in mind when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA one time per project.

If you have any questions, or to execute our corporate CLA, required if your contribution is on behalf of a company,  please drop us an email at opensource@newrelic.com.

## License
`newrelic-lambda-extension` is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License. The `newrelic-lambda-extension` also uses source code from third-party libraries. You can find full details on which libraries are used and the terms under which they are licensed in the [third-party notices document](THIRD_PARTY_NOTICES.md).
