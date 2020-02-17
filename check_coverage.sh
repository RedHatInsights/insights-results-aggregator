#!/usr/bin/env bash

THRESHOLD=95
ERR_MESSAGE="Code coverage have to be at least $THRESHOLD%"

if (( $(go tool cover -func=coverage.out | grep -i total | awk '{print $NF}' | egrep "^[-1-9]+" -o) > $THRESHOLD )); then
    exit 0
else
    echo -e "\033[31m$ERR_MESSAGE\e[0m"
    exit 1
fi
