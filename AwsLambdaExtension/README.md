# New Relic Telemetry API Extension

The New Relic Telemetry API Extension collects telemetry data from both Lambda Telemetry API and New Relic Agents and sends it to New Relic.
The following environment variables can be used to configre it:

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
