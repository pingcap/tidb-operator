def replace(file, operatorImage, e2eImage, testApiserverImage) {
	def SRC_E2E_FILE_CONTENT = readFile file: file
	def DST_E2E_FILE_CONTENT = SRC_E2E_FILE_CONTENT.replaceAll("- --operator-image=.*", "- --operator-image=${operatorImage}")
	DST_E2E_FILE_CONTENT = DST_E2E_FILE_CONTENT.replaceAll("image:.*", "image: ${e2eImage}")
	DST_E2E_FILE_CONTENT = DST_E2E_FILE_CONTENT.replaceAll("- --test-apiserver-image=.*", "- --test-apiserver-image=${testApiserverImage}")
	DST_E2E_FILE_CONTENT = DST_E2E_FILE_CONTENT.replaceAll("imagePullPolicy: Always", "imagePullPolicy: IfNotPresent")
	writeFile file: file, text: "${DST_E2E_FILE_CONTENT}"
}

def replace_wh(file, operatorImage) {
	def SRC_WH_FILE_CONTENT = readFile file: file
	def DST_WH_FILE_CONTENT = SRC_WH_FILE_CONTENT.replaceAll("image:.*", "image: ${operatorImage}")
	writeFile file: file, text: "${DST_WH_FILE_CONTENT}"
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

	def GITHASH
	def UCLOUD_OSS_URL = "http://pingcap-dev.hk.ufileos.com"
	def BUILD_URL = "git@github.com:pingcap/tidb-operator.git"
	def PROJECT_DIR = "go/src/github.com/pingcap/tidb-operator"

	catchError {
		node('build_go1130_memvolume'){
			container("golang") {
				def WORKSPACE = pwd()
				dir("${PROJECT_DIR}"){
					stage('build tidb-operator binary'){
						checkout changelog: false, poll: false, scm: [$class: 'GitSCM', branches: [[name: "${BUILD_BRANCH}"]], doGenerateSubmoduleConfigurations: false, extensions: [], submoduleCfg: [], userRemoteConfigs: [[credentialsId: "${CREDENTIALS_ID}", refspec: '+refs/pull/*:refs/remotes/origin/pr/*', url: "${BUILD_URL}"]]]
						//git credentialsId: "k8s", url: "${BUILD_URL}", branch: "${ghprbActualCommit}"
						GITHASH = sh(returnStdout: true, script: "git rev-parse HEAD").trim()
						sh """
						export GOPATH=${WORKSPACE}/go
						export PATH=${WORKSPACE}/go/bin:\$PATH
						if ! hash hg 2>/dev/null; then
							sudo yum install -y mercurial
						fi
						hg --version
						make
						rm -rf tests/images/test-apiserver/bin
						make e2e-build
						"""
					}
				}
				stash excludes: "${PROJECT_DIR}/vendor/**", includes: "${PROJECT_DIR}/**", name: "tidb-operator"
			}
		}

		node('k8s-kind') {
			def WORKSPACE = pwd()
			def E2E_IMAGE = "localhost:5000/pingcap/tidb-operator-e2e:${GITHASH.take(7)}"
			def IMAGE_TAG = "localhost:5000/pingcap/tidb-operator:${GITHASH.take(7)}"
			def APISERVER_IMAGE = "localhost:5000/pingcap/test-apiserver:${GITHASH.take(7)}"
			deleteDir()
			unstash 'tidb-operator'

			dir("${PROJECT_DIR}"){
				stage('build tidb-operator image'){
					sh """
					docker build -t ${IMAGE_TAG} images/tidb-operator
					"""
				}

				stage('start prepare runtime environment'){
					def wh_yaml = "manifests/webhook.yaml"
					replace_wh(wh_yaml, IMAGE_TAG)
					sh """
					cp -r charts/tidb-operator tests/images/e2e
					cp -r charts/tidb-cluster tests/images/e2e
					cp -r charts/tidb-backup tests/images/e2e
					cp -r manifests tests/images/e2e
					docker build -t ${E2E_IMAGE} tests/images/e2e
					if [ -f tests/images/test-apiserver/bin/tidb-apiserver ]; then docker build -t ${APISERVER_IMAGE} tests/images/test-apiserver; fi
					"""
				}

				stage('start run operator e2e test'){
					def operator_yaml = "tests/manifests/e2e/e2e.yaml"
					replace(operator_yaml, IMAGE_TAG, E2E_IMAGE, APISERVER_IMAGE)
					ansiColor('xterm') {
					def ns = "tidb-operator-e2e"
					sh """#/usr/bin/env bash
					echo "BASH VERSION: \$BASH_VERSION"
					tidbOperatorImage="${IMAGE_TAG}"
					tidbOperatorE2EImage="${E2E_IMAGE}"
					tidbOperatorApiServerImage="${APISERVER_IMAGE}"
					declare -A clusters
					clusters['kind']='127.0.0.1:5000'
					clusters['kind2']='127.0.0.2:5000'
					clusters['kind3']='127.0.0.3:5000'
					clusters['kind4']='127.0.0.4:5000'
					while true
					do
						for cluster in \${!clusters[*]}; do
							lockfile=/k8s-locks/\$cluster
							if [ -e \$lockfile ]; then
								prId=`cat \$lockfile`
								if [ "${ghprbPullId}" == "\$prId" ]; then
									clusterName=\$cluster
									echo "####### branch ${ghprbPullId}/${BUILD_BRANCH} get env:\$cluster ########"
									echo "####### if you want to debug,please exec the command: export KUBECONFIG=`/root/go/bin/kind get kubeconfig-path --name=\$cluster ` in jenkins slave node ########"
										break 2
								fi
							fi
						done

						for cluster in \$(shuf -e \${!clusters[*]}); do
							lockfile=/k8s-locks/\$cluster
							if [ ! -e \$lockfile ]; then
									touch \$lockfile
									echo ${ghprbPullId} > \$lockfile
									clusterName=\$cluster
									echo "####### branch ${ghprbPullId}/${BUILD_BRANCH} get env:\$cluster #######"
									echo "####### if you want to debug,please exec the command: export KUBECONFIG=`/root/go/bin/kind get kubeconfig-path --name=\$cluster ` in jenkins slave node ########"
									break 2
							else
									prId=`cat \$lockfile`
									echo "env:\$cluster have be used by PR: \$prId"
							fi
						done
						echo "sleep 30 second and then retry"
						sleep 30
					done
					trap " echo '############ end of e2e test running on the cluster: \$clusterName ############'; rm -f /k8s-locks/\$clusterName " INT TERM EXIT

					tidbOperatorImage=\${tidbOperatorImage/localhost:5000/\${clusters[\$clusterName]}}
					tidbOperatorE2EImage=\${tidbOperatorE2EImage/localhost:5000/\${clusters[\$clusterName]}}
					tidbOperatorApiServerImage=\${tidbOperatorApiServerImage/localhost:5000/\${clusters[\$clusterName]}}
					echo "pushing tidb-operator image ${IMAGE_TAG} via \$tidbOperatorImage"
					docker tag ${IMAGE_TAG} \$tidbOperatorImage
					docker push \$tidbOperatorImage
					docker rmi -f ${IMAGE_TAG}
					docker rmi -f \$tidbOperatorImage
					echo "pushing tidb-operator e2e image ${E2E_IMAGE} via \$tidbOperatorE2EImage"
					docker tag ${E2E_IMAGE} \$tidbOperatorE2EImage
					docker push \$tidbOperatorE2EImage
					docker rmi -f ${E2E_IMAGE}
					docker rmi -f \$tidbOperatorE2EImage
					echo "pushing apiserver e2e image ${APISERVER_IMAGE} via \$tidbOperatorApiServerImage"
					if [ -f tests/images/test-apiserver/bin/tidb-apiserver ]; then
						docker tag ${APISERVER_IMAGE} \$tidbOperatorApiServerImage
						docker push \$tidbOperatorApiServerImage
						docker rmi -f ${APISERVER_IMAGE}
						docker rmi -f \$tidbOperatorApiServerImage
					fi
					export KUBECONFIG=`/root/go/bin/kind get kubeconfig-path --name="\$clusterName"`
					elapseTime=0
					period=5
					threshold=300

					kubectl create ns ${ns} || true
					kubectl delete validatingwebhookconfiguration --all
					sleep 5
					kubectl delete -f ${operator_yaml} --ignore-not-found
					kubectl wait --for=delete -n ${ns} pod/tidb-operator-e2e || true
					# Create sa first and wait for the controller to create the secret for this sa
					# This avoids `No API token found for service account " tidb-operator-e2e"`
					# TODO better way to work around this issue
					kubectl -n tidb-operator-e2e create sa tidb-operator-e2e
					kubectl apply -f ${operator_yaml}

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
							kubectl logs -f tidb-operator-e2e -n ${ns}
						else
							execType=0
							kubectl logs --tail 1 -f tidb-operator-e2e -n ${ns}
						fi
						## verify that the pod status is exit normally
						status=`kubectl get po tidb-operator-e2e -n ${ns} -ojsonpath='{.status.phase}'`
						if [[ \$status == Succeeded ]]
						then
							exit 0
						elif [[ \$status == Failed ]]
						then
							exit 1
						fi
						execType=1
					done
					trap - INT TERM ERR
					rm -f /k8s-locks/\$clusterName
					"""
					}
				}

				if ( BUILD_BRANCH == "master") {
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
		}
		currentBuild.result = "SUCCESS"
	}

	stage('Summary') {
		def CHANGELOG = getChangeLogText()
		def duration = ((System.currentTimeMillis() - currentBuild.startTimeInMillis) / 1000 / 60).setScale(2, BigDecimal.ROUND_HALF_UP)
		def slackmsg = "[#${ghprbPullId}: ${ghprbPullTitle}]" + "\n" +
		"${ghprbPullLink}" + "\n" +
		"${ghprbPullDescription}" + "\n" +
		"Integration Common Test Result: `${currentBuild.result}`" + "\n" +
		"Elapsed Time: `${duration} mins` " + "\n" +
		"${CHANGELOG}" + "\n" +
		"${env.RUN_DISPLAY_URL}"

		if (currentBuild.result != "SUCCESS") {
			slackSend channel: '#cloud_jenkins', color: 'danger', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
			return
		}

		if ( BUILD_BRANCH == "master" ){
			slackmsg = "${slackmsg}" + "\n" +
			"Binary Download URL:" + "\n" +
			"${UCLOUD_OSS_URL}/builds/pingcap/operator/${GITHASH}/centos7/tidb-operator.tar.gz"
		}

		slackSend channel: '#cloud_jenkins', color: 'good', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
	}
}
return this
