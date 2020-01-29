#!/bin/bash

cd "$(dirname $0)"

go get golang.org/x/lint/golint

if golint `go list ./...` |
    grep -v ALL_CAPS |
    grep .; then
  exit 1
fi
