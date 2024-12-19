//
// Jenkins pipeline for e2e kind job (TiDB Operator v2).
//
// We uses ghprb plugin to build pull requests and report results. Some special
// environment variables will be available for jobs that are triggered by GitHub
// Pull Request events.
//
// - ghprbActualCommit
//
// For more information about this plugin, please check out https://plugins.jenkins.io/ghprb/.
//

// Able to override default values in Jenkins job via environment variables.
env.DEFAULT_GIT_REF = env.DEFAULT_GIT_REF ?: 'feature/v2'
env.DEFAULT_GINKGO_NODES = env.DEFAULT_GINKGO_NODES ?: '8'
env.DEFAULT_E2E_ARGS = env.DEFAULT_E2E_ARGS ?: ''
env.DEFAULT_DELETE_NAMESPACE_ON_FAILURE = env.DEFAULT_DELETE_NAMESPACE_ON_FAILURE ?: 'true'

properties([
    parameters([
        string(name: 'GIT_URL', defaultValue: 'https://github.com/pingcap/tidb-operator', description: 'git repo url'),
        string(name: 'GIT_REF', defaultValue: env.DEFAULT_GIT_REF, description: 'git ref spec to checkout, e.g. master, release-1.1'),
        string(name: 'RELEASE_VER', defaultValue: '', description: "the version string in released tarball"),
        string(name: 'PR_ID', defaultValue: '', description: 'pull request ID, this will override GIT_REF if set, e.g. 1889'),
        string(name: 'GINKGO_NODES', defaultValue: env.DEFAULT_GINKGO_NODES, description: 'the number of ginkgo nodes'),
        string(name: 'E2E_ARGS', defaultValue: env.DEFAULT_E2E_ARGS, description: "e2e args, e.g. --ginkgo.focus='\\[Stability\\]'"),
        string(name: 'DELETE_NAMESPACE_ON_FAILURE', defaultValue: env.DEFAULT_DELETE_NAMESPACE_ON_FAILURE, description: 'delete ns after test case fails'),
    ])
])

podYAML = '''\
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: tidb-operator-e2e
spec:
  containers:
  - name: main
    image: hub.pingcap.net/tidb-operator/kubekins-e2e:v8-go1.23.1
    command:
    - runner.sh
    - exec
    - bash
    - -c
    - |
      sleep 1d & wait
    # we need privileged mode in order to do docker in docker
    securityContext:
      privileged: true
    env:
    - name: DOCKER_IN_DOCKER_ENABLED
      value: "true"
<% if (resources && (resources.requests || resources.limits)) { %>
    resources:
    <% if (resources.requests) { %>
      requests:
        cpu: <%= resources.requests.cpu %>
        memory: <%= resources.requests.memory %>
        ephemeral-storage: <%= resources.requests.storage %>
    <% } %>
    <% if (resources.limits) { %>
      limits:
        cpu: <%= resources.limits.cpu %>
        memory: <%= resources.limits.memory %>
    <% } %>
<% } %>
    # kind needs /lib/modules from the host
    volumeMounts:
    - mountPath: /lib/modules
      name: modules
      readOnly: true
    # dind expects /var/lib/docker to be volume
    - name: docker-root
      mountPath: /var/lib/docker
    # legacy docker path for cr.io/k8s-testimages/kubekins-e2e
    - name: docker-graph
      mountPath: /docker-graph
    # use memory storage for etcd hostpath in kind cluster
    - name: kind-data-dir
      mountPath: /kind-data
    - name: etcd-data-dir
      mountPath: /mnt/tmpfs/etcd
  volumes:
  - name: modules
    hostPath:
      path: /lib/modules
      type: Directory
  - name: docker-root
    emptyDir: {}
  - name: docker-graph
    emptyDir: {}
  - name: kind-data-dir
    emptyDir: {}
  - name: etcd-data-dir
    emptyDir: {}
  tolerations:
  - effect: NoSchedule
    key: tidb-operator
    operator: Exists
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchExpressions:
            - key: app
              operator: In
              values:
              - tidb-operator-e2e
          topologyKey: kubernetes.io/hostname
'''

String buildPodYAML(Map m = [:]) {
    m.putIfAbsent("resources", [:])
    m.putIfAbsent("any", false)
    def engine = new groovy.text.SimpleTemplateEngine()
    def template = engine.createTemplate(podYAML).make(m)
    return template.toString()
}

e2ePodResources = [
    requests: [
        cpu: "16",
        memory: "32Gi",
        storage: "250Gi"
    ],
    limits: [
        cpu: "32",
        memory: "64Gi",
        storage: "250Gi"
    ],
]

