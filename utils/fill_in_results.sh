#!/bin/bash
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

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

WORKDIR="work"


# Check if Insights OCP is installed and properly initialized.
# Please look into README.md from ccx-rules-ocp repository how to setup
# Python virtual environment in order to be able to run insights-run command.
function check_insights_availability() {
    insights-run --help > /dev/null 2>&1 || { echo >&2 -e "${RED}Insights OCP needs to be installed and initialized. Aborting.${NC}"; exit 1; }
    echo -e "${GREEN}Insights OCP seems to be initialized properly${NC}"
}


# Decompress input file taken from Kraken or any other external source
function decompress_input() {
    echo -e "${BLUE}Decompressing input file $1${NC}"
    tar xvfz "$1"
    echo -e "${GREEN}Done${NC}"
}


# Input tarball has very strange directory structure (ask other teams why on Earth do they need it)
# We just move all files from this structure into one "flat" workdir
function move_input_to_workdir() {
    find . -name "*.tar.gz" -exec mv -i -- {} "$1" \;
}


# Remove now empty temporary directories
function cleanup_temp_directories() {
    # shellcheck disable=SC2045,SC2035
    for temp_directory in $(ls -1 -d */)
    do
        if [ "$temp_directory" != "$1/" ]
        then
            echo "Removing ${temp_directory}"
            rm -rf "${temp_directory}"
        else
            echo -e "${BLUE}Skipping ${temp_directory}${NC}"
        fi
    done
}

# Apply all OCP rules for all input files (gzipped tarballs)
# Output is formatted to be human readable
function apply_ocp_rules() {
    cwd=$(pwd)
    cd "$1" || exit

    echo -e "${BLUE}Starting applying OCP rules${NC}"
    for input in *.tar.gz
    do
        echo "$input"
        insights-run -p ccx_rules_ocp -f json "${input}" > x.json
        python -m json.tool < x.json > "$input.json"
    done
    echo -e "${GREEN}Done${NC}"

    # leftover
    rm x.json

    cd "${cwd}" || exit
}


# Clean tarballs after OCP rules are applied
function clean_tarballs() {
    rm "$1"/*.tar.gz
}


# Anonymize all OCP rules results
function anonymize() {
    cwd=$(pwd)
    cd "$1" || exit

    echo -e "${BLUE}Anonymizing OCP rules results${NC}"
    python3 "${cwd}"/anonymize.py
    echo -e "${GREEN}Done${NC}"

    cd "${cwd}" || exit
}


# Convert OCP rules results into JSONs compatible with aggregator
function convert_to_report() {
    cwd="$PWD"
    cd "$1" || exit

    echo -e "${BLUE}Converting OCP rules results into JSONs compatible with aggregator${NC}"
    python3 "${cwd}"/2report.py "$2" "$3"
    echo -e "${GREEN}Done${NC}"

    cd "${cwd}" || exit
}


# Perform cleanup after all previous operations
function cleanup_workdir() {
    echo -e "${BLUE}Cleanup${NC}"

    rm -f "$1"/*.tar.gz.json
    rm -f "$1"/s_*.json

    echo -e "${GREEN}Done${NC}"
}


# check parameters first
if [ $# -lt 3 ]
then
    echo -e "${BLUE}Usage:${NC}"
    echo "    fill_in_results.sh archive.tar.bz org_id cluster_name"
    echo -e "${BLUE}Example:${NC}"
    echo "    fill_in_results.sh external-rules-archives-2020-03-31.tar 11789772 5d5892d3-1f74-4ccf-91af-548dfc9767aa"
    exit 1
fi

# $1 - tarball with Insights operator results taken from external sources
# $2 - organization ID
# $3 - cluster name

check_insights_availability
decompress_input "$1"
mkdir -p ${WORKDIR}
move_input_to_workdir ${WORKDIR}
cleanup_temp_directories ${WORKDIR}
apply_ocp_rules ${WORKDIR}
clean_tarballs ${WORKDIR}
anonymize ${WORKDIR}
convert_to_report ${WORKDIR} "$2" "$3"
cleanup_workdir ${WORKDIR}
