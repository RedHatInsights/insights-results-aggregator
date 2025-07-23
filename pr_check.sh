#!/bin/bash
# Copyright 2022 Red Hat, Inc
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

# shellcheck disable=SC2317
set -exv


# --------------------------------------------
# Options that must be configured by app owner
# --------------------------------------------
APP_NAME="ccx-data-pipeline"  # name of app-sre "application" folder this component lives in
REF_ENV="insights-production"
# NOTE: insights-results-aggregator contains deployment for multiple services
#       for pull requests we need latest git PR version of these components to be
#       deployed to ephemeral env and overriding resource template --set-template-ref.
#       Using multiple components name in COMPONENT_NAME forces bonfire to use the
#       git version of clowdapp.yaml(or any other) file from the pull request.
COMPONENT_NAME="ccx-insights-results ccx-redis dvo-writer"  # name of app-sre "resourceTemplate" in deploy.yaml for this component
IMAGE="quay.io/cloudservices/insights-results-aggregator"
COMPONENTS="ccx-data-pipeline ccx-insights-results valkey-writer dvo-writer dvo-extractor insights-content-service ccx-smart-proxy ccx-mock-ams ccx-upgrades-rhobs-mock" # space-separated list of components to load
COMPONENTS_W_RESOURCES=""  # component to keep
CACHE_FROM_LATEST_IMAGE="true"
DEPLOY_FRONTENDS="false"
# Set the correct images for pull requests.
# pr_check in pull requests still uses the old cloudservices images
EXTRA_DEPLOY_ARGS="\
    --set-parameter ccx-insights-results/IMAGE=quay.io/cloudservices/insights-results-aggregator \
    --set-parameter ccx-redis/IMAGE=quay.io/cloudservices/insights-results-aggregator \
    --set-parameter dvo-writer/IMAGE=quay.io/cloudservices/insights-results-aggregator
"

export IQE_PLUGINS="ccx"
# Run all pipeline tests
export IQE_MARKER_EXPRESSION="pipeline"
export IQE_FILTER_EXPRESSION="not test_rbac"
export IQE_REQUIREMENTS_PRIORITY=""
export IQE_TEST_IMPORTANCE=""
export IQE_CJI_TIMEOUT="30m"
export IQE_SELENIUM="false"
export IQE_ENV="ephemeral"
export IQE_ENV_VARS="DYNACONF_USER_PROVIDER__rbac_enabled=false"

# NOTE: Uncomment to skip pull request integration tests and comment out
#       the rest of the file.
# mkdir artifacts
# echo '<?xml version="1.0" encoding="utf-8"?><testsuites><testsuite name="pytest" errors="0" failures="0" skipped="0" tests="1" time="0.014" timestamp="2021-05-13T07:54:11.934144" hostname="thinkpad-t480s"><testcase classname="test" name="test_stub" time="0.000" /></testsuite></testsuites>' > artifacts/junit-stub.xml

function build_image() {
   source $CICD_ROOT/build.sh
}

function deploy_ephemeral() {
   source $CICD_ROOT/deploy_ephemeral_env.sh
}

function run_smoke_tests() {
   # Workaround: cji_smoke_test.sh requires only one component name. Fallback to only one component name.
   export COMPONENT_NAME="ccx-insights-results"
   source $CICD_ROOT/cji_smoke_test.sh
   source $CICD_ROOT/post_test_results.sh  # publish results in Ibutsu
}


# Install bonfire repo/initialize
CICD_URL=https://raw.githubusercontent.com/RedHatInsights/bonfire/master/cicd
curl -s $CICD_URL/bootstrap.sh > .cicd_bootstrap.sh && source .cicd_bootstrap.sh
echo "creating PR image"
build_image

echo "deploying to ephemeral"
deploy_ephemeral

echo "PR smoke tests disabled"
run_smoke_tests
