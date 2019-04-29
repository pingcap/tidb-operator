def call(BUILD_BRANCH, RELEASE_TAG) {

    env.GOPATH = "/go"
    env.GOROOT = "/usr/local/go"
    env.PATH = "${env.GOROOT}/bin:${env.GOPATH}/bin:/bin:${env.PATH}:/home/jenkins/bin"

    def GITHASH
    def TKCTL_CLI_PACKAGE = "tkctl-${GOOS}-${GOARCH}-${RELEASE_TAG}"

    catchError {
        node('k8s_centos7_build') {
            def WORKSPACE = pwd()

            dir("${WORKSPACE}/operator"){
                stage('Build and release CLI to qiniu'){
                    checkout([$class: 'GitSCM', branches: [[name: "${BUILD_BRANCH}"]], userRemoteConfigs:[[url: "${BUILD_URL}", credentialsId: "${CREDENTIALS_ID}"]]])
                    GITHASH = sh(returnStdout: true, script: "git rev-parse HEAD").trim()
                    def GOARCH = "amd64"
                    ["linux", "darwin", "windows"].each {
                        sh """
                        GOOS=${it} GOARCH=${GOARCH} make cli
                        tar -zcf ${TKCTL_CLI_PACKAGE}.tgz tkctl
                        sha256sum ${TKCTL_CLI_PACKAGE}.tgz > ${TKCTL_CLI_PACKAGE}.sha256

                        upload.py ${TKCTL_CLI_PACKAGE}.tgz ${TKCTL_CLI_PACKAGE}.tgz
                        upload.py ${TKCTL_CLI_PACKAGE}.sha256 ${TKCTL_CLI_PACKAGE}.sha256
                        """
                    }
                }

                stage('Build and push debug images'){
                    withDockerServer([uri: "${env.DOCKER_HOST}"]) {
                        DOCKER_REGISTRY="" make debug-docker-push
                        DOCKER_REGISTRY="uhub.service.ucloud.cn" make debug-docker-push
                    }
                }
            }
        }
        currentBuild.result = "SUCCESS"
    }
    stage('Summary') {
        echo("echo summary info ########")
        def DURATION = ((System.currentTimeMillis() - currentBuild.startTimeInMillis) / 1000 / 60).setScale(2, BigDecimal.ROUND_HALF_UP)
        def slackmsg = "[${env.JOB_NAME.replaceAll('%2F','/')}-${env.BUILD_NUMBER}] `${currentBuild.result}`" + "\n" +
                "Elapsed Time: `${DURATION}` Mins" + "\n" +
                "tidb-operator Branch: `${BUILD_BRANCH}`, Githash: `${GITHASH.take(7)}`" + "\n" +
                "Display URL:" + "\n" +
                "${env.RUN_DISPLAY_URL}"

        if(currentBuild.result != "SUCCESS"){
            slackSend channel: '#cloud_jenkins', color: 'danger', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
        } else {
            slackmsg = "${slackmsg}" + "\n" +
                    "tkctl cli tool build and debug image build failed for BRANCH:${BUILD_BRANCH} and TAG:${RELEASE_TAG}`"
            slackSend channel: '#cloud_jenkins', color: 'good', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
        }
    }
}

return this
