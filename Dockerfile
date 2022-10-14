FROM golang:latest
WORKDIR /newrelic-lambda-extension
ENTRYPOINT [ "./localbuild.sh" ]
