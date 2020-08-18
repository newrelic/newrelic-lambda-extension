#!/bin/bash

accountId=$1

region=$2
echo "region set to ${region}"

# In this example, our Go lambda can use the "norpc" option to slim down the deployment package
# a little. That requires some custom packaging though.
#TODO: change byol to provided
runtime="byol"

handler="handler"
build_tags=""
if [ "go1.x" != $runtime ] ; then
  echo "Building stand-alone lambda"
  build_tags="-tags lambda.norpc"
  handler="bootstrap"
fi

env GOARCH=amd64 GOOS=linux go build ${build_tags} -ldflags="-s -w" -o ${handler}
zip go-example.zip "${handler}"

bucket="newrelic-example-${region}"
aws s3 mb s3://${bucket}
aws s3 cp go-example.zip s3://${bucket}
aws cloudformation deploy --region ${region} \
  --template-file template.yaml \
  --stack-name NewrelicExampleGo \
  --capabilities CAPABILITY_IAM \
  --parameter-overrides "NRAccountId=${accountId}"
