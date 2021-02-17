'use strict';
const newrelic = require('newrelic');
const AWS = require('aws-sdk')


/**
 * A Lambda function that logs the payload received from SQS.
 */
exports.sqsPayloadLoggerHandler = async (event, context) => {
    // An SQS invocation event consists of a batch of one or more messages
    console.info(JSON.stringify(event));
    const SNS_TOPIC_ARN = process.env.SNS_TOPIC;
    console.log("SNS_TOPIC_ARN: %s", SNS_TOPIC_ARN);

    const transaction = newrelic.getTransaction();

    await Promise.all(event.Records.map(async r => {

        // Find, parse, and attach the trace context
        if (r.messageAttributes.NRDT) {
            let traceContext = JSON.parse(r.messageAttributes.NRDT.stringValue);
            transaction.acceptDistributedTraceHeaders("Queue", traceContext);
        }

        console.log("Processed message %s with body %s", r.messageId, r.body);

        const traceContextObject = {};
        transaction.insertDistributedTraceHeaders(traceContextObject);
        const traceContextJson = JSON.stringify(traceContextObject);

        const sns = new AWS.SNS({apiVersion: '2010-03-31'});
        const params = {
            Message: r.body,
            TopicArn: SNS_TOPIC_ARN,
            MessageAttributes: {
                "NRDT": {
                    "DataType": "String",
                    "StringValue": traceContextJson,
                }
            }
        };
        return sns.publish(params).promise();
    }).map(async sendPromise => {
        const data = await sendPromise;
        console.log("SNS MessageID is " + data.MessageId);
    }));
    console.log("Done sending")

    transaction.end();
}
