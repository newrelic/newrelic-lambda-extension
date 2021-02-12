import json
import boto3
import os

QUEUE_URL = os.environ.get('QUEUE_URL')

def lambda_handler(event, context):

    sqs = boto3.client("sqs")

    message_data = sqs.send_message(
        QueueUrl=QUEUE_URL,
        MessageBody="Hello",
    )

    print("Sent message id %s to queue URL %s" % (message_data["MessageId"], QUEUE_URL))

    return {
        "statusCode": 200,
        "body": json.dumps(message_data),
    }
