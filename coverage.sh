#!/usr/bin/env bash

set -e
echo "" > coverage.txt

go test -coverprofile=profile.out -covermode=atomic main.go
if [ -f profile.out ]; then
    cat profile.out >> coverage.txt
    rm profile.out
fi

for d in $(go list ./... | grep -v vendor); do
    go test -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done
