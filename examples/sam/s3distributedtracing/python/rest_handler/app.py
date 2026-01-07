import json
import os
import uuid

# Note that neither of these dependencies needs to be packaged with your function. boto3 is part of the Lambda
# platform and newrelic is packaged with the layer
import boto3
import newrelic.agent

BUCKET_NAME = os.environ.get('BUCKET_NAME')

# We're using the copy-paste style of browser agent integration here.
# Retrieve configuration from environment variables. 
NR_ACCOUNT_ID = os.environ.get('NEW_RELIC_ACCOUNT_ID') or ""
NR_TRUST_KEY = os.environ.get('NEW_RELIC_TRUSTED_ACCOUNT_KEY') or ""
NR_BROWSER_APP_ID = os.environ.get('NEW_RELIC_BROWSER_APPLICATION_ID') or ""
NR_BROWSER_AGENT_ID = os.environ.get('NEW_RELIC_BROWSER_AGENT_ID') or ""
NR_LICENSE_KEY = os.environ.get('NEW_RELIC_BROWSER_KEY') or ""

BROWSER_AGENT = f"""
<script type="text/javascript">
;NREUM.loader_config={{accountID:"{NR_ACCOUNT_ID}",trustKey:"{NR_TRUST_KEY}",agentID:"{NR_BROWSER_AGENT_ID}",licenseKey:"{NR_LICENSE_KEY}",applicationID:"{NR_BROWSER_APP_ID}"}};
;NREUM.info={{beacon:"bam.nr-data.net",errorBeacon:"bam.nr-data.net",licenseKey:"{NR_LICENSE_KEY}",applicationID:"{NR_BROWSER_APP_ID}",sa:1}};
;/* New Relic Browser Agent Snippet */
(function(){{var e=window.NREUM||(window.NREUM={{}});e.loader_config=NREUM.loader_config;e.info=NREUM.info;}})();
</script>
"""

# This string literal is our response to GET requests. It's probably best to manage static resources differently,
# such as by using an S3 bucket. But this approach keeps the example simple.
GET_RESPONSE = """
<html>
<head>
<title>NRDT S3 Demo App</title>
""" + BROWSER_AGENT + """
</head>
<body>
<form id="post_me" name="post_me" method="POST" action="">
    <label for="message">Message</label>
    <input id="message" name="message" type="text" value="Hello world" />
    <button type="submit" name="submit">Submit</button>
</form>
<div id="output" style="white-space: pre-wrap; font-family: monospace;">
</div>
<script>
const formElem = document.getElementById("post_me");
const messageElem = document.getElementById("message");
formElem.addEventListener("submit", (ev) => {
    newrelic.interaction()
        .setName("submitMessage")
        .save();
    fetch(location.href, {
        "method": "POST",
        "body": messageElem.value
    })
    .then(resp => resp.text())
    .then(body => {
        document.getElementById("output").innerText = body;
    });
    ev.preventDefault();
});
</script>
</body>
</html>
"""


def nr_trace_context_dict():
    """Generate a distributed trace context as a dictionary for S3 metadata"""
    # The Python agent expects a list as an out-param
    dt_headers = []
    newrelic.agent.insert_distributed_trace_headers(headers=dt_headers)
    # Convert the list of tuples to a dict for S3 metadata
    return dict(dt_headers)


def upload_word_to_s3(word):
    """Uploads a word to S3 with trace context as metadata."""
    # Get the S3 client
    s3 = boto3.client("s3")
    
    # Generate a unique key for this word
    key = f"words/{uuid.uuid4()}.txt"
    
    # Get trace context as metadata
    trace_metadata = nr_trace_context_dict()
    
    # Upload to S3 with metadata
    s3.put_object(
        Bucket=BUCKET_NAME,
        Key=key,
        Body=word.encode('utf-8'),
        Metadata=trace_metadata
    )
    
    return key


def upload_words_to_s3(words):
    """Turn a list of strings into S3 objects with trace context metadata"""
    uploaded_keys = []
    
    for word in words:
        key = upload_word_to_s3(word)
        uploaded_keys.append(key)
    
    return uploaded_keys


import logging

logger = logging.getLogger()
logger.setLevel(logging.INFO)

def lambda_handler(event, context):
    # Log trace details for verification
    metadata = newrelic.agent.get_linking_metadata()
    logger.info(f"New Relic Trace Details - Trace ID: {metadata.get('trace.id')}, Span ID: {metadata.get('span.id')}")

    if event['httpMethod'] == 'GET':
        # For our example, we return a static HTML page in response to GET requests
        return {
            "statusCode": 200,
            "headers": {
                "Content-Type": "text/html"
            },
            "isBase64Encoded": False,
            "body": GET_RESPONSE
        }
    elif event['httpMethod'] == 'POST':
        # Handle POST requests by splitting the post body into words, and uploading each to S3
        body = event.get('body', '') or ''
        words = body.split()
        uploaded_keys = upload_words_to_s3(words)
        # Returns the list of uploaded S3 keys
        return {
            "statusCode": 200,
            "headers": {
                "Content-Type": "application/json"
            },
            "isBase64Encoded": False,
            "body": json.dumps({"uploaded_keys": uploaded_keys}),
        }
