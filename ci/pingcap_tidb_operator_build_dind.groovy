def tidbClusterReplace(file) {
	def SRC_E2E_FILE_CONTENT = readFile file: file
	def DST_E2E_FILE_CONTENT = SRC_E2E_FILE_CONTENT.replaceAll("image:.*pingcap/pd", "image: localhost:5000/pingcap/pd")
	DST_E2E_FILE_CONTENT = DST_E2E_FILE_CONTENT.replaceAll("image:.*pingcap/tidb", "image: localhost:5000/pingcap/tidb")
	DST_E2E_FILE_CONTENT = DST_E2E_FILE_CONTENT.replaceAll("image:.*pingcap/tikv", "image: localhost:5000/pingcap/tikv")
	DST_E2E_FILE_CONTENT = DST_E2E_FILE_CONTENT.replaceAll("image:.*prom/pushgateway", "image: localhost:5000/pingcap/pushgateway")
	DST_E2E_FILE_CONTENT = DST_E2E_FILE_CONTENT.replaceAll("image:.*pingcap/tidb-dashboard-installer", "image: localhost:5000/pingcap/tidb-dashboard-installer")
	DST_E2E_FILE_CONTENT = DST_E2E_FILE_CONTENT.replaceAll("image:.*grafana/grafana", "image: localhost:5000/pingcap/grafana")
	DST_E2E_FILE_CONTENT = DST_E2E_FILE_CONTENT.replaceAll("image:.*prom/prometheus", "image: localhost:5000/pingcap/prometheus")
	writeFile file: file, text: "${DST_E2E_FILE_CONTENT}"
}

