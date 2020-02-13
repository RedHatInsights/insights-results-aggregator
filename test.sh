#!/usr/bin/env bash

export TEST_KAFKA_ADDRESS=localhost:9092

function cleanup()
{
	echo Exiting and killing all children...
	pkill -P $$
}
trap cleanup EXIT

# check if file is locked (db is used by another process)
if fuser test.db &> /dev/null; then
        echo Database is locked, please kill or close the process using the file:
        fuser test.db
        exit 1
fi

go build -race

if [ $? -eq 0 ]
then
    echo "Service build ok"
else
    echo "Build failed"
    exit 1
fi

echo "Creating test database"
rm -f test.db
./local_storage/create_test_database_sqlite.sh
if [ $? -eq 0 ]
then
    echo "Done"
else
    echo "Creating DB failed"
    exit 1
fi

echo "Starting a service"
./insights-results-aggregator &

if [ $? -ne 0 ]; then
	echo "Could not start the service"
	exit 1
fi

function test_metrics() {
	go test ./tests/metrics
	return $?
}
function test_rest_api() {
	# TODO: complete
	return 0
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
	*)
		# all tests
		# exit value will be 0 if every test returned 0
		EXIT_VALUE=0

		test_metrics
		EXIT_VALUE=$(($EXIT_VALUE + $?))
		test_rest_api
		EXIT_VALUE=$(($EXIT_VALUE + $?))
		;;
esac

echo -e "------------------------------------------------------------------------------------------------"

exit $EXIT_VALUE
