//
// Jenkins pipeline for e2e kind job.
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
env.DEFAULT_GIT_REF = env.DEFAULT_GIT_REF ?: 'master'
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
        string(name: 'DELETE_NAMESPACE_ON_FAILURE', defaultValue: env.DEFAULT_DELETE_NAMESPACE_ON_FAILURE, description: 'delete ns after test case fails')
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
    image: hub-new.pingcap.net/tidb-operator/kubekins-e2e:v20210808-1eaeec7-master
    command:
    - runner.sh
    # Clean containers on TERM signal in root process to avoid cgroup leaking.
    # https://github.com/pingcap/tidb-operator/issues/1603#issuecomment-582402196
    - exec
    - bash
    - -c
    - |
      function clean() {
        echo "info: clean all containers to avoid cgroup leaking"
        docker kill `docker ps -q` || true
        docker system prune -af || true
      }
      function setup_docker_mirror() {
        sed -i "s/mirror.gcr.io/registry-mirror.pingcap.net/g" /etc/default/docker
        service docker restart
      }
      setup_docker_mirror
      trap clean TERM
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
        ephemeral-storage: 150Gi
    <% } %>
    <% if (resources.limits) { %>
      limits:
        cpu: <%= resources.limits.cpu %>
        memory: <%= resources.limits.memory %>
        ephemeral-storage: 150Gi
    <% } %>
<% } %>
    # kind needs /lib/modules and cgroups from the host
    volumeMounts:
    - mountPath: /lib/modules
      name: modules
      readOnly: true
    - mountPath: /sys/fs/cgroup
      name: cgroup
    # dind expects /var/lib/docker to be volume
    - name: docker-root
      mountPath: /var/lib/docker
    # legacy docker path for cr.io/k8s-testimages/kubekins-e2e
    - name: docker-graph
      mountPath: /docker-graph
    # use memory storage for etcd hostpath in kind cluster
    - name: etcd-data-dir
      mountPath: /mnt/tmpfs/etcd
  volumes:
  - name: modules
    hostPath:
      path: /lib/modules
      type: Directory
  - name: cgroup
    hostPath:
      path: /sys/fs/cgroup
      type: Directory
  - name: docker-root
    emptyDir: {}
  - name: docker-graph
    emptyDir: {}
  - name: etcd-data-dir
    emptyDir:
      medium: Memory
  tolerations:
  - effect: NoSchedule
    key: tidb-operator
    operator: Exists
  affinity:
<% if (!any) { %>
    # run on nodes prepared for tidb-operator by default
    # https://github.com/pingcap/tidb-operator/issues/1603
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: ci.pingcap.com
            operator: In
            values:
            - tidb-operator
<% } %>
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
        cpu: "6",
        memory: "10Gi"
    ],
    limits: [
        cpu: "8",
        memory: "16Gi"
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
                            println "debug host: 172.16.5.15"
                            println "debug command: kubectl -n jenkins-tidb exec -ti ${NODE_NAME} bash"
                            sh """
                            echo "====== shell env ======"
                            echo "pwd: \$(pwd)"
                            env
                            echo "====== go env ======"
                            go env
                            echo "====== docker version ======"
                            docker version
                            """
                        }
                        stage('Run') {
                            sh """#!/bin/bash
                            export GOPATH=${WORKSPACE}/go
                            export ARTIFACTS=${ARTIFACTS}
                            export RUNNER_SUITE_NAME=${name}
                            ${code}
                            """
                        }
                    }
                } finally {
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
        def buildPodLabel = "tidb-operator-build-v2-pingcap-docker-mirror"
        def resources = [
            requests: [
                cpu: "6",
                memory: "10Gi"
            ],
            limits: [
                cpu: "8",
                memory: "32Gi"
            ],
        ]
        podTemplate(
            cloud: "kubernetes-ng",
            namespace: "jenkins-tidb-operator",
            label: buildPodLabel,
            yaml: buildPodYAML(resources: resources, any: true),
            // We allow this pod to remain active for a while, later jobs can
            // reuse cache in previous created nodes.
            idleMinutes: 180,
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
                    }

                    stage("Build and Test") {
                        withCredentials([
                            string(credentialsId: "tp-codecov-token", variable: 'CODECOV_TOKEN')
                        ]) {
                            sh """#!/bin/bash
                            set -eu
                            echo "info: building"
                            GOARCH=arm64 make build
                            GOARCH=amd64 make build
                            make e2e-build
                            if [ "${GIT_REF}" == "master" ]; then
                                echo "info: run unit tests and report coverage results for master branch"
                                make test GOFLAGS='-race' GO_COVER=y
                                curl -s https://codecov.io/bash | bash -s - -t \${CODECOV_TOKEN} || echo 'Codecov did not collect coverage reports'
                            fi
                            """
                        }
                    }

                    stage("Prepare for e2e") {
                        withCredentials([usernamePassword(credentialsId: 'TIDB_OPERATOR_HUB_AUTH', usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD')]) {
                            sh """#!/bin/bash
                            set -eu
                            echo "info: logging into hub.pingcap.net"
                            docker login -u \$USERNAME --password-stdin hub.pingcap.net <<< \$PASSWORD
                            echo "info: build and push images for e2e"
                            echo "test: show docker daemon config file"
                            cat /etc/docker/daemon.json || true
                            NO_BUILD=y DOCKER_REPO=hub.pingcap.net/tidb-operator-e2e IMAGE_TAG=${IMAGE_TAG} make docker-push e2e-docker-push
                            echo "info: download binaries for e2e"
                            SKIP_BUILD=y SKIP_IMAGE_BUILD=y SKIP_UP=y SKIP_TEST=y SKIP_DOWN=y ./hack/e2e.sh
                            echo "info: change ownerships for jenkins"
                            # we run as root in our pods, this is required
                            # otherwise jenkins agent will fail because of the lack of permission
                            chown -R 1000:1000 .
                            """
                        }
                        stash excludes: "vendor/**,deploy/**,tests/**", name: "tidb-operator"
                    }
                }
            }
        }
        }

        def GLOBALS = "KIND_ETCD_DATADIR=/mnt/tmpfs/etcd SKIP_BUILD=y SKIP_IMAGE_BUILD=y DOCKER_REPO=hub.pingcap.net/tidb-operator-e2e IMAGE_TAG=${IMAGE_TAG} DELETE_NAMESPACE_ON_FAILURE=${params.DELETE_NAMESPACE_ON_FAILURE} GINKGO_NO_COLOR=y"
        build("tidb-operator", "${GLOBALS} GINKGO_NODES=${params.GINKGO_NODES} ./hack/e2e.sh -- ${params.E2E_ARGS}")

        if (GIT_REF ==~ /^(master|)$/ || GIT_REF ==~ /^(release-.*)$/
            || GIT_REF ==~ /^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$/) {
            // Upload assets if the git ref is the master branch or version tag
            podTemplate(yaml: buildPodYAML(resources: [requests: [cpu: "1", memory: "2Gi"]]),
                        cloud: "kubernetes-ng", namespace: "jenkins-tidb-operator",
            ) {
                node(POD_LABEL) {
                    container("main") {
                        dir("${PROJECT_DIR}") {
                            unstash 'tidb-operator'
                            stage('upload tidb-operator binaries and charts'){
                              sh """
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
