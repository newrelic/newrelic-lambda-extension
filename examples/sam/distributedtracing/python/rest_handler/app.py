import json
import os
import uuid

import boto3
import newrelic

QUEUE_URL = os.environ.get('QUEUE_URL')


def nrTraceContextJson():
    dt_headers = []
    newrelic.agent.insert_distributed_trace_headers(headers=dt_headers)
    return json.dumps(dict(dt_headers))


def lambda_handler(event, context):
    sqs = boto3.client("sqs")

    send_status = sqs.send_message_batch(
        QueueUrl=QUEUE_URL,
        Entries=[
            {
                'Id': str(uuid.uuid1()),
                'MessageBody': "Hello",
                'MessageAttributes': {
                    "NRDT": {
                        'DataType': 'String',
                        'StringValue': nrTraceContextJson()
                    }
                }
            },
            {
                'Id': str(uuid.uuid1()),
                'MessageBody': "World",
                'MessageAttributes': {
                    "NRDT": {
                        'DataType': 'String',
                        'StringValue': nrTraceContextJson()
                    }
                }
            },
        ]
    )

    return {
        "statusCode": 200,
        "body": json.dumps(send_status),
    }
