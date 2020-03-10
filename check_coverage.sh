#!/usr/bin/env bash
# Copyright 2020 Red Hat, Inc
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


THRESHOLD=83
ERR_MESSAGE="Code coverage have to be at least $THRESHOLD%"

go_tool_cover_output=$(go tool cover -func=coverage.out)

echo "$go_tool_cover_output"

if (( $(echo "$go_tool_cover_output" | grep -i total | awk '{print $NF}' | grep -E "^[0-9]+" -o) >= THRESHOLD )); then
    exit 0
else
    echo -e "\\033[31m$ERR_MESSAGE\\e[0m"
    exit 1
fi
