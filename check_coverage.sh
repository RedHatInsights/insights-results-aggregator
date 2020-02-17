#!/usr/bin/env bash

THRESHOLD=55
ERR_MESSAGE="Code coverage have to be at least $THRESHOLD%"

go_tool_cover_output=$(go tool cover -func=coverage.out)

echo "$go_tool_cover_output"

if (( $(echo $go_tool_cover_output | grep -i total | awk '{print $NF}' | egrep "^[-1-9]+" -o) > $THRESHOLD )); then
    exit 0
else
    echo -e "\033[31m$ERR_MESSAGE\e[0m"
    exit 1
fi
