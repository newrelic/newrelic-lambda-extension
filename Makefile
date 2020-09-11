build: clean
	go build -o ./extensions/newrelic-lambda-extension

clean:
	rm -rf extensions
	rm -f preview-extensions-ggqizro707

dist: clean
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o ./extensions/newrelic-lambda-extension
	touch preview-extensions-ggqizro707

zip: dist
	zip extensions.zip preview-extensions-ggqizro707 extensions

publish: zip
	aws lambda publish-layer-version --region us-east-1 --layer-name newrelic-lambda-extension --zip-file fileb://extensions.zip

test:
	go test ./...
