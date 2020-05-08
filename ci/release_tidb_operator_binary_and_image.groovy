def call(BUILD_BRANCH, RELEASE_TAG, CREDENTIALS_ID, CHART_ITEMS) {
	def GITHASH
	def ACCESS_KEY
	def SECRET_KEY
	def UCLOUD_OSS_URL = "http://pingcap-dev.hk.ufileos.com"

	catchError {
		node('delivery') {
			container("delivery") {
				def WORKSPACE = pwd()
				withCredentials([string(credentialsId: "${env.QN_ACCESS_KET_ID}", variable: 'QN_access_key'), string(credentialsId: "${env.QN_SECRET_KEY_ID}", variable: 'Qiniu_secret_key')]) {
					ACCESS_KEY = QN_access_key
					SECRET_KEY = Qiniu_secret_key
				}

				sh "chown -R jenkins:jenkins ./"
				deleteDir()

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

					stage('Push tidb-backup-manager Docker Image'){
						withDockerServer([uri: "${env.DOCKER_HOST}"]) {
							docker.build("uhub.service.ucloud.cn/pingcap/tidb-backup-manager:${RELEASE_TAG}", "images/tidb-backup-manager").push()
							docker.build("pingcap/tidb-backup-manager:${RELEASE_TAG}", "images/tidb-backup-manager").push()
						}
					}

					stage('Release charts to qiniu'){
						ansiColor('xterm') {
						sh """
						set +x
						export QINIU_ACCESS_KEY="${ACCESS_KEY}"
						export QINIU_SECRET_KEY="${SECRET_KEY}"
						export QINIU_BUCKET_NAME="charts"
						set -x
						curl https://raw.githubusercontent.com/pingcap/docs-cn/master/scripts/upload.py -o upload.py
						sed -i 's%http://download.pingcap.org%http://charts.pingcap.org%g' upload.py
						sed -i 's/python3/python/g' upload.py
						chmod +x upload.py
						for chartItem in ${CHART_ITEMS}
						do
							chartPrefixName=\$chartItem-${RELEASE_TAG}
							echo "======= release \$chartItem chart ======"
							sed -i "s/version:.*/version: ${RELEASE_TAG}/g" charts/\$chartItem/Chart.yaml
                            # update image tag to current release
                            sed -r -i "s#pingcap/(tidb-operator|tidb-backup-manager):.*#pingcap/\\1:${RELEASE_TAG}#g" charts/\$chartItem/values.yaml
							tar -zcf \${chartPrefixName}.tgz -C charts \$chartItem
							sha256sum \${chartPrefixName}.tgz > \${chartPrefixName}.sha256
							./upload.py \${chartPrefixName}.tgz \${chartPrefixName}.tgz
							./upload.py \${chartPrefixName}.sha256 \${chartPrefixName}.sha256
						done
						#Generate index.yaml for helm repo
						wget https://storage.googleapis.com/kubernetes-helm/helm-v2.14.1-linux-amd64.tar.gz
						tar -zxvf helm-v2.14.1-linux-amd64.tar.gz
						mv linux-amd64/helm /usr/local/bin/helm
						chmod +x /usr/local/bin/helm
						#ls
						curl http://charts.pingcap.org/index.yaml -o index.yaml
						cat index.yaml
						helm repo index . --url http://charts.pingcap.org/ --merge index.yaml
						cat index.yaml
						./upload.py index.yaml index.yaml
						"""
						}
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
			return
		}

		slackmsg = "${slackmsg}" + "\n" +
		"tidb-operator Docker Image: `pingcap/tidb-operator:${RELEASE_TAG}`" + "\n" +
		"tidb-operator Docker Image: `uhub.ucloud.cn/pingcap/tidb-operator:${RELEASE_TAG}`" + "\n" +
		"tidb-backup-manager Docker Image: `pingcap/tidb-backup-manager:${RELEASE_TAG}`" + "\n" +
		"tidb-backup-manager Docker Image: `uhub.ucloud.cn/pingcap/tidb-backup-manager:${RELEASE_TAG}`"


		for(String chartItem : CHART_ITEMS.split(' ')){
			slackmsg = "${slackmsg}" + "\n" +
			"${chartItem} charts Download URL: http://charts.pingcap.org/${chartItem}-${RELEASE_TAG}.tgz"
		}
		slackmsg = "${slackmsg}" + "\n" +
		"charts index Download URL: http://charts.pingcap.org/index.yaml"

		slackSend channel: '#cloud_jenkins', color: 'good', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
	}
}

return this
