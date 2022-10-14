#!/bin/sh
set -xeu

go mod tidy
rm -rf extensions
rm -f preview-extensions-ggqizro707
rm -f /tmp/newrelic-lambda-extension.arm64.zip
rm -f /tmp/newrelic-lambda-extension.x86_64.zip
env GOARCH=arm64 GOOS=linux go build -ldflags="-s -w" -o ./localExtensions/arm64/newrelic-lambda-extension
env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o ./localExtensions/x86_64/newrelic-lambda-extension
touch preview-extensions-ggqizro707
