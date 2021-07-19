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

# Settings for database to be documented
# (can be taken from config.toml, config-devel.toml etc.)
DB_ADDRESS=localhost
DB_LOGIN=postgres
DB_PASSWORD=postgres
DB_NAME=aggregator

# Generate the documentation
OUTPUT_DIR=docs/db-description-3

java -jar schemaspy-6.1.0.jar -cp . -t pgsql -u ${DB_LOGIN} -p ${DB_PASSWORD} -host ${DB_ADDRESS} -s public -o ${OUTPUT_DIR} -db ${DB_NAME} -dp postgresql-42.2.20.jre7.jar
