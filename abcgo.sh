#!/bin/bash
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

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

go get -u github.com/droptheplot/abcgo

echo -e "${BLUE}ABC metric${NC}"
abcgo -path .

threshold=40
echo -e "${BLUE}Functions with ABC metrics greater than ${threshold}${NC}:"

if [[ $(abcgo -path . -sort -format raw | awk "\$4>${threshold}" | tee /dev/tty | wc -l) -ne 0 ]]
then
    echo -e "${RED}Functions with too high ABC metrics detected!${NC}"
    exit 1
else
    echo -e "${GREEN}ABC metrics is ok for all functions in all packages${NC}"
    exit 0
fi

