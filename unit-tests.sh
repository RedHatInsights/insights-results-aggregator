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
    if [ -z "$TEST_TO_RUN" ]; then
        echo "No specific tests given. Running all available tests."
        run_cmd="$(go list ./... | grep -v tests | tr '\n' ' ')"
    else
        echo "Running specific test $TEST_TO_RUN"
        run_cmd="$TEST_TO_RUN"
    fi
    # shellcheck disable=SC2046
    if ! go test -timeout 10m -coverprofile coverage.out $run_cmd
    then
        echo "unit tests failed"
        exit 1
    fi
}

function check_composer() {
    if command -v docker > /dev/null && docker compose version > /dev/null 2>&1; then
        COMPOSER="docker compose"
    elif command -v docker-compose > /dev/null; then
        COMPOSER=docker-compose
    elif command -v podman-compose > /dev/null; then
        COMPOSER=podman-compose
    else
        echo "Please, install docker-compose or podman-compose to run this tests"
        exit 1
    fi
}


function wait_for_postgres() {
    echo -n "Waiting for postgres to be available"
    until psql "dbname=aggregator user=postgres password=postgres host=localhost sslmode=disable" -c '\q' 2> /dev/null; do
         echo -n "."
         sleep 1
    done

    echo
    echo "Postgres already available. Continuing"
}

if [ -z "$CI" ]; then
    echo "Running postgres container locally"
    check_composer
    echo \"$COMPOSER\" to be used as composer
    $COMPOSER up -d > /dev/null
    wait_for_postgres
fi

path_to_config=$(pwd)/config-devel.toml
export INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE="$path_to_config"
run_unit_tests

if [ -z "$CI" ]; then
    echo "Stopping postgres container"
    $COMPOSER down > /dev/null
fi
