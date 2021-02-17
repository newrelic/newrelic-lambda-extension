'use strict';
const newrelic = require('newrelic');

/**
 * A Lambda function that logs the payload received from SQS.
 */
exports.sqsPayloadLoggerHandler = async (event, context) => {
    event.Records.forEach(r => {
        console.info(JSON.stringify(event));
        if (r.messageAttributes.NRDT) {
            let nrdtHeaders = JSON.parse(r.messageAttributes.NRDT.stringValue);
            console.info("NRDT headers: %s", JSON.stringify(nrdtHeaders));

            nrdtHeaders = nrdtHeaders.reduce((o, p) => { o[p[0]] = p[1]; return o;}, {});
            console.info("NRDT headers processed: %s", JSON.stringify(nrdtHeaders));
            newrelic.getTransaction().acceptDistributedTraceHeaders("Queue", nrdtHeaders);
        }
    })
}
