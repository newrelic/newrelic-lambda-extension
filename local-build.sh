#!/bin/bash

set -x

echo "Building newrelic-lambda-extension..."

cat << EOF > Dockerfile
FROM golang:latest
WORKDIR /newrelic-lambda-extension
ENTRYPOINT [ "./build-arm64.sh" ]
EOF

cat << EOF > build-arm64.sh
#!/bin/sh
set -xeu

go mod tidy 
rm -rf extensions 
rm -f preview-extensions-ggqizro707 
rm -f /tmp/newrelic-lambda-extension.x86_64.zip 
rm -f /tmp/newrelic-lambda-extension.arm64.zip 
env GOARCH=arm64 GOOS=linux go build -ldflags="-s -w" -o ./extensions/newrelic-lambda-extension 
touch preview-extensions-ggqizro707
EOF

chmod +x build-arm64.sh

docker build . -t build-lambda-extension-m1:latest
docker run -v /Users/emilio/Dev/newrelic-lambda-extension:/newrelic-lambda-extension build-lambda-extension-m1:latest

rm Dockerfile
rm build-arm64.sh
echo "Done"

echo "Building Telemetry API Extension..."
cd AwsLambdaExtension

cat << EOF > Dockerfile
FROM golang:latest
WORKDIR /AwsLambdaExtension
ENTRYPOINT [ "./build-deploy.sh" ]
EOF

docker build . -t build-telemetry-extension-m1:latest
docker run -v /Users/emilio/Dev/newrelic-lambda-extension/AwsLambdaExtension:/AwsLambdaExtension build-telemetry-extension-m1:latest

rm Dockerfile
zip -r extension.zip ./extensions/
mv extension.zip ../extensions
mv extensions/AwsLambdaExtension ../extensions
echo "Done"
exit 0
