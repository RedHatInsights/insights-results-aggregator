@Library("github.com/RedHatInsights/insights-pipeline-lib@v3") _

node {
    pipelineUtils.cancelPriorBuilds()

    pipelineUtils.runIfMasterOrPullReq {
        runStages()
    }
}

def getBaseCommit() {
    def baseCommit = ''
    def latestCommit = sh(label: 'Get previous commit', script: "git rev-parse HEAD", returnStdout: true)?.trim()
    def previousCommit = sh(label: 'Get previous commit', script: "git rev-parse HEAD^", returnStdout: true)?.trim()
    if (env?.CHANGE_ID == null) {
        baseCommit = env.GIT_COMMIT
    } else if("${env.GIT_COMMIT}".equals("${latestCommit}")) {
        baseCommit = env.GIT_COMMIT
    } else {
        baseCommit = previousCommit
    }
    env.GIT_BASE_COMMIT = baseCommit
    return baseCommit
}

def runStages() {

    openShiftUtils.withNode(yaml: "jenkins_slave_pod_template.yaml") {
        checkout scm
        
        gitUtils.stageWithContext("Style", shortenURL = false) {

            styleStatus = sh(script: "make fmt vet lint cyclo shellcheck errcheck goconst gosec ineffassign abcgo", returnStatus: true)

            if (styleStatus != 0) {
                error("Style check failed")
            }
        }
    }
    
    openShiftUtils.withNode(image: "registry.access.redhat.com/rhscl/python-36-rhel7") {
        checkout scm

        gitUtils.stageWithContext("JSON", shortenURL = false) {

            jsonCheckStatus = sh(script: "make json-check", returnStatus: true)

            if (jsonCheckStatus != 0) {
                error("Json check failed")
            }
        }
    }

    openShiftUtils.withNode(image: "openapitools/openapi-generator-cli") {
        checkout scm

        gitUtils.stageWithContext("OpenAPI", shortenURL = false) {

            openapiCheckStatus = sh(script: "docker-entrypoint.sh validate -i openapi.json", returnStatus: true)

            if (openapiCheckStatus != 0) {
                error("OpenAPI check failed")
            }
        }
    }

    openShiftUtils.withNode(yaml: "jenkins_slave_pod_template.yaml") {
        checkout scm

        gitUtils.stageWithContext("Unit-tests", shortenURL = false) {
            unitTestsStatus = sh(script: "make test-postgres", returnStatus: true)
            withEnv(["TERM=xterm"]){
                sh "./check_coverage.sh"
            }
            withCredentials([string(credentialsId: "ira-codecov", variable: "CODECOV_TOKEN")]) {
                sh "curl -s https://codecov.io/bash | bash -s -- -C ${getBaseCommit()}"
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
