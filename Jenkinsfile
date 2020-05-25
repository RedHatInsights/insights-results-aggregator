@Library("github.com/RedHatInsights/insights-pipeline-lib@v3") _

node {
    pipelineUtils.cancelPriorBuilds()

    pipelineUtils.runIfMasterOrPullReq {
        runStages()
    }
}

def runStages() {

    openShiftUtils.withNode(yaml: "jenkins_slave_pod_template.yaml") {
        checkout scm

        gitUtils.stageWithContext("Style", shortenURL = false) {
            styleStatus = sh(script: "make style", returnStatus: true)
        }

        if (styleStatus != 0) {
            error("Style check failed")
        }

        gitUtils.stageWithContext("Unit-tests", shortenURL = false) {
            unitTestsStatus = sh(script: "make test", returnStatus: true)
            sh "./check_coverage.sh"
            withCredentials([string(credentialsId: "ira-codecov", variable: "CODECOV_TOKEN")]) {
                sh "bash <(curl -s https://codecov.io/bash)"
            }
        }

        if (unitTestsStatus != 0) {
            error("Unit tests failed")
        }

        gitUtils.stageWithContext("Integration-tests", shortenURL = false) {
            unitTestsStatus = sh(script: "make integration_tests", returnStatus: true)
        }

        if (unitTestsStatus != 0) {
            error("Integration tests failed")
        }
    }
}