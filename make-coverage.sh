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

TIMEOUT=2m
TIMEOUT_INTEGRATION=1440m


rm coverage.out 2>/dev/null

case $1 in
"unit-postgres")
    echo "Running unit tests with postgres..."
    go test -timeout $TIMEOUT -coverprofile=coverage.out ./... 1>&2
    ;;
"rest")
    echo rest # TODO: rest
    echo "Running REST API tests..."
    rm test.db 2> /dev/null
    rm ./insights-results-aggregator.test 2> /dev/null

    go test -timeout $TIMEOUT -c -v -tags testrunmain -coverpkg="./..." . 1>&2

    # would be better to put inside a subshell, but linter doesn't like it
    export INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE=./tests/tests
    (
        echo "Migrating the DB"
        go run . migrate latest
        echo "Starting a service"
        ./insights-results-aggregator.test -test.v -test.run "^TestRunMain$" -test.coverprofile=coverage.out 1>&2
    ) &

    # TODO: use curl in a loop to test when service is available
    sleep 2s
    ./test.sh --verbose --no-service 1>&2
    pkill --signal SIGINT -f insights-results-aggregator.test

    rm ./insights-results-aggregator.test 2>/dev/null

    ;;
"integration")
    echo "Running a service..."
    echo "Start your integration tests and press Ctrl+C when they are done"

    export INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE=./config-devel
    go test -timeout $TIMEOUT_INTEGRATION -v -tags testrunmain -run "^TestRunMain$" -coverprofile=coverage.out -coverpkg="./..." . 1>&2
    ;;
*)
    echo 'Please, choose "unit-postgres", "rest" or "integration"'
    echo "Aggregator's output will be redirected to stderr."
    echo "Coverage is saved to 'coverage.out' file"
    exit 1
    ;;
esac

go tool cover -func=coverage.out | grep -E "^total"
