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

# Updates the ./tests/content/ok/ directory with the latest rules to test against

function clean_up() {
    rm -rf "$CLONE_TEMP_DIR"
}
trap clean_up EXIT

RULES_REPO="https://gitlab.cee.redhat.com/ccx/ccx-rules-ocp.git"
CONTENT_DIR="$( cd "$(dirname "$0")" && pwd )/tests/content"
OK_RULES_DIR="${CONTENT_DIR}/ok"

CLONE_TEMP_DIR="${CONTENT_DIR}/tmp"
EXTERNAL_RULES_CONTENT="${CLONE_TEMP_DIR}/content/external/rules/"
RULES_CONFIG="${CLONE_TEMP_DIR}/content/config.yaml"

echo "Attempting to clone repository into ${CLONE_TEMP_DIR}"

git clone "${RULES_REPO}" "${CLONE_TEMP_DIR}"
if [ $? -ne 0 ]
then
    echo "Couldn't clone rules repository"
    exit 1
fi

rm -rf "$OK_RULES_DIR"

if [ $? -ne 0 ]
then
    echo "Couldn't remove previous content"
    exit 1
fi

mv "${EXTERNAL_RULES_CONTENT}" "${OK_RULES_DIR}" && mv "${RULES_CONFIG}" "${OK_RULES_DIR}"

if [ $? -ne 0 ]
then
    echo "Couldn't move rules content from cloned repository"
    exit 1
fi

rm -rf "${CLONE_TEMP_DIR}"

echo "tests/content/ok/ updated with latest rules"
exit 0
