#!/usr/bin/env bash

# Copyright 2021 Red Hat, Inc
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

# Script to generate documentation with DB schema description

function cleanUpAndExit {
    if [[ $DB_KEEP_RUNNING ==  0 ]]; then
        podman stop $DB_POD_NAME
    fi
    exit $1
}

# Settings for database to be documented
# (can be taken from config.toml, config-devel.toml etc.)
DB_ADDRESS=localhost
DB_LOGIN=postgres
DB_PASSWORD=postgres
DB_NAME=aggregator

DB_POD_NAME=postgres_aggregator
DB_KEEP_RUNNING=1

# Generate the documentation
OUTPUT_DIR=docs/db-description-3

# Launch local postgres from scratch
podman run --name=$DB_POD_NAME --rm --network=host -e POSTGRES_PASSWORD=$DB_PASSWORD -e POSTGRES_USER=$DB_LOGIN -e POSTGRES_DB=$DB_NAME -d postgres:10

if [[ $? != 0 ]]; then
    echo "Cannot launch postgress database locally"
    cleanUpAndExit 1
fi

# Run migration to latest
make  # compiles aggregator
if [[ $? != 0 ]]; then
    echo "Cannot build aggregator"
    cleanUpAndExit 2
fi

sleep 5
INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE=./config-devel.toml ./insights-results-aggregator migration latest

if [[ $? != 0 ]]; then
    echo "Failure generating latest migration"
    cleanUpAndExit 3
fi

java -jar schemaspy-6.1.0.jar -cp . -t pgsql -u ${DB_LOGIN} -p ${DB_PASSWORD} -host ${DB_ADDRESS} -s public -o ${OUTPUT_DIR} -db ${DB_NAME} -dp postgresql-42.2.20.jre7.jar

cleanUpAndExit $?