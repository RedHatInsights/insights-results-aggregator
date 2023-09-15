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

# retrieve the latest tag set in repository
version=$(git describe --always --tags --abbrev=0)

buildtime=$(date)
branch=$(git rev-parse --abbrev-ref HEAD)
commit=$(git rev-parse HEAD)

utils_version=$(go list -m github.com/RedHatInsights/insights-operator-utils | awk '{print $2}')

go build "$@" -ldflags="-X 'main.BuildTime=$buildtime' -X 'main.BuildVersion=$version' -X 'main.BuildBranch=$branch' -X 'main.BuildCommit=$commit' -X 'main.UtilsVersion=$utils_version'"
exit $?
