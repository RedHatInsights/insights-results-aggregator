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

STORAGE=$1

function run_unit_tests() {
    # shellcheck disable=SC2046
    if ! go test -coverprofile coverage.out $(go list ./... | grep -v tests | tr '\n' ' ')
    then
        echo "unit tests failed"
        exit 1
    fi
}

if [ -z "$STORAGE" ] ; then
    # tests with sqlite
    echo "running unit tests with sqlite in memory"
    # shellcheck disable=SC2034
    CONFIG_FILE=config.toml
    run_unit_tests
fi

if [ "$STORAGE" == "postgres" ]; then
    # tests with postgres
    echo "running unit tests with postgres"
    # shellcheck disable=SC2034
    CONFIG_FILE=config-devel.toml
    run_unit_tests
fi
