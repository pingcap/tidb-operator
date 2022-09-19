def call(BUILD_BRANCH, RELEASE_TAG, CREDENTIALS_ID) {
	def GITHASH
	def TKCTL_CLI_PACKAGE
	def GOARCH = "amd64"
	def OS_LIST = ["linux", "darwin", "windows"]
	def DEBUG_LIST = ["debug-launcher", "tidb-control", "tidb-debug"]
	def BUILD_URL = "git@github.com:pingcap/tidb-operator.git"
	def PROJECT_DIR = "go/src/github.com/pingcap/tidb-operator"

	catchError {
		def build_pod_label = "${JOB_NAME}-${BUILD_NUMBER}-build"
		podTemplate(label: build_pod_label,
				cloud: "kubernetes-ng",
				namespace: "jenkins-tidb-operator",
				idleMinutes: 0,
				containers: [
						containerTemplate(
							name: 'golang', alwaysPullImage: true,
							image: "hub.pingcap.net/jenkins/centos7_golang-1.16:latest", 
							ttyEnabled: true, privileged: true,
							resourceRequestCpu: '2000m', resourceRequestMemory: '4Gi',
							command: '/bin/sh -c', args: 'cat',
							envVars: [containerEnvVar(key: 'GOPATH', value: '/go')]     
						)
				],
				volumes: [
						emptyDirVolume(mountPath: '/home/jenkins', memory: true)
						],
		) {
        node(build_pod_label) {
			container("golang") {
				def WORKSPACE = pwd()
				dir("${PROJECT_DIR}") {
					stage('build tkcli') {
						checkout([$class: 'GitSCM', branches: [[name: "${BUILD_BRANCH}"]], userRemoteConfigs: [[url: "${BUILD_URL}", credentialsId: "${CREDENTIALS_ID}"]]])
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
						sh "make debug-build"

						OS_LIST.each {
							TKCTL_CLI_PACKAGE = "tkctl-${it}-${GOARCH}-${RELEASE_TAG}"
							sh """
							GOOS=${it} GOARCH=${GOARCH} make cli
							tar -zcf ${TKCTL_CLI_PACKAGE}.tgz tkctl
							sha256sum ${TKCTL_CLI_PACKAGE}.tgz > ${TKCTL_CLI_PACKAGE}.sha256
							"""
						}
					}
				}

				stash excludes: "${PROJECT_DIR}/vendor/**", includes: "${PROJECT_DIR}/**", name: "tidb-operator"
			}
		}
		}
		def delivery_pod_label = "${JOB_NAME}-${BUILD_NUMBER}"
		podTemplate(label: delivery_pod_label,
				cloud: "kubernetes-ng",
				namespace: "jenkins-tidb-operator",
				idleMinutes: 0,
				containers: [
						containerTemplate(
							name: 'delivery', alwaysPullImage: true,
							image: "hub.pingcap.net/jenkins/jenkins-slave-centos7:delivery3", ttyEnabled: true,
							resourceRequestCpu: '2000m', resourceRequestMemory: '4Gi',
							command: '/bin/sh -c', args: 'cat' 
						),
						containerTemplate(
							name: 'dind', alwaysPullImage: false,
							image: "docker:18.09-dind", ttyEnabled: true, privileged: true,
							resourceRequestCpu: '2000m', resourceRequestMemory: '4Gi',
							envVars: [containerEnvVar(key: 'DOCKER_HOST', value: 'tcp://localhost:2375')]    
						)
				],
		) {
		node(delivery_pod_label) {
			container("delivery") {
				def WORKSPACE = pwd()
				sh "chown -R jenkins:jenkins ./"
				deleteDir()
				unstash 'tidb-operator'

				dir("${PROJECT_DIR}") {
					stage('Publish tkcli') {
						OS_LIST.each {
							TKCTL_CLI_PACKAGE = "tkctl-${it}-${GOARCH}-${RELEASE_TAG}"
							sh """
							upload.py ${TKCTL_CLI_PACKAGE}.tgz ${TKCTL_CLI_PACKAGE}.tgz
							upload.py ${TKCTL_CLI_PACKAGE}.sha256 ${TKCTL_CLI_PACKAGE}.sha256
							"""
						}

						stage('Push utility docker images') {
							withDockerServer([uri: "${env.DOCKER_HOST}"]) {
								DEBUG_LIST.each {
									docker.build("pingcap/${it}:${RELEASE_TAG}", "misc/images/${it}").push()
                                    withDockerRegistry([url: "https://registry.cn-beijing.aliyuncs.com", credentialsId: "ACR_TIDB_ACCOUNT"]) {
                                        sh """
                                        docker tag pingcap/${it}:${RELEASE_TAG} registry.cn-beijing.aliyuncs.com/tidb/${it}:${RELEASE_TAG}
                                        docker push registry.cn-beijing.aliyuncs.com/tidb/${it}:${RELEASE_TAG}
                                        """
                                    }
								}
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

		DEBUG_LIST.each{
			slackmsg = "${slackmsg}" + "\n" +
			"${it} Docker Image: Docker Image: `pingcap/${it}:${RELEASE_TAG}`" + "\n" +
			"${it} Docker Image: Docker Image: `uhub.ucloud.cn/pingcap/${it}:${RELEASE_TAG}`"
		}

		OS_LIST.each{
			slackmsg = "${slackmsg}" + "\n" +
			"tkctl ${it} binary Download URL: https://download.pingcap.org/tkctl-${it}-${GOARCH}-${RELEASE_TAG}.tgz"
		}
		slackSend channel: '#cloud_jenkins', color: 'good', teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
	}
}

return this
