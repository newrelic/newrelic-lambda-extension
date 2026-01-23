"use strict";
const newrelic = require("newrelic");
const AWS = require("aws-sdk");

/**
 * A Lambda function that processes S3 events, retrieves trace context from object metadata,
 * and logs the object content.
 */
exports.s3Handler = async (event, context) => {
  // An S3 invocation event consists of a batch of one or more S3 event records
  console.info(JSON.stringify(event));

  const transaction = newrelic.getTransaction();
  const s3 = new AWS.S3();

  await Promise.all(
    event.Records.map(async (record) => {
      // Extract bucket and key from the S3 event
      const bucket = record.s3.bucket.name;
      const key = decodeURIComponent(record.s3.object.key.replace(/\+/g, " "));

      console.log(`Processing S3 object: ${bucket}/${key}`);

      // Get the object including its metadata
      const s3Object = await s3
        .getObject({
          Bucket: bucket,
          Key: key,
        })
        .promise();

      // The object body contains the word
      const word = s3Object.Body.toString("utf-8");

      // The trace context is stored in the object's metadata
      const traceContext = s3Object.Metadata || {};

      // Accept the distributed trace headers from S3 metadata
      if (Object.keys(traceContext).length > 0) {
        transaction.acceptDistributedTraceHeaders("S3", traceContext);
      }

      // Log trace details for verification
      const metadata = newrelic.getLinkingMetadata();
      console.info(
        `New Relic Trace Details - Trace ID: ${metadata["trace.id"]}, Span ID: ${metadata["span.id"]}`
      );

      console.log("Processed S3 object %s with content: %s", key, word);
    })
  );

  transaction.end();
};
