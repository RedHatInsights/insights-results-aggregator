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

COLORS_RED='\033[0;31m'
COLORS_RESET='\033[0m'
LOG_LEVEL="fatal"
VERBOSE_OUTPUT=false
NO_SERVICE=false

echo bash version is:
bash --version

if [[ $* == *verbose* ]] || [[ -n "${VERBOSE}" ]]; then
    # print all possible logs
    LOG_LEVEL=""
    VERBOSE_OUTPUT=true
fi

if [[ $* == *no-service* ]]; then
    NO_SERVICE=true
fi

function cleanup() {
    print_descendent_pids() {
        pids=$(pgrep -P "$1")
        echo "$pids"
        for pid in $pids; do
            print_descendent_pids "$pid"
        done
    }

    echo Exiting and killing all children...

    children=$(print_descendent_pids $$)

    # disable the message when you send stop signal to child processes
    set +m

    for pid in $(echo -en "$children"); do
        # nicely asking a process to commit suicide
        if ! kill -PIPE "$pid" &>/dev/null; then
            # we even gave them plenty of time to think
            sleep 2
        fi
    done

    # restore the message back since we want to know that process wasn't stopped correctory
    # set -m

    for pid in $(echo -en "$children"); do
        # murdering those who're alive
        kill -9 "$pid" &>/dev/null
    done

    sleep 1
}
trap cleanup EXIT

# check if file is locked (db is used by another process)
if fuser test.db &>/dev/null; then
    echo Database is locked, please kill or close the process using the file:
    fuser test.db
    exit 1
fi

go clean -testcache

if go build -race; then
    echo "Service build ok"
else
    echo "Build failed"
    exit 1
fi

function migrate_db_to_latest() {
    echo "Migrating DB to the latest migration version..."

    if INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE=./tests/tests \
        ./insights-results-aggregator migrate latest >/dev/null; then

        echo "Database migration was successful"
        return 0
    else
        echo "Unable to migrate DB"
        return 1
    fi
}

function populate_db_with_mock_data() {
    echo "Populating db with mock data..."

    if ./local_storage/populate_db_with_mock_data.sh; then
        echo "Database successfully populated with mock data"
        return 0
    else
        echo "Unable to populate db with mock data"
        return 1
    fi
}

function check_composer() {
    if command -v docker-compose > /dev/null; then
        COMPOSER=docker-compose
    elif command -v podman-compose > /dev/null; then
        COMPOSER=podman-compose
    else
        echo "Please, install docker-compose or podman-compose to run this tests"
        exit 1
    fi
}

function start_service() {
    if [ "$NO_SERVICE" = true ]; then
        echo "Not starting service"
        return
    fi

    echo "Starting a service"
    # TODO: stop parent(this script) if service died
    INSIGHTS_RESULTS_AGGREGATOR__LOGGING__LOG_LEVEL=$LOG_LEVEL \
        INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE=./tests/tests \
        INSIGHTS_RESULTS_AGGREGATOR__TESTS_DB=aggregator \
        INSIGHTS_RESULTS_AGGREGATOR__TESTS_DB_ADMIN_PASS=postgres \
        ./insights-results-aggregator ||
        echo -e "${COLORS_RED}service exited with error${COLORS_RESET}" &
    # shellcheck disable=2181
    if [ $? -ne 0 ]; then
        echo "Could not start the service"
        exit 1
    fi
}

function test_rest_api() {
    migrate_db_to_latest
    start_service
    sleep 2
    populate_db_with_mock_data
    sleep 1

    echo "Building REST API tests utility"
    if go build -o rest-api-tests tests/rest_api_tests.go; then
        echo "REST API tests build ok"
    else
        echo "Build failed"
        return 1
    fi
    sleep 1
    curl http://localhost:8080/api/v1/ || {
        echo -e "${COLORS_RED}server is not running(for some reason)${COLORS_RESET}"
        exit 1
    }

    OUTPUT=$(./rest-api-tests 2>&1)
    EXIT_CODE=$?

    if [ "$VERBOSE_OUTPUT" = true ]; then
        echo "$OUTPUT"
    else
        echo "$OUTPUT" | grep -v -E "^Pass "
    fi

    return $EXIT_CODE
}

function wait_for_postgres() {
    until psql "dbname=aggregator user=postgres password=postgres host=localhost sslmode=disable" -c '\q' > /dev/null 2>&1; do
         sleep 2
    done
}


echo -e "------------------------------------------------------------------------------------------------"

if [ -z "$CI" ]; then
    echo "Running postgres container locally"
    check_composer
    $COMPOSER up -d > /dev/null
    wait_for_postgres
fi

case $1 in
rest_api)
    test_rest_api
    EXIT_VALUE=$?
    ;;
*)
    # all tests
    # exit value will be 0 if every test returned 0
    EXIT_VALUE=0

    test_rest_api
    EXIT_VALUE=$((EXIT_VALUE + $?))

    ;;
esac

if [ -z "$CI" ]; then
    echo "Stopping postgres container"
    $COMPOSER down > /dev/null
fi

echo -e "------------------------------------------------------------------------------------------------"

exit $EXIT_VALUE
