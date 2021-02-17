import json
import boto3
import os
import newrelic

QUEUE_URL = os.environ.get('QUEUE_URL')

def lambda_handler(event, context):

    sqs = boto3.client("sqs")

    dt_headers = []
    newrelic.agent.insert_distributed_trace_headers(headers=dt_headers)

    message_data = sqs.send_message(
        QueueUrl=QUEUE_URL,
        MessageBody="Hello",
        MessageAttributes={
            "NRDT": {
                'DataType': 'String',
                'StringValue': json.dumps(dt_headers)
            }
        }
    )

    print("Sent message id %s to queue URL %s" % (message_data["MessageId"], QUEUE_URL))

    return {
        "statusCode": 200,
        "body": json.dumps(message_data),
    }
