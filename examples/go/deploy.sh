#!/bin/bash

region="us-east-1"
if [ -n "$1" ] ; then
  region=$1
  echo "region set to ${region}"
fi

#TODO: change byol to provided; default to "go1.x"
runtime="byol"

handler="handler"
build_tags=""
if [ "byol" == $runtime ] ; then
  echo "Building stand-alone lambda"
  build_tags="-tags lambda.norpc"
  handler="bootstrap"
fi

env GOARCH=amd64 GOOS=linux go build ${build_tags} -ldflags="-s -w" -o ${handler}
zip go-example.zip "${handler}"

bucket="newrelic-example-${region}"
aws s3 mb s3://${bucket}
aws s3 cp go-example.zip s3://${bucket}
aws cloudformation deploy --region ${region} --template-file template.yaml --stack-name NewrelicExampleGo --capabilities CAPABILITY_NAMED_IAM
