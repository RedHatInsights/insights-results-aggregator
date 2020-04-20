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

function cleanup()
{
	print_descendent_pids() {
		pids=$(pgrep -P "$1")
		echo "$pids"
		for pid in $pids; do
			print_descendent_pids "$pid"
		done
	}

	echo Exiting and killing all children...
	for pid in $(print_descendent_pids $$); do
		if ! kill "$pid" &> /dev/null; then
			# wait for it to stop correctly
			sleep 1
			kill -9 "$pid" &> /dev/null
		fi
	done
	sleep 1
}
trap cleanup EXIT

# check if file is locked (db is used by another process)
if fuser test.db &> /dev/null; then
	echo Database is locked, please kill or close the process using the file:
	fuser test.db
	exit 1
fi


go clean -testcache

if go build -race
then
	echo "Service build ok"
else
	echo "Build failed"
	exit 1
fi

function populate_db_with_mock_data() {
	echo "Populating db with mock data..."

	if ./local_storage/populate_db_with_mock_data.sh
	then
		echo "Done"
		return 0
	else
		echo "Unable to populate db with mock data"
		return 1
	fi
}

function start_service() {
	echo "Starting a service"
	# TODO: stop parent(this script) if service died
	INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE=./tests/tests ./insights-results-aggregator || \
		echo -e "${COLORS_RED}service exited with error${COLORS_RESET}" &
	# shellcheck disable=2181
	if [ $? -ne 0 ]
	then
		echo "Could not start the service"
		exit 1
	fi
}

function test_rest_api() {
	start_service
	# Retry populating the database with mock data N times.
        # Wait 2 seconds between the retry starts to settle down
	# Wait 1 second between attempts.
	# Without this, the DB could be locked because
	# the migrations have not yet finished.
	sleep 2
	for i in {1..5}
	do
		populate_db_with_mock_data 2> /dev/null && break || sleep 1
	done

	echo "Building REST API tests utility"
	if go build -o rest-api-tests tests/rest_api_tests.go
	then
		echo "REST API tests build ok"
	else
		echo "Build failed"
		return 1
	fi
	sleep 1
	curl http://localhost:8080/api/v1/ || {
		echo -e "${COLORS_RED}server is not running(for some reason)${COLORS_RESET}"; exit 1
	}
	./rest-api-tests
	return $?
}

function test_openapi() {
	echo "Testing OpenAPI specifications file"
	# shellcheck disable=2181
	docker run --rm -v "${PWD}":/local/:Z openapitools/openapi-generator-cli validate -i ./local/openapi.json

	if [ $? -eq 0 ]
	then
		echo "OpenAPI spec file is OK"
	else
		echo "OpenAPI spec file validation failed"
		exit 1
	fi
}

echo -e "------------------------------------------------------------------------------------------------"

case $1 in
	rest_api)
		test_rest_api
		EXIT_VALUE=$?
		;;
	openapi)
		test_openapi
		EXIT_VALUE=$?
		;;
	*)
		# all tests
		# exit value will be 0 if every test returned 0
		EXIT_VALUE=0

		test_rest_api
		EXIT_VALUE=$((EXIT_VALUE + $?))

		test_openapi
		EXIT_VALUE=$((EXIT_VALUE + $?))
		;;
esac

echo -e "------------------------------------------------------------------------------------------------"

exit $EXIT_VALUE
