//
// E2E Jenkins file.
//

import groovy.transform.Field
import groovy.text.SimpleTemplateEngine

podYAML = '''\
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: tidb-operator-e2e
spec:
  containers:
  - name: main
    image: gcr.io/k8s-testimages/kubekins-e2e:v20200311-1e25827-master
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
        ephemeral-storage: 60Gi
    <% } %>
    <% if (resources.limits) { %>
      limits:
        cpu: <%= resources.limits.cpu %>
        memory: <%= resources.limits.memory %>
        ephemeral-storage: 60Gi
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
			cpu: "8",
			memory: "8G"
		],
		limits: [
			cpu: "8",
			memory: "8G"
		],
	]

e2eSerialResources = [
		requests: [
			cpu: "4",
			memory: "8G"
		],
		limits: [
			cpu: "4",
			memory: "8G"
		],
	]

def build(String SHELL_CODE, String ARTIFACTS = "", Map resources = e2ePodResources) {
	podTemplate(yaml: buildPodYAML(resources: resources)) {
		node(POD_LABEL) {
			container('main') {
				def WORKSPACE = pwd()
				try {
					dir("${WORKSPACE}/go/src/github.com/pingcap/tidb-operator") {
						unstash 'tidb-operator'
						stage("Debug Info") {
							println "debug host: 172.16.5.5"
							println "debug command: kubectl -n jenkins-ci exec -ti ${NODE_NAME} bash"
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
							ansiColor('xterm') {
								sh """#!/bin/bash
								export GOPATH=${WORKSPACE}/go
								echo "info: restore docker images"
								images=(
									tidb-operator:latest
									tidb-backup-manager:latest
									tidb-operator-e2e:latest
								)
								for image in \${images[@]}; do
									docker load -i \$image.tar.gz
								done
								${SHELL_CODE}
								"""
							}
						}
					}
				} finally {
					if (ARTIFACTS != "") {
						dir(ARTIFACTS) {
							archiveArtifacts artifacts: "**", allowEmptyArchive: true
							junit testResults: "*.xml", allowEmptyResults: true
						}
					}
				}
			}
		}
	}
}

def getChangeLogText() {
	def changeLogText = ""
	for (int i = 0; i < currentBuild.changeSets.size(); i++) {
		for (int j = 0; j < currentBuild.changeSets[i].items.length; j++) {
			def commitId = "${currentBuild.changeSets[i].items[j].commitId}"
			def commitMsg = "${currentBuild.changeSets[i].items[j].msg}"
			changeLogText += "\n" + "`${commitId.take(7)}` ${commitMsg}"
		}
	}
	return changeLogText
}

def call(BUILD_BRANCH, CREDENTIALS_ID, CODECOV_CREDENTIALS_ID) {
	timeout (time: 2, unit: 'HOURS') {

	def GITHASH
	def UCLOUD_OSS_URL = "http://pingcap-dev.hk.ufileos.com"
	def BUILD_URL = "git@github.com:pingcap/tidb-operator.git"
	def PROJECT_DIR = "go/src/github.com/pingcap/tidb-operator"

	catchError {
		// use fixed label, so we can reuse previous workers
		def buildPodLabel = "tidb-operator-build"
		def resources = [
			requests: [
				cpu: "8",
				memory: "8G"
			],
			limits: [
				cpu: "8",
				memory: "16G"
			],
		]
		podTemplate(
			label: buildPodLabel,
			yaml: buildPodYAML(resources: resources, any: true),
			// We allow this pod to remain active for a while, later jobs can
			// reuse cache in previous workers.
			idleMinutes: 180,
		) {
		node(buildPodLabel) {
			container("main") {
				dir("${PROJECT_DIR}") {

					stage('Checkout') {
						checkout changelog: false,
						poll: false,
						scm: [
							$class: 'GitSCM',
							branches: [[name: "${BUILD_BRANCH}"]],
							userRemoteConfigs: [[
								credentialsId: "${CREDENTIALS_ID}",
								refspec: '+refs/heads/*:refs/remotes/origin/* +refs/pull/*:refs/remotes/origin/pr/*',
								url: "${BUILD_URL}",
							]]
						]

						GITHASH = sh(returnStdout: true, script: "git rev-parse HEAD").trim()
					}

					stage("Build and Test") {
						withCredentials([
							string(credentialsId: "${CODECOV_CREDENTIALS_ID}", variable: 'CODECOV_TOKEN')
						]) {
							ansiColor('xterm') {
								sh """#!/bin/bash
								echo "info: building"
								make build e2e-build
								if [ "${BUILD_BRANCH}" == "master" ]; then
									echo "info: run unit tests and report coverage results for master branch"
									make test GOFLAGS='-race' GO_COVER=y
									curl -s https://codecov.io/bash | bash -s - -t \${CODECOV_TOKEN} || echo 'Codecov did not collect coverage reports'
								fi
								"""
							}
						}
					}

					stage("Prepare for e2e") {
						ansiColor('xterm') {
							sh """#!/bin/bash
							echo "info: preparing for e2e (this will build docker images only)"
							SKIP_BUILD=y SKIP_UP=y SKIP_TEST=y SKIP_DOWN=y ./hack/e2e.sh
							echo "info: saving docker images"
							images=(
								tidb-operator:latest
								tidb-backup-manager:latest
								tidb-operator-e2e:latest
							)
							for image in \${images[@]}; do
								docker save localhost:5000/pingcap/\$image -o \$image.tar.gz
							done
							"""
						}
					}

					stash excludes: "vendor/**,deploy/**", name: "tidb-operator"
				}
			}
		}
		}

		def artifacts = "go/src/github.com/pingcap/tidb-operator/artifacts"
		def GLOBALS = "SKIP_IMAGE_BUILD=y"
		def builds = [:]
		builds["E2E v1.12.10"] = {
			build("${GLOBALS} RUNNER_SUITE_NAME=e2e-v1.12 IMAGE_TAG=${GITHASH} SKIP_BUILD=y GINKGO_NODES=6 KUBE_VERSION=v1.12.10 REPORT_DIR=\$(pwd)/artifacts REPORT_PREFIX=v1.12.10_ ./hack/e2e.sh -- --preload-images --operator-killer", artifacts)
		}
		builds["E2E v1.12.10 AdvancedStatefulSet"] = {
			build("${GLOBALS} RUNNER_SUITE_NAME=e2e-v1.12-advanced-statefulset IMAGE_TAG=${GITHASH} SKIP_BUILD=y GINKGO_NODES=6 KUBE_VERSION=v1.12.10 REPORT_DIR=\$(pwd)/artifacts REPORT_PREFIX=v1.12.10_advanced_statefulset ./hack/e2e.sh -- --preload-images --operator-features AdvancedStatefulSet=true", artifacts)
		}
		builds["E2E v1.18.0"] = {
			build("${GLOBALS} RUNNER_SUITE_NAME=e2e-v1.18 IMAGE_TAG=${GITHASH} SKIP_BUILD=y GINKGO_NODES=6 KUBE_VERSION=v1.18.0 REPORT_DIR=\$(pwd)/artifacts REPORT_PREFIX=v1.18.0_ ./hack/e2e.sh -- -preload-images --operator-killer", artifacts)
		}
		builds["E2E v1.12.10 Serial"] = {
			build("${GLOBALS} RUNNER_SUITE_NAME=e2e-v1.12-serial IMAGE_TAG=${GITHASH} SKIP_BUILD=y KUBE_VERSION=v1.12.10 REPORT_DIR=\$(pwd)/artifacts REPORT_PREFIX=v1.12.10_serial_ ./hack/e2e.sh -- --preload-images --ginkgo.focus='\\[Serial\\]' --install-operator=false", artifacts, e2eSerialResources)
		}
		builds.failFast = false
		parallel builds

		if (!(BUILD_BRANCH ==~ /[a-z0-9]{40}/)) {
			podTemplate(yaml: buildPodYAML(resources: [requests: [cpu: "1", memory: "1G"]])) {
				node(POD_LABEL) {
					container("main") {
						dir("${PROJECT_DIR}") {
							unstash 'tidb-operator'
							stage('upload tidb-operator binaries and charts'){
								withCredentials([
									string(credentialsId: 'UCLOUD_PUBLIC_KEY', variable: 'UCLOUD_PUBLIC_KEY'),
									string(credentialsId: 'UCLOUD_PRIVATE_KEY', variable: 'UCLOUD_PRIVATE_KEY'),
								]) {
									sh """
									export UCLOUD_UFILE_PROXY_HOST=mainland-hk.ufileos.com
									export UCLOUD_UFILE_BUCKET=pingcap-dev
									export BUILD_BRANCH=${BUILD_BRANCH}
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
	}

	stage('Summary') {
		def CHANGELOG = getChangeLogText()
		def duration = ((System.currentTimeMillis() - currentBuild.startTimeInMillis) / 1000 / 60).setScale(2, BigDecimal.ROUND_HALF_UP)
		def slackmsg = "[#${env.ghprbPullId}: ${env.ghprbPullTitle}]" + "\n" +
		"${env.ghprbPullLink}" + "\n" +
		"${env.ghprbPullDescription}" + "\n" +
		"Integration Common Test Result: `${currentBuild.result}`" + "\n" +
		"Elapsed Time: `${duration} mins` " + "\n" +
		"${CHANGELOG}" + "\n" +
		"${env.RUN_DISPLAY_URL}"

		if (currentBuild.result != "SUCCESS") {
			slackSend channel: '#cloud_jenkins', color: 'danger', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
			return
		}

		if ( !(BUILD_BRANCH ==~ /[a-z0-9]{40}/) ){
			slackmsg = "${slackmsg}" + "\n" +
			"Binary Download URL:" + "\n" +
			"${UCLOUD_OSS_URL}/builds/pingcap/operator/${GITHASH}/centos7/tidb-operator.tar.gz"
		}

		slackSend channel: '#cloud_jenkins', color: 'good', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
	}

	}
}

return this

// vim: noet
