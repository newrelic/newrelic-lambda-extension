# Distribute Tracing example

Here, we demonstrate trace context propagation for Distributed Tracing, in several non-trivial configurations.

- We start with a static HTML page, served by a Python Lambda function, and instrumented using the New Relic 
  Browser Agent. It presents a text box, and submits the text via an HTTP POST to an API Gateway proxy endpoint,
  backed by that same Python Lambda function.
- In response to the POST, the Python Lambda splits the message into words, and sends each word as a message
  to an SQS queue, propagating the trace context in the SQS message headers.
- A NodeJS Lambda function is triggered by the SQS Queue. It processes messages, and picks up the Trace Context from
  the headers. Each SQS message is produced to an SNS Topic, and the trace headers are propagated.
- A Java Lambda function is triggered by the SNS topic. It accepts the trace context, and simply logs the messages.

Each function (and the Browser Agent) produces spans, which are sent to New Relic. In general, Lambda telemetry is
buffered, and sent to New Relic during some _subsequent_ invocation. So, depending on how the buffering works out, 
some spans will be sent before others, and the trace may be temporarily missing some data, until all the Lambda 
function instances are either invoked an additional time, or shut down. 
