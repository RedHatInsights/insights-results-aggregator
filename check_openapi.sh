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

echo "Testing OpenAPI specifications file"
# shellcheck disable=2181

if docker run --rm -v "${PWD}":/local/:Z openapitools/openapi-generator-cli validate -i ./local/openapi.json; then
    echo "OpenAPI spec file is OK"
else
    echo "OpenAPI spec file validation failed"
    exit 1
fi
