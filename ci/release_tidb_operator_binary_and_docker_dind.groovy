def call(BUILD_BRANCH, RELEASE_TAG) {

	env.GOPATH = "/go"
	env.GOROOT = "/usr/local/go"
	env.PATH = "${env.GOROOT}/bin:${env.GOPATH}/bin:/bin:${env.PATH}:/home/jenkins/bin"

	def GITHASH
	def UCLOUD_OSS_URL = "http://pingcap-dev.hk.ufileos.com"
	def TIDB_OPERATOR_CHART = "tidb-operator-${RELEASE_TAG}"
	def TIDB_CLUSTER_CHART = "tidb-cluster-${RELEASE_TAG}"

	catchError {
		node('k8s_centos7_build') {
			def WORKSPACE = pwd()

			dir("${WORKSPACE}/operator"){
				stage('Download tidb-operator binary'){
					GITHASH = sh(returnStdout: true, script: "curl ${UCLOUD_OSS_URL}/refs/pingcap/operator/${BUILD_BRANCH}/centos7/sha1").trim()
					sh "curl ${UCLOUD_OSS_URL}/builds/pingcap/operator/${GITHASH}/centos7/tidb-operator.tar.gz | tar xz"
				}

				stage('Push tidb-operator Docker Image'){
					withDockerServer([uri: "${env.DOCKER_HOST}"]) {
						docker.build("uhub.service.ucloud.cn/pingcap/tidb-operator:${RELEASE_TAG}", "images/tidb-operator").push()
						docker.build("pingcap/tidb-operator:${RELEASE_TAG}", "images/tidb-operator").push()
					}
				}

				stage('Release charts to qiniu'){
					sh """
					# release tidb-operator chart
					sed -i "s/version:.*/version: ${RELEASE_TAG}/g" charts/tidb-operator/Chart.yaml
					tar -zcf ${TIDB_OPERATOR_CHART}.tgz -C charts tidb-operator
					sha256sum ${TIDB_OPERATOR_CHART}.tgz > ${TIDB_OPERATOR_CHART}.sha256

					upload.py ${TIDB_OPERATOR_CHART}.tgz ${TIDB_OPERATOR_CHART}.tgz
					upload.py ${TIDB_OPERATOR_CHART}.sha256 ${TIDB_OPERATOR_CHART}.sha256
					# release tidb-cluster chart
					tar -zcf ${TIDB_CLUSTER_CHART}.tgz -C charts tidb-cluster
					sha256sum ${TIDB_CLUSTER_CHART}.tgz > ${TIDB_CLUSTER_CHART}.sha256

					upload.py ${TIDB_CLUSTER_CHART}.tgz ${TIDB_CLUSTER_CHART}.tgz
					upload.py ${TIDB_CLUSTER_CHART}.sha256 ${TIDB_CLUSTER_CHART}.sha256
					"""
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
			"tidb-operator Docker Image: `pingcap/tidb-operator:${RELEASE_TAG}`" + "\n" +
			"tidb-operator Docker Image: `uhub.ucloud.cn/pingcap/tidb-operator:${RELEASE_TAG}`" + "\n" +
			"tidb-operator charts Download URL: http://download.pingcap.org/${TIDB_OPERATOR_CHART}.tgz" + "\n" +
			"tidb-operator charts sha256: http://download.pingcap.org/${TIDB_OPERATOR_CHART}.sha256" + "\n" +
			"tidb-cluster charts Download URL: http://download.pingcap.org/${TIDB_CLUSTER_CHART}.tgz" + "\n" +
			"tidb-cluster charts sha256: http://download.pingcap.org/${TIDB_CLUSTER_CHART}.sha256"
			slackSend channel: '#cloud_jenkins', color: 'good', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
		}
	}
}

return this
