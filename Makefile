build: clean
	go build -o ./extensions/newrelic-lambda-extension

clean:
	rm -rf extensions
	rm -f preview-extensions-ggqizro707
	rm -f extensions.zip

dist: clean
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o ./extensions/newrelic-lambda-extension
	touch preview-extensions-ggqizro707

zip: dist
	zip -r extensions.zip preview-extensions-ggqizro707 extensions

publish: zip
	aws lambda publish-layer-version --no-cli-pager --layer-name newrelic-lambda-extension --zip-file fileb://extensions.zip

test:
	go test ./...

coverage:
	./coverage.sh
