build: clean
	go build -o ./extensions/newrelic-lambda-extension

clean:
	rm -rf extensions

dist: clean
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o ./extensions/newrelic-lambda-extension

test:
	go test ./...
