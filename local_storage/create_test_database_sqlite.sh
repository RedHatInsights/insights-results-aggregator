#!/usr/bin/env bash

DATABASE=test.db

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"

cat "${SCRIPT_DIR}/schema_sqlite.sql" | sqlite3 "${SCRIPT_DIR}/../${DATABASE}"
cat "${SCRIPT_DIR}/test_data_sqlite.sql" | sqlite3 "${SCRIPT_DIR}/../${DATABASE}"