def operatorReplace(file, tag) {
	def SRC_E2E_FILE_CONTENT = readFile file: file
	def DST_E2E_FILE_CONTENT = SRC_E2E_FILE_CONTENT.replaceAll("operatorImage:.*", "operatorImage: ${tag}")
	writeFile file: file, text: "${DST_E2E_FILE_CONTENT}"
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

def call(BUILD_BRANCH, CREDENTIALS_ID) {
	env.GOROOT = "/usr/local/go"
	env.GOPATH = "/go"
	env.PATH = "/usr/local/bin:${env.GOROOT}/bin:${env.GOPATH}/bin:/bin:${env.PATH}:/home/jenkins/bin"

	def IMAGE_TAG
	def BACKUP_IMAGE_TAG
	def GITHASH
	def UCLOUD_OSS_URL = "http://pingcap-dev.hk.ufileos.com"
	def BUILD_URL = "git@github.com:pingcap/tidb-operator.git"
	def E2E_IMAGE = "localhost:5000/pingcap/tidb-operator-e2e:latest"
	def PROJECT_DIR = "go/src/github.com/pingcap/tidb-operator"

	catchError {
		node('k8s_centos7_build'){
			def WORKSPACE = pwd()

			dir("${PROJECT_DIR}"){
				stage('run unit test and build tidb-operator && e2e test binary'){
					checkout([$class: 'GitSCM', branches: [[name: "${BUILD_BRANCH}"]], userRemoteConfigs:[[url: "${BUILD_URL}", credentialsId: "${CREDENTIALS_ID}"]]])
					GITHASH = sh(returnStdout: true, script: "git rev-parse HEAD").trim()
					sh """
					export GOPATH=${WORKSPACE}/go:$GOPATH
					make check
					make test
					make
					make e2e-build
					"""
				}
			}
			stash excludes: "${PROJECT_DIR}/vendor/**", includes: "${PROJECT_DIR}/**", name: "tidb-operator"
		}

		node('k8s-dind') {
			def WORKSPACE = pwd()
			deleteDir()
			unstash 'tidb-operator'

			dir("${PROJECT_DIR}"){
				stage('push tidb-operator images'){
					IMAGE_TAG = "localhost:5000/pingcap/tidb-operator:${GITHASH.take(7)}"
					sh """
					docker build -t ${IMAGE_TAG} images/tidb-operator
					docker push ${IMAGE_TAG}
					"""
				}

				stage('start prepare runtime environment'){
					tidbClusterReplace("images/tidb-operator-e2e/tidb-cluster-values.yaml")
					operatorReplace("images/tidb-operator-e2e/tidb-operator-values.yaml", IMAGE_TAG)

					sh """
					mkdir -p images/tidb-operator-e2e/bin
					mv tests/e2e/e2e.test images/tidb-operator-e2e/bin
					cp -r charts/tidb-operator images/tidb-operator-e2e
					cp -r charts/tidb-cluster images/tidb-operator-e2e
					cat >images/tidb-operator-e2e/Dockerfile << __EOF__
FROM uhub.ucloud.cn/pingcap/base-e2e:latest

ADD bin/e2e.test /usr/local/bin/e2e.test
ADD tidb-operator /charts/tidb-operator
ADD tidb-cluster /charts/tidb-cluster
ADD tidb-cluster-values.yaml /tidb-cluster-values.yaml
ADD tidb-operator-values.yaml /tidb-operator-values.yaml
__EOF__
					docker build -t ${E2E_IMAGE} images/tidb-operator-e2e
					docker push ${E2E_IMAGE}
					"""
				}

				stage('start run operator e2e test'){
					def ns = "tidb-operator-e2e"
					def operator_yaml = "manifests/tidb-operator-e2e.yaml"

					ansiColor('xterm') {
					sh """
					elapseTime=0
					period=5
					threshold=300
					kubectl delete -f ${operator_yaml} || true
					sleep 5
					kubectl create -f ${operator_yaml}
					while true
					do
						sleep \$period
						elapseTime=\$(( elapseTime+\$period ))
						kubectl get po/tidb-operator-e2e -n ${ns} 2>/dev/null || continue
						kubectl get po/tidb-operator-e2e -n ${ns}|grep Running && break || true
						if [[ \$elapseTime -gt \$threshold ]]
						then
							echo "wait e2e pod timeout, elapseTime: \$elapseTime"
							exit 1
						fi
					done

					execType=0
					while true
					do
						if [[ \$execType -eq 0 ]]
						then
							kubectl logs -f tidb-operator-e2e -n ${ns}|tee -a result.log
						else
							execType=0
							kubectl logs --tail 1 -f tidb-operator-e2e -n ${ns}|tee -a result.log
						fi
						## verify that the log output is exit normally
						tail -5 result.log|egrep 'Test Suite Failed|Test Suite Passed'||execType=\$?
						[[ \$execType -eq 0 ]] && break
					done

					ret=0
					tail -1 result.log | grep 'Test Suite Passed' || ret=\$?
					rm result.log
					exit \$ret
					"""
					}
				}

				stage('upload tidb-operator binary and charts'){
					//upload binary and charts
					sh """
					cp ~/bin/config.cfg ./
					tar -zcvf tidb-operator.tar.gz images/tidb-operator charts
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

		currentBuild.result = "SUCCESS"
	}

	stage('Summary') {
		echo("echo summary info #########")
		def CHANGELOG = getChangeLogText()
		def DURATION = ((System.currentTimeMillis() - currentBuild.startTimeInMillis) / 1000 / 60).setScale(2, BigDecimal.ROUND_HALF_UP)
		def slackmsg = "[${env.JOB_NAME.replaceAll('%2F','/')}-${env.BUILD_NUMBER}] `${currentBuild.result}`" + "\n" +
		"`${BUILD_BRANCH} Build`" + "\n" +
		"Elapsed Time: `${DURATION}` Mins" + "\n" +
		"Build Branch: `${BUILD_BRANCH}`, Githash: `${GITHASH.take(7)}`" + "\n" +
		"${CHANGELOG}" + "\n" +
		"Display URL:" + "\n" +
		"${env.RUN_DISPLAY_URL}"


		if (currentBuild.result != "SUCCESS") {
			slackSend channel: '#cloud_jenkins', color: 'danger', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
		} else {
			slackmsg = "${slackmsg}" + "\n" +
			"Binary Download URL:" + "\n" +
			"${UCLOUD_OSS_URL}/builds/pingcap/operator/${GITHASH}/centos7/tidb-operator.tar.gz"
			slackSend channel: '#cloud_jenkins', color: 'good', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
		}
	}
}

return this
