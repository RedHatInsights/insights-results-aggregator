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


CONFIG_FILE=config-devel.toml
POSTGRES_PORT=$(cat $CONFIG_FILE | grep -i pg_port | grep -E -o '[0-9]+')

function run_unit_tests() {
    go test -coverprofile coverage.out $(go list ./... | grep -v tests)
    if [ $? -ne 0 ]; then
        echo "unit tests failed"
        exit 1
    fi
}

echo "running unit tests with sqlite in memory"
# tests with sqlite
run_unit_tests

if netstat -nltp 2>/dev/null | grep $POSTGRES_PORT >/dev/null; then
    # tests with postgres
    export INSIGHTS_RESULTS_AGGREGATOR__TESTS_DB="postgres"
    export INSIGHTS_RESULTS_AGGREGATOR__TESTS_DB_ADMIN_PASS="admin"
    echo "running unit tests with postgres on port $POSTGRES_PORT"
    run_unit_tests
else
    echo "postgres on port $POSTGRES_PORT is not running, skipping unit tests with postgres"
fi
