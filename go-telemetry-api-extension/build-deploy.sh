if [ ! -d "bin" ]; then
    mkdir bin
fi
if [ ! -d "bin/extensions" ]; then
    mkdir bin/extensions
fi
GOOS=linux GOARCH=amd64 go build -o bin/extensions/go-telemetry-api-extension main.go
chmod +x bin/extensions/go-telemetry-api-extension
cd bin
zip -r extension.zip extensions/
aws lambda publish-layer-version \
    --layer-name "go-telemetry-api-extension" \
    --zip-file  "fileb://extension.zip"
