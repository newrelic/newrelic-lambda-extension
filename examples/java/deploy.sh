#!/bin/bash

accountId=$1

region=$2
echo "region set to ${region}"

# TODO --use-container after deps are published
sam build #--use-container

bucket="newrelic-example-${region}"

aws s3 mb s3://${bucket}

sam package --region ${region} --s3-bucket=${bucket} --output-template-file packaged.yaml
aws cloudformation deploy --region ${region} \
  --template-file packaged.yaml \
  --stack-name NewrelicExampleJava \
  --capabilities CAPABILITY_IAM \
  --parameter-overrides "NRAccountId=${accountId}"
