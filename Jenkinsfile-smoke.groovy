/*
 * Requires: https://github.com/RedHatInsights/insights-pipeline-lib
 */

@Library("github.com/RedHatInsights/insights-pipeline-lib@v3") _

if (env.CHANGE_TARGET == "stable" && env.CHANGE_ID) {
    execSmokeTest (
        ocDeployerBuilderPath: "ccx-data-pipeline/ccx-data-pipeline",
        ocDeployerComponentPath: "ccx-data-pipeline/ccx-data-pipeline",
        ocDeployerServiceSets: "ccx-data-pipeline,ingress,buck-it,payload-tracker,engine,platform-mq",
        iqePlugins: ["iqe-ccx-plugin"],
        pytestMarker: ["ccx_smoke"]
    )
}
