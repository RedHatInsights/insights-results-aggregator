#!/usr/bin/env bash

export TEST_KAFKA_ADDRESS=localhost:9093

COLORS_RED='\033[0;31m'
COLORS_RESET='\033[0m'

function cleanup()
{
	print_descendent_pids() {
		pids=$(pgrep -P $1)
		echo $pids
		for pid in $pids; do
			print_descendent_pids $pid
		done
	}

	echo Exiting and killing all children...
	for pid in $(print_descendent_pids $$); do
		kill -9 $pid 2> /dev/null
	done
}
trap cleanup EXIT

# check if file is locked (db is used by another process)
if fuser test.db &> /dev/null; then
	echo Database is locked, please kill or close the process using the file:
	fuser test.db
	exit 1
fi

go clean -testcache

go build -race

if [ $? -eq 0 ]
then
    echo "Service build ok"
else
    echo "Build failed"
    exit 1
fi

rm -f test.db
db=$(printenv DATABASE)
echo "Creating test database. DB engine is $db"
case $db in
	postgresql)
		cd ./local_storage/
		./dockerize_postgres.sh
		# TODO: doesn't work without sleep, fix it
		# probably we need to wait for postgres to start
		sleep 2
		cd ../
		;;
	*)
		./local_storage/create_test_database_sqlite.sh
		export TEST_DB_DRIVER="sqlite3"
		;;
esac
if [ $? -eq 0 ]
then
	echo "Done"
else
	echo "Creating DB failed"
	exit 1
fi

function start_kafka() {
	cd ./local_storage/
	./dockerize_kafka.sh || {
		echo -e "${COLORS_RED}could not start kafka${COLORS_RESET}"
		exit 1
	}
	cd ../
	sleep 2
}

function start_service() {
	echo "Starting a service"
	case $db in
		postgresql)
			INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE=./tests/tests_postgresql \
				./insights-results-aggregator || \
				echo -e "${COLORS_RED}service exited with error${COLORS_RESET}" &
			;;
		*)
			INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE=./tests/tests_sqlite \
				./insights-results-aggregator || \
				echo -e "${COLORS_RED}service exited with error${COLORS_RESET}" &
			;;
	esac
	if [ $? -ne 0 ]; then
		echo "Could not start the service"
		exit 1
	fi
}

function test_metrics() {
	go test ./tests/metrics
	return $?
}
function test_rest_api() {
	start_service
	echo "Building REST API tests utility"
	go build -o rest-api-tests tests/rest_api_tests.go
	if [ $? -eq 0 ]
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
function test_message_processing() {
	start_kafka
	start_service
	go test ./tests/consumer -v
	return $?
}

echo -e "------------------------------------------------------------------------------------------------"

case $1 in
	metrics)
		test_metrics
		EXIT_VALUE=$?
		;;
	rest_api)
		test_rest_api
		EXIT_VALUE=$?
		;;
	message_processing)
		test_message_processing
		EXIT_VALUE=$?
		;;
	*)
		# all tests
		# exit value will be 0 if every test returned 0
		EXIT_VALUE=0

		test_metrics
		EXIT_VALUE=$(($EXIT_VALUE + $?))
		test_rest_api
		EXIT_VALUE=$(($EXIT_VALUE + $?))
		test_message_processing
		EXIT_VALUE=$(($EXIT_VALUE + $?))
		;;
esac

echo -e "------------------------------------------------------------------------------------------------"

exit $EXIT_VALUE
