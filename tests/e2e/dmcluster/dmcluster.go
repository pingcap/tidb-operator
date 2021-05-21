// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package dmcluster

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	utilpod "github.com/pingcap/tidb-operator/tests/e2e/util/pod"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/log"
	ctrlCli "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/tidbcluster"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	utiltc "github.com/pingcap/tidb-operator/tests/e2e/util/tidbcluster"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
)

var _ = ginkgo.Describe("DMCluster", func() {
	f := e2eframework.NewDefaultFramework("dm-cluster")

	var (
		dcName     string
		ns         string
		c          clientset.Interface
		config     *restclient.Config
		cli        versioned.Interface
		asCli      asclientset.Interface
		aggrCli    aggregatorclient.Interface
		apiExtCli  apiextensionsclientset.Interface
		genericCli ctrlCli.Client
		fwCancel   context.CancelFunc
		fw         portforward.PortForward
		cfg        *tests.Config
		ocfg       *tests.OperatorConfig
		oa         *tests.OperatorActions
	)

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		c = f.ClientSet
		var err error
		config, err = framework.LoadConfig()
		framework.ExpectNoError(err, "failed to load config")
		cli, err = versioned.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset for pingcap")
		asCli, err = asclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset for advanced-statefulset")
		aggrCli, err = aggregatorclient.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset for kube-aggregator")
		apiExtCli, err = apiextensionsclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset for apiextensions-apiserver")
		genericCli, err = ctrlCli.New(config, ctrlCli.Options{Scheme: scheme.Scheme})
		framework.ExpectNoError(err, "failed to create clientset for controller-runtime")
		ctx, cancel := context.WithCancel(context.Background())
		clientRawConfig, err := e2econfig.LoadClientRawConfig()
		framework.ExpectNoError(err, "failed to load raw config for tidb-operator")
		fw, err = portforward.NewPortForwarder(ctx, e2econfig.NewSimpleRESTClientGetter(clientRawConfig))
		framework.ExpectNoError(err, "failed to create port forwarder")
		fwCancel = cancel
		cfg = e2econfig.TestConfig
		ocfg = e2econfig.NewDefaultOperatorConfig(cfg)
		oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, cfg, nil, fw, f)
	})

	ginkgo.AfterEach(func() {
		if ginkgo.CurrentGinkgoTestDescription().Failed {
			// if the case failed, try to log out source and task status of DM.
			// NOTE: this can't work for the TLS case now.
			resp, err := tests.ShowDMSource(fw, ns, controller.DMMasterMemberName(dcName))
			if err != nil {
				log.Logf("failed to show sources for dc %s: %v", dcName, err)
			} else {
				log.Logf("sources for dc %s: %s", dcName, resp)
			}

			resp, err = tests.QueryDMStatus(fw, ns, controller.DMMasterMemberName(dcName))
			if err != nil {
				log.Logf("failed to query status for dc %s: %v", dcName, err)
			} else {
				log.Logf("status for dc %s: %s", dcName, resp)
			}
		}

		if fwCancel != nil {
			fwCancel()
		}
	})

	ginkgo.Context("[Feature:DM]", func() {
		ginkgo.It("setup replication for DM", func() {
			ginkgo.By("Deploy a basic dc")
			dcName = "basic-dm"
			dc := fixture.GetDMCluster(ns, dcName, utilimage.DMV2)
			dc.Spec.Master.Replicas = 1
			dc.Spec.Worker.Replicas = 1 // current versions of DM can always bind the first source to this only DM-worker instance.
			_, err := cli.PingcapV1alpha1().DMClusters(dc.Namespace).Create(dc)
			framework.ExpectNoError(err, "failed to create DmCluster: %q", dcName)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			ginkgo.By("Deploy TidbMonitor for DM")
			tc, err := cli.PingcapV1alpha1().TidbClusters(tests.DMTiDBNamespace).Get(tests.DMTiDBName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get TidbCluster for DM E2E tests")
			tm := fixture.NewTidbMonitor("monitor-dm-test", ns, tc, true, true, false)
			fixture.UpdateTidbMonitorForDM(tm, dc)
			_, err = cli.PingcapV1alpha1().TidbMonitors(ns).Create(tm)
			framework.ExpectNoError(err, "failed to deploy TidbMonitor for DmCluster %q", dcName)
			framework.ExpectNoError(tests.CheckTidbMonitor(tm, cli, c, fw), "failed to check TidbMonitor for DmCluster %q", dcName)

			ginkgo.By("Create MySQL sources")
			framework.ExpectNoError(tests.CreateDMSources(fw, dc.Namespace, controller.DMMasterMemberName(dcName)), "failed to create sources for DmCluster %q", dcName)

			ginkgo.By("Generate full stage data in upstream")
			framework.ExpectNoError(tests.GenDMFullData(fw, dc.Namespace), "failed to generate full stage data in upstream")

			ginkgo.By("Start a basic migration task")
			framework.ExpectNoError(tests.StartDMTask(fw, dc.Namespace, controller.DMMasterMemberName(dcName), tests.DMSingleTask, ""), "failed to start single source task")

			ginkgo.By("Check data for full stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 1), "failed to check full data")

			ginkgo.By("Generate incremental stage data in upstream")
			framework.ExpectNoError(tests.GenDMIncrData(fw, dc.Namespace), "failed to generate incremental stage data in upstream")

			ginkgo.By("Check data for incremental stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 1), "failed to check incremental data")
		})

		ginkgo.It("scale out with shard task for DM", func() {
			ginkgo.By("Deploy a basic dc")
			dcName = "scale-out-dm"
			dc := fixture.GetDMCluster(ns, dcName, utilimage.DMV2)
			dc.Spec.Master.Replicas = 3
			dc.Spec.Worker.Replicas = 1
			_, err := cli.PingcapV1alpha1().DMClusters(dc.Namespace).Create(dc)
			framework.ExpectNoError(err, "failed to create DmCluster: %q", dcName)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			ginkgo.By("Create MySQL sources")
			framework.ExpectNoError(tests.CreateDMSources(fw, dc.Namespace, controller.DMMasterMemberName(dcName)), "failed to create sources for DmCluster %q", dcName)

			ginkgo.By("Generate full stage data in upstream")
			framework.ExpectNoError(tests.GenDMFullData(fw, dc.Namespace), "failed to generate full stage data in upstream")

			ginkgo.By("Start a shard migration task but with one source not bound")
			framework.ExpectNoError(tests.StartDMTask(fw, dc.Namespace, controller.DMMasterMemberName(dcName), tests.DMShardTask, "have not bound"), "failed to start shard source task before scale out")

			ginkgo.By("Scale out DM-master and DM-worker")
			err = controller.GuaranteedUpdate(genericCli, dc, func() error {
				dc.Spec.Master.Replicas = 5
				dc.Spec.Worker.Replicas = 2
				return nil
			})
			framework.ExpectNoError(err, "failed to scale out DmCluster: %q", dc.Name)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			ginkgo.By("Start a shard migration task after scale out")
			framework.ExpectNoError(tests.StartDMTask(fw, dc.Namespace, controller.DMMasterMemberName(dcName), tests.DMShardTask, ""), "failed to start shard source task after scale out")

			ginkgo.By("Check data for full stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 2), "failed to check full data")

			ginkgo.By("Generate incremental stage data in upstream")
			framework.ExpectNoError(tests.GenDMIncrData(fw, dc.Namespace), "failed to generate incremental stage data in upstream")

			ginkgo.By("Check data for incremental stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 2), "failed to check incremental data")
		})

		ginkgo.It("scale in with shard task for DM", func() {
			ginkgo.By("Deploy a basic dc")
			dcName = "scale-in-dm"
			dc := fixture.GetDMCluster(ns, dcName, utilimage.DMV2)
			dc.Spec.Master.Replicas = 5
			dc.Spec.Worker.Replicas = 2
			_, err := cli.PingcapV1alpha1().DMClusters(dc.Namespace).Create(dc)
			framework.ExpectNoError(err, "failed to create DmCluster: %q", dcName)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			ginkgo.By("Create MySQL sources")
			framework.ExpectNoError(tests.CreateDMSources(fw, dc.Namespace, controller.DMMasterMemberName(dcName)), "failed to create sources for DmCluster %q", dcName)

			ginkgo.By("Generate full stage data in upstream")
			framework.ExpectNoError(tests.GenDMFullData(fw, dc.Namespace), "failed to generate full stage data in upstream")

			ginkgo.By("Start a shard migration task")
			framework.ExpectNoError(tests.StartDMTask(fw, dc.Namespace, controller.DMMasterMemberName(dcName), tests.DMShardTask, ""), "failed to start shard source task before scale in")

			ginkgo.By("Check data for full stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 2), "failed to check full data")

			ginkgo.By("Scale in DM-master and DM-worker")
			err = controller.GuaranteedUpdate(genericCli, dc, func() error {
				dc.Spec.Master.Replicas = 3
				dc.Spec.Worker.Replicas = 1
				return nil
			})
			framework.ExpectNoError(err, "failed to scale in DmCluster: %q", dc.Name)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			ginkgo.By("Generate incremental stage data in upstream")
			framework.ExpectNoError(tests.GenDMIncrData(fw, dc.Namespace), "failed to generate incremental stage data in upstream")

			ginkgo.By("Check data for incremental stage")
			framework.ExpectEqual(tests.CheckDMData(fw, dc.Namespace, 2), wait.ErrWaitTimeout, "incremental data should not equal")
		})

		ginkgo.It("restart pods for DM", func() {
			ginkgo.By("Deploy a basic dc")
			dcName = "restart-dm"
			dc := fixture.GetDMCluster(ns, dcName, utilimage.DMV2)
			dc.Spec.Master.Replicas = 3
			dc.Spec.Worker.Replicas = 1
			_, err := cli.PingcapV1alpha1().DMClusters(dc.Namespace).Create(dc)
			framework.ExpectNoError(err, "failed to create DmCluster: %q", dcName)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			ginkgo.By("Create MySQL sources")
			framework.ExpectNoError(tests.CreateDMSources(fw, dc.Namespace, controller.DMMasterMemberName(dcName)), "failed to create sources for DmCluster %q", dcName)

			ginkgo.By("Generate full stage data in upstream")
			framework.ExpectNoError(tests.GenDMFullData(fw, dc.Namespace), "failed to generate full stage data in upstream")

			ginkgo.By("Start a basic migration task")
			framework.ExpectNoError(tests.StartDMTask(fw, dc.Namespace, controller.DMMasterMemberName(dcName), tests.DMSingleTask, ""), "failed to start single source task")

			ginkgo.By("Check data for full stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 1), "failed to check full data")

			ginkgo.By("Restart DM-master pods one by one")
			for i := int32(0); i < dc.Spec.Master.Replicas; i++ {
				framework.ExpectNoError(c.CoreV1().Pods(ns).Delete(fmt.Sprintf("%s-%d", controller.DMMasterMemberName(dcName), i), &metav1.DeleteOptions{}), "failed to delete DM-master pod %d for DmCluster %q", i, dcName)
				<-time.After(time.Minute) // wait the previous pod to be deleted
				framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 5*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)
			}

			ginkgo.By("Restart the DM-worker pod")
			framework.ExpectNoError(c.CoreV1().Pods(ns).Delete(fmt.Sprintf("%s-0", controller.DMWorkerMemberName(dcName)), &metav1.DeleteOptions{}), "failed to delete the DM-worker pod for DmCluster %q", dcName)
			<-time.After(time.Minute) // wait the previous pod to be deleted
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 5*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			ginkgo.By("Generate incremental stage data in upstream")
			framework.ExpectNoError(tests.GenDMIncrData(fw, dc.Namespace), "failed to generate incremental stage data in upstream")

			ginkgo.By("Check data for incremental stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 1), "failed to check incremental data")
		})

		ginkgo.It("change config with dmctl for DM", func() {
			ginkgo.By("Deploy a basic dc")
			dcName = "change-config-dmctl"
			dc := fixture.GetDMCluster(ns, dcName, utilimage.DMV2)
			dc.Spec.Master.Replicas = 1
			dc.Spec.Worker.Replicas = 1
			_, err := cli.PingcapV1alpha1().DMClusters(dc.Namespace).Create(dc)
			framework.ExpectNoError(err, "failed to create DmCluster: %q", dcName)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			ginkgo.By("Create MySQL sources")
			framework.ExpectNoError(tests.CreateDMSources(fw, dc.Namespace, controller.DMMasterMemberName(dcName)), "failed to create sources for DmCluster %q", dcName)

			ginkgo.By("Check source config before updated")
			podName := fmt.Sprintf("%s-0", controller.DMMasterMemberName(dcName))
			sourceName := "replica-01"
			stdout, err := framework.RunHostCmd(ns, podName,
				fmt.Sprintf("/dmctl --master-addr=127.0.0.1:8261 get-config source %s", sourceName))
			framework.ExpectNoError(err, "failed to show source config for DmCluster %q: %v", dcName, err)
			framework.ExpectEqual(strings.Contains(stdout, "enable-gtid: true"), true, "original source config doesn't contain `enable-gtid: true`")
			framework.ExpectEqual(strings.Contains(stdout, "enable-relay: true"), true, "original source config doesn't contain `enable-relay: true`")

			ginkgo.By("Update and copy source config file")
			cfg := tests.DMSources[0]
			cfg = strings.ReplaceAll(cfg, "enable-gtid: true", "enable-gtid: false")
			cfg = strings.ReplaceAll(cfg, "enable-relay: true", "enable-relay: false")
			filename := "/tmp/change-config-dmctl-source.yaml"
			framework.ExpectNoError(ioutil.WriteFile(filename, []byte(cfg), 0o644), "failed to write updated source config file")
			_, err = framework.RunKubectl("cp", filename, fmt.Sprintf("%s/%s:%s", ns, podName, filename))
			framework.ExpectNoError(err, "failed to copy source file into dm-master pod")

			// directly update source config is not supported in DM now, so we choose to stop & re-create again.
			ginkgo.By("Stop MySQL source")
			_, err = framework.RunHostCmd(ns, podName,
				fmt.Sprintf("/dmctl --master-addr=127.0.0.1:8261 operate-source stop %s", filename))
			framework.ExpectNoError(err, "failed to stop source for DmCluster %q", dcName)

			ginkgo.By("Create MySQL source again")
			_, err = framework.RunHostCmd(ns, podName,
				fmt.Sprintf("/dmctl --master-addr=127.0.0.1:8261 operate-source create %s", filename))
			framework.ExpectNoError(err, "failed to create source again for DmCluster %q", dcName)

			ginkgo.By("Check source config after updated")
			stdout, err = framework.RunHostCmd(ns, podName,
				fmt.Sprintf("/dmctl --master-addr=127.0.0.1:8261 get-config source %s", sourceName))
			framework.ExpectNoError(err, "failed to show source config after updated for DmCluster %q: %v", dcName, err)
			framework.ExpectEqual(strings.Contains(stdout, "enable-gtid: false"), true, "original source config doesn't contain `enable-gtid: false`")
			framework.ExpectEqual(strings.Contains(stdout, "enable-relay: false"), true, "original source config doesn't contain `enable-relay: false`")
		})

		ginkgo.It("deploy DM with TLS enabled", func() {
			dcName = "tls-dm"

			ginkgo.By("Install CA certificate")
			framework.ExpectNoError(tidbcluster.InstallTiDBIssuer(ns, dcName), "failed to install CA certificate")

			ginkgo.By("Install MySQL certificate")
			framework.ExpectNoError(tidbcluster.InstallMySQLCertificates(ns, dcName), "failed to install MySQL certificate")
			<-time.After(30 * time.Second) // wait for secrets to be created

			ginkgo.By("Deploy MySQL with TLS enabled")
			mysqlSecretName := fmt.Sprintf("%s-mysql-secret", dcName)
			framework.ExpectNoError(tests.DeployDMMySQLWithTLSEnabled(c, ns, mysqlSecretName), "failed to deploy MySQL server with TLS enabled")

			ginkgo.By("Get MySQL TLS secret")
			mysqlSecret, err := c.CoreV1().Secrets(ns).Get(mysqlSecretName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get MySQL TLS secret")

			ginkgo.By("Check MySQL with TLS enabled ready")
			framework.ExpectNoError(tests.CheckDMMySQLReadyWithTLSEnabled(fw, ns, mysqlSecret), "failed to wait for MySQL ready")

			ginkgo.By("Install TiDB server and client certificate")
			framework.ExpectNoError(tidbcluster.InstallTiDBCertificates(ns, dcName), "failed to install TiDB server and client certificate")
			<-time.After(30 * time.Second)

			ginkgo.By("Deploy TiDB with client TLS enabled")
			tc := fixture.GetTidbCluster(ns, dcName, utilimage.TiDBV4)
			tc.Spec.PD.Replicas = 1
			tc.Spec.TiKV.Replicas = 1
			tc.Spec.TiDB.Replicas = 1
			tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

			ginkgo.By("Install DM components certificate")
			framework.ExpectNoError(tidbcluster.InstallDMCertificates(ns, dcName), "failed to install DM components certificate")
			<-time.After(30 * time.Second)

			ginkgo.By("Deploy a basic dc with TLS enabled")
			tidbClientSecretName := fmt.Sprintf("%s-tidb-client-secret", dcName)
			dc := fixture.GetDMCluster(ns, dcName, utilimage.DMV2)
			dc.Spec.Master.Replicas = 1
			dc.Spec.Worker.Replicas = 1
			dc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			dc.Spec.TLSClientSecretNames = []string{mysqlSecretName, tidbClientSecretName}
			_, err = cli.PingcapV1alpha1().DMClusters(dc.Namespace).Create(dc)
			framework.ExpectNoError(err, "failed to create DmCluster: %q", dcName)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			ginkgo.By("Deploy TidbMonitor for DM")
			tm := fixture.NewTidbMonitor("monitor-dm-test", ns, tc, true, true, false)
			fixture.UpdateTidbMonitorForDM(tm, dc)
			_, err = cli.PingcapV1alpha1().TidbMonitors(ns).Create(tm)
			framework.ExpectNoError(err, "failed to deploy TidbMonitor for DmCluster %q", dcName)
			framework.ExpectNoError(tests.CheckTidbMonitor(tm, cli, c, fw), "failed to check TidbMonitor for DmCluster %q", dcName)

			ginkgo.By("Create MySQL sources")
			podName := fmt.Sprintf("%s-0", controller.DMMasterMemberName(dcName))
			sourceCfg := tests.DMSources[0] + fmt.Sprintf(`
  security:
    ssl-ca: /var/lib/source-tls/%[1]s/ca.crt
    ssl-cert: /var/lib/source-tls/%[1]s/tls.crt
    ssl-key: /var/lib/source-tls/%[1]s/tls.key
`, mysqlSecretName)
			sourceCfg = strings.ReplaceAll(sourceCfg, "dm-mysql-0.dm-mysql.dm-mysql", "dm-mysql-0.dm-mysql")
			filename := "/tmp/dm-with-tls-source.yaml"
			framework.ExpectNoError(ioutil.WriteFile(filename, []byte(sourceCfg), 0o644), "failed to write source config file")
			_, err = framework.RunKubectl("cp", filename, fmt.Sprintf("%s/%s:%s", ns, podName, filename))
			framework.ExpectNoError(err, "failed to copy source file into dm-master pod")
			_, err = framework.RunHostCmd(ns, podName,
				fmt.Sprintf("/dmctl --master-addr=127.0.0.1:8261 --ssl-ca=%s --ssl-cert=%s --ssl-key=%s operate-source create %s",
					"/var/lib/dm-master-tls/ca.crt", // use the dm-master TLS certs
					"/var/lib/dm-master-tls/tls.crt",
					"/var/lib/dm-master-tls/tls.key",
					filename))
			framework.ExpectNoError(err, "failed to create source for DmCluster %q", dcName)

			ginkgo.By("Generate full stage data in upstream")
			framework.ExpectNoError(tests.GenDMFullDataWithMySQLNamespaceWithTLSEnabled(fw, ns, ns, mysqlSecret), "failed to generate full stage data in upstream")

			ginkgo.By("Start a basic migration task")
			taskCfg := tests.DMSingleTask
			taskCfg = strings.ReplaceAll(taskCfg, `
  password: ""
`, fmt.Sprintf(`
  password: ""
  security:
    ssl-ca: /var/lib/source-tls/%[1]s/ca.crt
    ssl-cert: /var/lib/source-tls/%[1]s/tls.crt
    ssl-key: /var/lib/source-tls/%[1]s/tls.key
`, tidbClientSecretName))
			taskCfg = strings.ReplaceAll(taskCfg, "dm-tidb-tidb.dm-tidb", fmt.Sprintf("%s-tidb", dcName))
			taskCfg = fmt.Sprintf(taskCfg, ns, ns)
			filename = "/tmp/dm-with-tls-task.yaml"
			framework.ExpectNoError(ioutil.WriteFile(filename, []byte(taskCfg), 0o644), "failed to write task config file")
			_, err = framework.RunKubectl("cp", filename, fmt.Sprintf("%s/%s:%s", ns, podName, filename))
			framework.ExpectNoError(err, "failed to copy task file into dm-master pod")
			_, err = framework.RunHostCmd(ns, podName,
				fmt.Sprintf("/dmctl --master-addr=127.0.0.1:8261 --ssl-ca=%s --ssl-cert=%s --ssl-key=%s start-task %s",
					"/var/lib/dm-master-tls/ca.crt", // use the dm-master TLS certs
					"/var/lib/dm-master-tls/tls.crt",
					"/var/lib/dm-master-tls/tls.key",
					filename))
			framework.ExpectNoError(err, "failed to start single source task for DmCluster %q", dcName)

			ginkgo.By("Get TiDB client TLS secret")
			tidbClientSecret, err := c.CoreV1().Secrets(ns).Get(tidbClientSecretName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get TiDB client TLS secret")

			ginkgo.By("Check data for full stage")
			framework.ExpectNoError(tests.CheckDMDataWithTLSEnabled(fw, ns, ns, ns, dcName, 1, mysqlSecret, tidbClientSecret), "failed to check full data")

			ginkgo.By("Generate incremental stage data in upstream")
			framework.ExpectNoError(tests.GenDMIncrDataWithMySQLNamespaceWithTLSEnabled(fw, ns, ns, mysqlSecret), "failed to generate incremental stage data in upstream")

			ginkgo.By("Check data for incremental stage")
			framework.ExpectNoError(tests.CheckDMDataWithTLSEnabled(fw, ns, ns, ns, dcName, 1, mysqlSecret, tidbClientSecret), "failed to check incremental data")
		})

		ginkgo.It("kill pods for DM", func() {
			ginkgo.By("Deploy a basic dc")
			dcName := "kill-dm"
			dc := fixture.GetDMCluster(ns, dcName, utilimage.DMV2)
			dc.Spec.Master.Replicas = 3
			dc.Spec.Worker.Replicas = 2
			_, err := cli.PingcapV1alpha1().DMClusters(dc.Namespace).Create(dc)
			framework.ExpectNoError(err, "failed to create DmCluster: %q", dcName)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			// TODO: check all masters are healthy; check all workers are bound (two source and two dm-worker replicas)

			ginkgo.By("Create MySQL sources")
			framework.ExpectNoError(tests.CreateDMSources(fw, dc.Namespace, controller.DMMasterMemberName(dcName)), "failed to create sources for DmCluster %q", dcName)

			ginkgo.By("Generate full stage data in upstream")
			framework.ExpectNoError(tests.GenDMFullData(fw, dc.Namespace), "failed to generate full stage data in upstream")

			ginkgo.By("Start a basic migration task")
			framework.ExpectNoError(tests.StartDMTask(fw, dc.Namespace, controller.DMMasterMemberName(dcName), tests.DMSingleTask, ""), "failed to start single source task")

			ginkgo.By("Check data for full stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 1), "failed to check full data")

			ginkgo.By("Random kill dm-master or dm-worker pods")
			var podNames []string
			for i := int32(0); i < dc.Spec.Master.Replicas; i++ {
				podNames = append(podNames, fmt.Sprintf("%s-%d", controller.DMMasterMemberName(dcName), i))
			}
			for i := int32(0); i < dc.Spec.Worker.Replicas; i++ {
				podNames = append(podNames, fmt.Sprintf("%s-%d", controller.DMWorkerMemberName(dcName), i))
			}
			rand.Shuffle(len(podNames), func(i, j int) {
				podNames[i], podNames[j] = podNames[j], podNames[i]
			})
			for _, podName := range podNames {
				pod, err := c.CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
				framework.ExpectNoError(err, "failed to get pod %q for DmCluster %q", podName, dcName)
				log.Logf("kill pod %s", podName)
				framework.ExpectNoError(c.CoreV1().Pods(ns).Delete(podName, &metav1.DeleteOptions{}), "failed to kill pod %q", podName)
				framework.ExpectNoError(utilpod.WaitForPodsAreChanged(c, []corev1.Pod{*pod}, 3*time.Minute))
				// TODO: check the killed pod become healthy
			}

			ginkgo.By("Generate incremental stage data in upstream")
			framework.ExpectNoError(tests.GenDMIncrData(fw, dc.Namespace), "failed to generate incremental stage data in upstream")

			ginkgo.By("Check data for incremental stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 1), "failed to check incremental data")
		})
	})
})
