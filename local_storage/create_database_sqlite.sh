#!/bin/sh

DATABASE=aggregator.db

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"

echo "Creating DB schema"
cat "${SCRIPT_DIR}/schema_sqlite.sql" | sqlite3 "${SCRIPT_DIR}/../${DATABASE}"

echo "Inserting test data"
cat "${SCRIPT_DIR}/test_data_sqlite.sql" | sqlite3 "${SCRIPT_DIR}/../${DATABASE}"
