#!/usr/bin/env bash

go test -run NO_TEST -bench=. ./...
exit $?
