build: clean
	go build -o ./extensions/newrelic-lambda-extension

clean:
	rm -rf extensions
	rm -f preview-extensions-ggqizro707
	rm -f extensions.zip

dist-x86_64: clean
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o ./extensions/newrelic-lambda-extension
	touch preview-extensions-ggqizro707

dist-arm64: clean
	env GOARCH=arm64 GOOS=linux go build -ldflags="-s -w" -o ./extensions/newrelic-lambda-extension
	touch preview-extensions-ggqizro707

zip-x86_64: dist-x86_64
	zip -r extensions.zip preview-extensions-ggqizro707 extensions

zip-arm64: dist-arm64
	zip -r extensions.zip preview-extensions-ggqizro707 extensions

publish: zip-x86_64
	aws lambda publish-layer-version --no-cli-pager --layer-name newrelic-lambda-extension --zip-file fileb://extensions.zip

test:
	go test -race ./...

coverage:
	./coverage.sh
