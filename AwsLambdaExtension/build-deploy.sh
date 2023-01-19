#!/bin/bash

if [ ! -d "./extensions" ]; then
    mkdir ./extensions
fi

GOOS=linux GOARCH=arm64 go build -o ./extensions/AwsLambdaExtension main.go
chmod +x ./extensions/AwsLambdaExtension

#aws lambda publish-layer-version \
#    --layer-name "AwsLambdaExtension" \
#    --description "New Relic Telemetry API Extension" \
#    --compatible-architectures "arm64" \
#    --zip-file  "fileb://extension.zip"
