if [ ! -d "bin" ]; then
    mkdir bin
fi
if [ ! -d "bin/extensions" ]; then
    mkdir bin/extensions
fi
GOOS=linux GOARCH=amd64 go build -o bin/extensions/AwsLambdaExtension main.go
chmod +x bin/extensions/AwsLambdaExtension
cd bin
zip -r extension.zip extensions/
aws lambda publish-layer-version \
    --layer-name "AwsLambdaExtension" \
    --description "New Relic Lambda Extension" \
    --compatible-architectures "x86_64" \
    --zip-file  "fileb://extension.zip"
