build: clean
	go build -o ./extensions/newrelic-lambda-extension

build-arm64: clean
	mkdir extensions
	env GOARCH=arm64 GOOS=linux go build -ldflags="-s -w" -o ./extensions/newrelic-lambda-extension
	chmod +x ./extensions/newrelic-lambda-extension
	zip -r ./extensions/extension.zip ./extensions/

build-x86_64: clean
	mkdir extensions
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o ./extensions/newrelic-lambda-extension
	chmod +x ./extensions/newrelic-lambda-extension
	zip -r ./extensions/extension.zip ./extensions/

clean:
	rm -rf extensions

ci-build-x86_64: clean
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o ./extensions/newrelic-lambda-extension

ci-build-arm64: clean
	env GOARCH=arm64 GOOS=linux go build -ldflags="-s -w" -o ./extensions/newrelic-lambda-extension

zip-x86_64: ci-build-x86_64
	zip -r /tmp/newrelic-lambda-extension.x86_64.zip extensions

zip-arm64: dist-arm64
	zip -r /tmp/newrelic-lambda-extension.arm64.zip extensions

test:
	@echo "Normal tests"
	go test ./...
	@echo "\n\nRace check"
	go test -race ./...

coverage:
	./coverage.sh
