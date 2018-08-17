def call(BUILD_BRANCH, RELEASE_TAG) {

	env.GOPATH = "/go"
	env.GOROOT = "/usr/local/go"
	env.PATH = "${env.GOROOT}/bin:${env.GOPATH}/bin:/bin:${env.PATH}:/home/jenkins/bin"

	def GITHASH
	def UCLOUD_OSS_URL = "http://pingcap-dev.hk.ufileos.com"

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
					tar -zcf tidb-operator-charts-${RELEASE_TAG}.tar.gz charts
					sha256sum tidb-operator-charts-${RELEASE_TAG}.tar.gz > tidb-operator-charts-${RELEASE_TAG}.sha256

					upload.py tidb-operator-charts-${RELEASE_TAG}.tar.gz tidb-operator-charts-${RELEASE_TAG}.tar.gz
					upload.py tidb-operator-charts-${RELEASE_TAG}.sha256 tidb-operator-charts-${RELEASE_TAG}.sha256
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
			"tidb-operator charts Download URL: http://download.pingcap.org/tidb-operator-charts-${RELEASE_TAG}.tar.gz" + "\n" +
			"tidb-operator charts sha256: http://download.pingcap.org/tidb-operator-charts-${RELEASE_TAG}.sha256"
			slackSend channel: '#cloud_jenkins', color: 'good', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
		}
	}
}

return this
