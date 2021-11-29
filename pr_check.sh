#!/bin/bash
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


# --------------------------------------------
# Options that must be configured by app owner
# --------------------------------------------
export APP_NAME="ccx-data-pipeline"  # name of app-sre "application" folder this component lives in
export COMPONENT_NAME="ccx-insights-results"  # name of app-sre "resourceTemplate" in deploy.yaml for this component
export IMAGE="quay.io/cloudservices/insights-results-aggregator"

export IQE_PLUGINS="ccx"
export IQE_MARKER_EXPRESSION="smoke" # ccx_data_pipeline_smoke does not exits (at least yet) as marker in the plugin
export IQE_FILTER_EXPRESSION=""
export IQE_CJI_TIMEOUT="30m"


# Install bonfire repo/initialize
CICD_URL=https://raw.githubusercontent.com/RedHatInsights/bonfire/master/cicd
curl -s $CICD_URL/bootstrap.sh > .cicd_bootstrap.sh && source .cicd_bootstrap.sh

# Build the image and push to quay
source $CICD_ROOT/build.sh

# Run the unit tests with an ephemeral db
# source $APP_ROOT/unit_test.sh

# Deploy rbac to an ephemeral namespace for testing
source $CICD_ROOT/deploy_ephemeral_env.sh

# Run smoke tests with ClowdJobInvocation
source $CICD_ROOT/cji_smoke_test.sh
