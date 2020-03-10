// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
 * Requires: https://github.com/RedHatInsights/insights-pipeline-lib
 */

@Library("github.com/RedHatInsights/insights-pipeline-lib@v3") _

if (env.CHANGE_TARGET == "stable" && env.CHANGE_ID) {
    execSmokeTest (
        ocDeployerBuilderPath: "ccx-data-pipeline/insights-results-aggregator",
        ocDeployerComponentPath: "ccx-data-pipeline/insights-results-aggregator",
        ocDeployerServiceSets: "ccx-data-pipeline,ingress,buck-it,engine,platform-mq",
        iqePlugins: ["iqe-ccx-plugin"],
        pytestMarker: ["smoke"]
    )
}
