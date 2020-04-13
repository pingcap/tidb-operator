//
// Jenkins pipeline for VM jobs.
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
    volumeMounts:
    # dind expects /var/lib/docker to be volume
    - name: docker-root
      mountPath: /var/lib/docker
  volumes:
  - name: docker-root
    emptyDir: {}
'''

// Able to override default values in Jenkins job via environment variables.

if (!env.DEFAULT_GIT_URL) {
    env.DEFAULT_GIT_URL = "https://github.com/pingcap/tidb-operator"
}

if (!env.DEFAULT_GIT_REF) {
    env.DEFAULT_GIT_REF = "master"
}

if (!env.DEFAULT_GCP_PROJECT) {
    env.DEFAULT_GCP_PROJECT = ""
}

if (!env.DEFAULT_GCP_ZONE) {
    env.DEFAULT_GCP_ZONE = "us-central1-b"
}

if (!env.DEFAULT_NAME) {
    env.DEFAULT_NAME = "tidb-operator-e2e"
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
        string(name: 'GIT_URL', defaultValue: env.DEFAULT_GIT_URL, description: 'git repo url')
        string(name: 'GIT_REF', defaultValue: env.DEFAULT_GIT_REF, description: 'git ref spec to checkout, e.g. master, release-1.1')
        string(name: 'PR_ID', defaultValue: '', description: 'pull request ID, this will override GIT_REF if set, e.g. 1889')
        string(name: 'GCP_PROJECT', defaultValue: env.DEFAULT_GCP_PROJECT, description: 'the GCP project ID')
        string(name: 'GCP_ZONE', defaultValue: env.DEFAULT_GCP_ZONE, description: 'the GCP zone')
        string(name: 'NAME', defaultValue: env.DEFAULT_NAME, description: 'the name of VM instance')
    }

    environment {
        GIT_REF = ''
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
            }
        }

        stage("Checkout") {
            steps {
                checkout scm: [
                        $class: 'GitSCM',
                        branches: [[name: GIT_REF]],
                        userRemoteConfigs: [[
                            refspec: '+refs/heads/*:refs/remotes/origin/* +refs/pull/*:refs/remotes/origin/pr/*',
                            url: "${params.GIT_URL}",
                        ]]
                    ]
            }
        }

        stage("Run") {
            steps {
                withCredentials([
                    file(credentialsId: 'TIDB_OPERATOR_GCP_CREDENTIALS', variable: 'GCP_CREDENTIALS'),
                    file(credentialsId: 'TIDB_OPERATOR_GCP_SSH_PRIVATE_KEY', variable: 'GCP_SSH_PRIVATE_KEY'),
                    file(credentialsId: 'TIDB_OPERATOR_GCP_SSH_PUBLIC_KEY', variable: 'GCP_SSH_PUBLIC_KEY'),
                    file(credentialsId: 'TIDB_OPERATOR_REDHAT_PULL_SECRET', variable: 'REDHAT_PULL_SECRET'),
                ]) {
                    sh """
                    #!/bin/bash
                    export GIT_REF=${GIT_REF}
                    export SYNC_FILES=\$REDHAT_PULL_SECRET:/tmp/pull-secret.txt
                    # TODO make the command configurable
                    ./ci/run-in-vm.sh PULL_SECRET_FILE=/tmp/pull-secret.txt ./hack/e2e-openshift.sh
                    """
                }
            }
        }
    }
}

// vim: et sw=4 ts=4
