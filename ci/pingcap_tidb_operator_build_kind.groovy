//
// E2E Jenkins file.
//

import groovy.transform.Field

@Field
def podYAML = '''
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: main
    image: gcr.io/k8s-testimages/kubekins-e2e:v20191108-9467d02-master
    command:
    - runner.sh
    - sleep
    - 99d
    # we need privileged mode in order to do docker in docker
    securityContext:
      privileged: true
    env:
    - name: DOCKER_IN_DOCKER_ENABLED
      value: "true"
    resources:
      requests:
        memory: "8000Mi"
        cpu: 8000m
        ephemeral-storage: "50Gi"
      limits:
        memory: "8000Mi"
        cpu: 8000m
        ephemeral-storage: "50Gi"
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
'''

def build(SHELL_CODE, ARTIFACTS = "") {
	podTemplate(yaml: podYAML) {
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
								sh """
								export GOPATH=${WORKSPACE}/go
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
	def CODECOV_TOKEN
	def UCLOUD_OSS_URL = "http://pingcap-dev.hk.ufileos.com"
	def BUILD_URL = "git@github.com:pingcap/tidb-operator.git"
	def PROJECT_DIR = "go/src/github.com/pingcap/tidb-operator"

	catchError {
		node('build_go1130_memvolume') {
			container("golang") {
				def WORKSPACE = pwd()
				dir("${PROJECT_DIR}") {
					stage('Checkout') {
						checkout changelog: false,
						poll: false,
						scm: [
							$class: 'GitSCM',
							branches: [[name: "${BUILD_BRANCH}"]],
							doGenerateSubmoduleConfigurations: false,
							extensions: [],
							submoduleCfg: [],
							userRemoteConfigs: [[
								credentialsId: "${CREDENTIALS_ID}",
								refspec: '+refs/heads/*:refs/remotes/origin/* +refs/pull/*:refs/remotes/origin/pr/*',
								url: "${BUILD_URL}",
							]]
						]

						GITHASH = sh(returnStdout: true, script: "git rev-parse HEAD").trim()
						withCredentials([string(credentialsId: "${CODECOV_CREDENTIALS_ID}", variable: 'codecovToken')]) {
							CODECOV_TOKEN = codecovToken
						}
					}

					stage("Check") {
						ansiColor('xterm') {
							sh """
							export GOPATH=${WORKSPACE}/go
							export PATH=${WORKSPACE}/go/bin:\$PATH
							make check-setup
							make check
							"""
						}
					}

					stage("Build and Test") {
						ansiColor('xterm') {
							sh """
							make
							make e2e-build
							if [ ${BUILD_BRANCH} == "master" ]
							then
								make test GO_COVER=y
								curl -s https://codecov.io/bash | bash -s - -t ${CODECOV_TOKEN} || echo 'Codecov did not collect coverage reports'
							else
								make test
							fi
							"""
						}
					}

					stage("Prepare for e2e") {
						ansiColor('xterm') {
							sh """
							hack/prepare-e2e.sh
							"""
						}
					}

					stash excludes: "vendor/**,deploy/**", name: "tidb-operator"
				}
			}
		}

		def artifacts = "go/src/github.com/pingcap/tidb-operator/artifacts"
		// unstable in our IDC, disable temporarily
		//def MIRRORS = "DOCKER_IO_MIRROR=https://dockerhub.azk8s.cn GCR_IO_MIRROR=https://gcr.azk8s.cn QUAY_IO_MIRROR=https://quay.azk8s.cn"
		def MIRRORS = "DOCKER_IO_MIRROR=http://172.16.4.143:5000 QUAY_IO_MIRROR=http://172.16.4.143:5001"
		def builds = [:]
		builds["E2E v1.12.10"] = {
			build("${MIRRORS} IMAGE_TAG=${GITHASH} SKIP_BUILD=y GINKGO_NODES=8 KUBE_VERSION=v1.12.10 REPORT_DIR=\$(pwd)/artifacts REPORT_PREFIX=v1.12.10_ ./hack/e2e.sh -- --preload-images --ginkgo.skip='\\[Serial\\]'", artifacts)
		}
		builds["E2E v1.16.4"] = {
			build("${MIRRORS} IMAGE_TAG=${GITHASH} SKIP_BUILD=y GINKGO_NODES=8 KUBE_VERSION=v1.16.4 REPORT_DIR=\$(pwd)/artifacts REPORT_PREFIX=v1.16.4_ ./hack/e2e.sh -- --preload-images --ginkgo.skip='\\[Serial\\]'", artifacts)
		}
		builds["E2E v1.12.10 AdvancedStatefulSet"] = {
			build("${MIRRORS} IMAGE_TAG=${GITHASH} SKIP_BUILD=y GINKGO_NODES=8 KUBE_VERSION=v1.12.10 REPORT_DIR=\$(pwd)/artifacts REPORT_PREFIX=v1.12.10_advanced_statefulset ./hack/e2e.sh -- --preload-images --ginkgo.skip='\\[Serial\\]' --operator-features AdvancedStatefulSet=true", artifacts)
		}
		builds["E2E v1.17.0"] = {
			build("${MIRRORS} IMAGE_TAG=${GITHASH} SKIP_BUILD=y GINKGO_NODES=8 KUBE_VERSION=v1.17.0 REPORT_DIR=\$(pwd)/artifacts REPORT_PREFIX=v1.17.0_ ./hack/e2e.sh -- -preload-images --ginkgo.skip='\\[Serial\\]'", artifacts)
		}
		builds["E2E v1.12.10 Serial"] = {
			build("${MIRRORS} IMAGE_TAG=${GITHASH} SKIP_BUILD=y KUBE_VERSION=v1.12.10 REPORT_DIR=\$(pwd)/artifacts REPORT_PREFIX=v1.12.10_serial_ ./hack/e2e.sh -- --preload-images --ginkgo.focus='\\[Serial\\]' --install-operator=false", artifacts)
		}
		builds.failFast = false
		parallel builds

		// we requires ~/bin/config.cfg, filemgr-linux64 utilities on k8s-kind node
		// TODO make it possible to run on any node
		node('k8s-kind') {
			dir("${PROJECT_DIR}"){
				deleteDir()
				unstash 'tidb-operator'
				if ( !(BUILD_BRANCH ==~ /[a-z0-9]{40}/) ) {
					stage('upload tidb-operator, backup-manager binary and charts'){
						//upload binary and charts
						sh """
						cp ~/bin/config.cfg ./
						tar -zcvf tidb-operator.tar.gz images/tidb-operator images/backup-manager charts
						filemgr-linux64 --action mput --bucket pingcap-dev --nobar --key builds/pingcap/operator/${GITHASH}/centos7/tidb-operator.tar.gz --file tidb-operator.tar.gz
						"""
						//update refs
						writeFile file: 'sha1', text: "${GITHASH}"
						sh """
						filemgr-linux64 --action mput --bucket pingcap-dev --nobar --key refs/pingcap/operator/${BUILD_BRANCH}/centos7/sha1 --file sha1
						rm -f sha1 tidb-operator.tar.gz config.cfg
						"""
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
