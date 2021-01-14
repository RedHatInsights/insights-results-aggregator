#!/bin/bash

# Copyright 2020, 2021  Red Hat, Inc
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

BLUE=$(tput setaf 4)
RED_BG=$(tput setab 1)
GREEN_BG=$(tput setab 2)
NC=$(tput sgr0) # No Color

echo -e "${BLUE}Detecting ineffectual assignments in Go code${NC}"

if ! [ -x "$(command -v ineffassign)" ]
then
    echo -e "${BLUE}Installing ineffassign${NC}"
    GO111MODULE=off go get github.com/gordonklaus/ineffassign
fi

if ! ineffassign ./...
then
    echo -e "${RED_BG}[FAIL]${NC} Code with ineffectual assignments detected"
    exit 1
else
    echo -e "${GREEN_BG}[OK]${NC} No ineffectual assignments has been detected"
    exit 0
fi
