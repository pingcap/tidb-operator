//
// E2E Jenkins file.
//

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
						if( BUILD_BRANCH ==~ /[a-z0-9]{40}/ ){
							// checkout pull request
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
									refspec: '+refs/pull/*:refs/remotes/origin/pr/*',
									url: "${BUILD_URL}",
								]]
							]
						} else {
							// checkout branch, such as: master、release-1.0、release-1.1
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
									url: "${BUILD_URL}",
								]]
							]
						}
						//git credentialsId: "k8s", url: "${BUILD_URL}", branch: "${ghprbActualCommit}"
						GITHASH = sh(returnStdout: true, script: "git rev-parse HEAD").trim()
						withCredentials([string(credentialsId: "${CODECOV_CREDENTIALS_ID}", variable: 'codecovToken')]) {
							sh """
							export GOPATH=${WORKSPACE}/go
							export PATH=${WORKSPACE}/go/bin:\$PATH
							if ! hash hg 2>/dev/null; then
								sudo yum install -y mercurial
							fi
							hg --version
							make check-setup
							make check
							if [ ${BUILD_BRANCH} == "master" ]
							then
								make test GO_COVER=y
								curl -s https://codecov.io/bash | bash -s - -t ${codecovToken} || echo'Codecov did not collect coverage reports'
							else
								make test
							fi
							make
							make e2e-build
							"""
						}
					}
				}
				stash excludes: "${PROJECT_DIR}/vendor/**,${PROJECT_DIR}/deploy/**", includes: "${PROJECT_DIR}/**", name: "tidb-operator"
			}
		}

		node('k8s-kind') {
			deleteDir()
			unstash 'tidb-operator'

			dir("${PROJECT_DIR}"){
				stage('start run operator e2e test'){
					ansiColor('xterm') {
					sh """#/usr/bin/env bash
					echo "info: setup go"
					export PATH=\$PATH:/usr/local/go/bin
					echo "==== Start Build Environment ===="
					echo "BASH VERSION: \$BASH_VERSION"
					echo "PATH: \$PATH"
					echo "PROJECT_DIR: ${PROJECT_DIR}"
					if ! command -v go &>/dev/null; then
						echo "error: go is required"
						exit 1
					fi
					go version
					if ! command -v docker &>/dev/null; then
						echo "error: docker is required"
						exit 1
					fi
					docker version
					echo "==== End Build Environment ===="
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

                    export KUBECONFIG=`/root/go/bin/kind get kubeconfig-path --name="\$clusterName"`
                    DOCKER_REGISTRY=\${clusters[\$clusterName]} IMAGE_TAG=${GITHASH} SKIP_BUILD=y make e2e
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

// vim: noet
