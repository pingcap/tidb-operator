//
// Jenkins pipeline for EKS e2e job.
//
// This script is written in declarative syntax. Refer to
// https://jenkins.io/doc/book/pipeline/syntax/ for more details.
//
// Note that parameters of the job is configured in this script.
//

import groovy.transform.Field

@Field
def podYAML = '''
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: main
    image: gcr.io/k8s-testimages/kubekins-e2e:v20200311-1e25827-master
    command:
    - runner.sh
    - sleep
    - 1d
    # we need privileged mode in order to do docker in docker
    securityContext:
      privileged: true
    env:
    - name: DOCKER_IN_DOCKER_ENABLED
      value: "true"
    resources:
      requests:
        memory: "4000Mi"
        cpu: 2000m
        ephemeral-storage: "20Gi"
      limits:
        memory: "4000Mi"
        cpu: 2000m
        ephemeral-storage: "20Gi"
    volumeMounts:
    # dind expects /var/lib/docker to be volume
    - name: docker-root
      mountPath: /var/lib/docker
  volumes:
  - name: docker-root
    emptyDir: {}
'''

// Able to override default values in Jenkins job via environment variables.
if (!env.DEFAULT_GIT_REF) {
    env.DEFAULT_GIT_REF = "master"
}

if (!env.DEFAULT_GINKGO_NODES) {
    env.DEFAULT_GINKGO_NODES = "8"
}

if (!env.DEFAULT_E2E_ARGS) {
    env.DEFAULT_E2E_ARGS = "--ginkgo.skip='\\[Serial\\]|\\[Stability\\]' --ginkgo.focus='\\[tidb-operator\\]'"
}

if (!env.DEFAULT_CLUSTER) {
    env.DEFAULT_CLUSTER = "jenkins-tidb-operator-e2e"
}

if (!env.DEFAULT_AWS_REGION) {
    env.DEFAULT_AWS_REGION = "us-west-2"
}

pipeline {
    agent {
        kubernetes {
            yaml podYAML
            defaultContainer "main"
            customWorkspace "/home/jenkins/agent/workspace/go/src/github.com/pingcap/tidb-operator"
        }
    }

    options {
        timeout(time: 3, unit: 'HOURS')
    }

    parameters {
        string(name: 'GIT_URL', defaultValue: 'git@github.com:pingcap/tidb-operator.git', description: 'git repo url')
        string(name: 'GIT_REF', defaultValue: env.DEFAULT_GIT_REF, description: 'git ref spec to checkout, e.g. master, release-1.1')
        string(name: 'PR_ID', defaultValue: '', description: 'pull request ID, this will override GIT_REF if set, e.g. 1889')
        string(name: 'GINKGO_NODES', defaultValue: env.DEFAULT_GINKGO_NODES, description: 'the number of ginkgo nodes')
        string(name: 'E2E_ARGS', defaultValue: env.DEFAULT_E2E_ARGS, description: "e2e args, e.g. --ginkgo.focus='\\[Stability\\]'")
        string(name: 'CLUSTER', defaultValue: env.DEFAULT_CLUSTER, description: 'the name of the cluster')
        string(name: 'AWS_REGION', defaultValue: env.DEFAULT_AWS_REGION, description: 'the AWS region')
        booleanParam(name: 'DELETE_NAMESPACE_ON_FAILURE', defaultValue: true, description: 'delete namespace on failure or not')
    }

    environment {
        GIT_REF = ''
        ARTIFACTS = "${env.WORKSPACE}/artifacts"
    }

    stages {
        stage("Prepare") {
            steps {
                // The declarative model for Jenkins Pipelines has a restricted
                // subset of syntax that it allows in the stage blocks. We use
                // script step to bypass the restriction.
                // https://jenkins.io/doc/book/pipeline/syntax/#script
                script {
                    GIT_REF = params.GIT_REF
                    if (params.PR_ID != "") {
                        GIT_REF = "refs/remotes/origin/pr/${params.PR_ID}/head"
                    }
                }
                echo "env.NODE_NAME: ${env.NODE_NAME}"
                echo "env.WORKSPACE: ${env.WORKSPACE}"
                echo "GIT_REF: ${GIT_REF}"
                echo "ARTIFACTS: ${ARTIFACTS}"
            }
        }

        stage("Checkout") {
            steps {
                checkout scm: [
                        $class: 'GitSCM',
                        branches: [[name: GIT_REF]],
                        userRemoteConfigs: [[
                            credentialsId: 'github-sre-bot-ssh',
                            refspec: '+refs/heads/*:refs/remotes/origin/* +refs/pull/*:refs/remotes/origin/pr/*',
                            url: "${params.GIT_URL}",
                        ]]
                    ]
            }
        }

        stage("Run") {
            steps {
                withCredentials([
                    string(credentialsId: 'TIDB_OPERATOR_AWS_ACCESS_KEY_ID', variable: 'AWS_ACCESS_KEY_ID'),
                    string(credentialsId: 'TIDB_OPERATOR_AWS_SECRET_ACCESS_KEY', variable: 'AWS_SECRET_ACCESS_KEY'),
                ]) {
                    sh """
                    export PROVIDER=eks
                    export CLUSTER=${params.CLUSTER}
                    export AWS_REGION=${params.AWS_REGION}
                    export GINKGO_NODES=${params.GINKGO_NODES}
                    export DELETE_NAMESPACE_ON_FAILURE=${params.DELETE_NAMESPACE_ON_FAILURE}
                    export ARTIFACTS=${ARTIFACTS}
                    export GINKGO_NO_COLOR=y
                    echo "info: try to clean the cluster created previously"
                    ./ci/aws-clean-eks.sh \$CLUSTER
                    echo "info: begin to run e2e"
                    ./hack/e2e.sh -- ${params.E2E_ARGS}
                    """
                }
            }
        }
    }

    post {
        always {
            dir(ARTIFACTS) {
                sh """#!/bin/bash
                echo "info: change ownerships for jenkins"
                chown -R 1000:1000 .
                echo "info: print total size of artifacts"
                du -sh .
                echo "info: list all files"
                find .
                """
                archiveArtifacts artifacts: "**", allowEmptyArchive: true
                junit testResults: "*.xml", allowEmptyResults: true
            }
        }
        failure {
            sh """
            export CLUSTER=${params.CLUSTER}
            echo "info: cleaning eks cluster \$CLUSTER"
            ./ci/aws-clean-eks.sh \$CLUSTER
            """
        }
    }
}

// vim: et sw=4 ts=4
