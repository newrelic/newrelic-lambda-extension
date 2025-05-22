build: clean
	go build -o ./extensions/newrelic-lambda-extension

clean:
	rm -rf extensions
	rm -f preview-extensions-ggqizro707
	rm -f /tmp/newrelic-lambda-extension.x86_64.zip
	rm -f /tmp/newrelic-lambda-extension.arm64.zip

dist-x86_64: clean
	env GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -o ./extensions/newrelic-lambda-extension
	touch preview-extensions-ggqizro707

dist-arm64: clean
	env GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -o ./extensions/newrelic-lambda-extension
	touch preview-extensions-ggqizro707

zip-x86_64: dist-x86_64
	zip -r /tmp/newrelic-lambda-extension.x86_64.zip preview-extensions-ggqizro707 extensions

zip-arm64: dist-arm64
	zip -r /tmp/newrelic-lambda-extension.arm64.zip preview-extensions-ggqizro707 extensions

test:
	@echo "Normal tests"
	go test ./...
	@echo "\n\nRace check"
	go test -race ./...

coverage:
	./coverage.sh

publish: zip-x86_64
	aws lambda publish-layer-version --no-cli-pager --layer-name newrelic-lambda-extension-x86_64 --zip-file fileb:///tmp/newrelic-lambda-extension.x86_64.zip
