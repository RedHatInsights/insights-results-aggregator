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

        gitUtils.stageWithContext("Unit-tests", shortenURL = false) {
            unitTestsStatus = sh(script: "make test-postgres", returnStatus: true)
            withEnv(["TERM=xterm"]){
                sh "./check_coverage.sh"
            }
            withCredentials([string(credentialsId: "ira-codecov", variable: "CODECOV_TOKEN")]) {
                withEnv(["TERM=xterm"]){
                    sh "curl -s https://codecov.io/bash | bash"
                }
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
