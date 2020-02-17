#!/usr/bin/env bash

THRESHOLD=95

if (( $(go tool cover -func=coverage.out | grep -i total | awk '{print $NF}' | egrep "^[-1-9]+" -o) > $THRESHOLD )); then
	exit 0
else
	exit 1
fi