def build(String name, String code, Map resources = e2ePodResources) {
    podTemplate(yaml: buildPodYAML(resources: resources), namespace: "jenkins-tidb-operator", cloud: "kubernetes-ng") {
        node(POD_LABEL) {
            container('main') {
                def WORKSPACE = pwd()
                def ARTIFACTS = "${WORKSPACE}/go/src/github.com/pingcap/tidb-operator/_artifacts"
                try {
                    dir("${WORKSPACE}/go/src/github.com/pingcap/tidb-operator") {
                        unstash 'tidb-operator'
                        stage("Debug Info") {
                            sh """
                            sleep 1d & wait
                            echo "====== shell env ======"
                            echo "pwd: \$(pwd)"
                            env
                            unset GOSUMDB
                            echo "====== go env ======"
                            go env
                            echo "====== docker version ======"
                            docker version
                            """
                        }
                        stage('Run') {
                            sh """#!/bin/bash
                            unset GOSUMDB
                            export GOPATH=${WORKSPACE}/go
                            export ARTIFACTS=${ARTIFACTS}
                            export RUNNER_SUITE_NAME=${name}

                            echo "info: create local path for data"
                            mount --make-rshared /
                            ${code}
                            """
                        }
                    }
                } finally {
                    stage('Artifacts') {
                        dir(ARTIFACTS) {
                            sh """#!/bin/bash
                            echo "info: change ownerships for jenkins"
                            chown -R 1000:1000 .
                            echo "info: print total size of artifacts"
                            du -sh .
                            echo "info: list all files"
                            find .
                            echo "info: moving all artifacts into a sub-directory"
                            shopt -s extglob
                            mkdir ${name}
                            mv !(${name}) ${name}/
                            """
                            archiveArtifacts artifacts: "${name}/**", allowEmptyArchive: true
                            junit testResults: "${name}/*.xml", allowEmptyResults: true, keepLongStdio: true
                        }
                    }
                }
            }
        }
    }
}


try {

    def GITHASH
    def IMAGE_TAG

    def PROJECT_DIR = "go/src/github.com/pingcap/tidb-operator"

    // Git ref to checkout
    def GIT_REF = params.GIT_REF
    if (params.PR_ID != "") {
        GIT_REF = "refs/remotes/origin/pull/${params.PR_ID}/head"
    } else if (env.ghprbActualCommit) {
        // for PR jobs triggered by ghprb plugin
        GIT_REF = env.ghprbActualCommit
    }

    timeout (time: 2, unit: 'HOURS') {
        // use fixed label, so we can reuse previous workers
        // increase version in pod label when we update pod template
        def buildPodLabel = "tidb-operator-build-v8-pingcap-docker-mirror"
        def resources = [
            requests: [
                cpu: "4",
                memory: "10Gi",
                storage: "50Gi"
            ],
            limits: [
                cpu: "8",
                memory: "32Gi",
                storage: "50Gi"
            ],
        ]
        podTemplate(
            cloud: "kubernetes-ng",
            namespace: "jenkins-tidb-operator",
            label: buildPodLabel,
            yaml: buildPodYAML(resources: resources, any: true),
            // We allow this pod to remain active for a while, later jobs can
            // reuse cache in previous created nodes.
            idleMinutes: 30,
        ) {
        node(buildPodLabel) {
            container("main") {
                dir("${PROJECT_DIR}") {

                    stage('Checkout') {
                        sh """
                        echo "info: change ownerships for jenkins"
                        # we run as root in our pods, this is required
                        # otherwise jenkins agent will fail because of the lack of permission
                        chown -R 1000:1000 .
                        git config --global --add safe.directory '*'
                        """

                        // clean stale files because we may reuse previous created nodes
                        deleteDir()

                        try {
                            checkout changelog: false, poll: false, scm: [
                                    $class: 'GitSCM',
                                    branches: [[name: "${GIT_REF}"]],
                                    userRemoteConfigs: [[
                                            refspec: '+refs/heads/*:refs/remotes/origin/* +refs/pull/*:refs/remotes/origin/pull/*',
                                            url: "${params.GIT_URL}",
                                    ]]
                            ]
                        } catch (info) {
                            retry(3) {
                                echo "checkout failed, retry.."
                                sleep 10
                                checkout changelog: false, poll: false, scm: [
                                        $class: 'GitSCM',
                                        branches: [[name: "${GIT_REF}"]],
                                        userRemoteConfigs: [[
                                                refspec: '+refs/heads/*:refs/remotes/origin/* +refs/pull/*:refs/remotes/origin/pull/*',
                                                url: "${params.GIT_URL}",
                                        ]]
                                ]
                            }
                        }

                        GITHASH = sh(returnStdout: true, script: "git rev-parse HEAD").trim()
                        IMAGE_TAG = env.JOB_NAME + "-" + GITHASH.substring(0, 6)

                        stash 'tidb-operator'
                    }
                }
            }
        }
        }

        def GLOBALS = "KIND_DATA_HOSTPATH=/kind-data KIND_ETCD_DATADIR=/mnt/tmpfs/etcd E2E=y SKIP_BUILD=y SKIP_IMAGE_BUILD=y DOCKER_REPO=hub.pingcap.net/tidb-operator-e2e IMAGE_TAG=${IMAGE_TAG} DELETE_NAMESPACE_ON_FAILURE=${params.DELETE_NAMESPACE_ON_FAILURE} GINKGO_NO_COLOR=y"
        build("tidb-operator", "make e2e")
    }
    currentBuild.result = "SUCCESS"
} catch (err) {
    println("fatal: " + err)
    currentBuild.result = 'FAILURE'
}
