//
// Jenkins pipeline to release nightly build images.
//
// We uses ghprb plugin to build pull requests and report results. Some special
// environment variables will be available for jobs that are triggered by GitHub
// Pull Request events.
//
// - ghprbActualCommit
//
// For more information about this plugin, please check out https://plugins.jenkins.io/ghprb/.
//

import groovy.text.SimpleTemplateEngine

// Able to override default values in Jenkins job via environment variables.
if (!env.DEFAULT_GIT_REF) {
    env.DEFAULT_GIT_REF = "master"
}

properties([
    parameters([
        string(name: 'GIT_URL', defaultValue: 'https://github.com/pingcap/tidb-operator', description: 'git repo url'),
        string(name: 'GIT_REF', defaultValue: env.DEFAULT_GIT_REF, description: 'git ref spec to checkout, e.g. master, release-1.1'),
        string(name: 'RELEASE_VER', defaultValue: '', description: "the version string in released tarball"),
    ])
])

podYAML = '''
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: tidb-operator-nightly
spec:
  containers:
  - name: golang-builder
    image: 'golang:latest'
    command: ["/bin/bash", "-c", "--"]
    args: ["trap : TERM; sleep infinity & wait"]
'''

try {
    def GITHASH
    def IMAGE_TAG

    def PROJECT_DIR = "go/src/github.com/pingcap/tidb-operator"

    // Git ref to checkout
    def GIT_REF = params.GIT_REF
    if (params.GIT_REF == "") {
        GIT_REF = env.ghprbActualCommit
    }

    timeout (time: 20, unit: 'MINUTES') {
        // use fixed label, so we can reuse previous workers
        // increase version in pod label when we update pod template
        def buildPodLabel = "tidb-operator-build-nightly"
        def resources = [
            requests: [
                cpu: "4",
                memory: "4G"
            ],
            limits: [
                cpu: "8",
                memory: "16G"
            ],
        ]
        podTemplate(
            label: buildPodLabel,
            yaml: podYAML,
            // We allow this pod to remain active for a while, later jobs can
            // reuse cache in previous created nodes.
            idleMinutes: 180,
        ) {
        node(buildPodLabel) {
            container("golang-builder") {
                dir("${PROJECT_DIR}") {

                    stage('Checkout') {
                        sh """
                        echo "info: change ownerships for jenkins"
                        # we run as root in our pods, this is required
                        # otherwise jenkins agent will fail because of the lack of permission
                        chown -R 1000:1000 .
                        """

                        // clean stale files because we may reuse previous created nodes
                        deleteDir()

                        checkout changelog: false, poll: false, scm: [
                                $class: 'GitSCM',
                                branches: [[name: "${GIT_REF}"]],
                                userRemoteConfigs: [[
                                    refspec: '+refs/heads/*:refs/remotes/origin/* +refs/pull/*:refs/remotes/origin/pull/*',
                                    url: "${params.GIT_URL}",
                                ]]
                            ]

                        GITHASH = sh(returnStdout: true, script: "git rev-parse HEAD").trim()
                        IMAGE_TAG = env.JOB_NAME + "-" + GITHASH.substring(0, 6)
                    }

                    stage("Test and Build") {
                        sh """#!/bin/bash
                        set -eu
                        echo "info: run unit tests"
                        GOFLAGS='-race' make test
                        echo "info: building"
                        make build
                        """
                    }

                    stage('Upload binaries and charts'){
                        withCredentials([
                            string(credentialsId: 'UCLOUD_PUBLIC_KEY', variable: 'UCLOUD_PUBLIC_KEY'),
                            string(credentialsId: 'UCLOUD_PRIVATE_KEY', variable: 'UCLOUD_PRIVATE_KEY'),
                        ]) {
                            sh """
                            export UCLOUD_UFILE_PROXY_HOST=pingcap-dev.hk.ufileos.com
                            export UCLOUD_UFILE_BUCKET=pingcap-dev
                            export BUILD_BRANCH=${GIT_REF}
                            export GITHASH=${GITHASH}
                            ./ci/upload-binaries-charts.sh
                            """
                        }
                    }
                }
            }
        }
        }
    }

    currentBuild.result = "SUCCESS"
} catch (err) {
    println("fatal: " + err)
    currentBuild.result = 'FAILURE'
}

// vim: et sw=4 ts=4
