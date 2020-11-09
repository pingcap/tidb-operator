// Copyright 2020 PingCAP, Inc.
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
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	"gopkg.in/yaml.v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

var dmVersion string = "v2.0.0"
var tidbVersion string = "v4.0.8"

var _ = ginkgo.Describe("[tidb-operator] [DMCluster]", func() {
	f := e2eframework.NewDefaultFramework("dm-cluster")

	var discoveryAddr string
	var cli versioned.Interface
	var tcName string = "basic"
	var ns string

	var crdUtil *tests.CrdTestUtil
	var asCli asclientset.Interface
	var stsGetter typedappsv1.StatefulSetsGetter
	var tc *v1alpha1.TidbCluster
	var mysqlPod *corev1.Pod

	// TODO Setup 2 mysql with binlog for test

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		config, err := framework.LoadConfig()
		framework.ExpectNoError(err, "failed to load config")
		cli, err = versioned.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")

		asCli, err = asclientset.NewForConfig(config)
		framework.ExpectNoError(err)
		ocfg := e2econfig.NewDefaultOperatorConfig(e2econfig.TestConfig)
		if ocfg.Enabled(features.AdvancedStatefulSet) {
			stsGetter = helper.NewHijackClient(f.ClientSet, asCli).AppsV1()
		} else {
			stsGetter = f.ClientSet.AppsV1()
		}
		crdUtil = tests.NewCrdTestUtil(cli, f.ClientSet, asCli, stsGetter)

		// create mysql pod
		mysqlPod, err = createMysql(f, "mysql-0")
		framework.ExpectNoError(err, "failed to create mysql")
		err = e2epod.WaitForPodsReady(f.ClientSet, mysqlPod.Namespace, mysqlPod.Name, 0 /*minReadySeconds*/)
		framework.ExpectNoError(err, "failed to wait mysql ready")

		// create tidb cluster
		tc = fixture.GetTidbCluster(f.Namespace.Name, tcName, tidbVersion)
		ginkgo.By(fmt.Sprintf("Create tidbcluster: %s", tc.Name))
		tc, err = cli.PingcapV1alpha1().TidbClusters(f.Namespace.Name).Create(tc)
		framework.ExpectNoError(err)
		ginkgo.By("Waiting for tidb cluster ready")
		err = crdUtil.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
		framework.ExpectNoError(err, "failed to wait tidb up")
		// ref: https://docs.pingcap.com/zh/tidb-in-kubernetes/dev/deploy-tidb-dm#discovery-%E9%85%8D%E7%BD%AE
		// format: http://${tidb_cluster_name}-discovery.${tidb_namespace}:10261
		discoveryAddr = fmt.Sprintf("http://%s-discovery.%s:10261", tc.Name, tc.Namespace)
	})

	ginkgo.AfterEach(func() {
	})

	ginkgo.Context("Basic: Deploying, Scaling, Update Configuration DMCluster", func() {
		ginkgo.It("dummy test DMCluster", func() {
			var err error
			dc := newDMC("basic", dmVersion, discoveryAddr, 3, 3)

			ginkgo.By(fmt.Sprintf("Creating dmcluster %s", dc.Name))
			_, err = cli.PingcapV1alpha1().DMClusters(ns).Create(dc)
			framework.ExpectNoError(err)

			ginkgo.By("Checking dm-master pods created")
			label := labels.SelectorFromSet(labels.Set(map[string]string{"app.kubernetes.io/component": "dm-master"}))
			masterList, err := e2epod.PodsCreatedByLabel(
				f.ClientSet,
				ns,
				dc.Name+"-dm-master",
				3,
				label)
			framework.ExpectNoError(err)

			ginkgo.By("Checking dm-worker pods created")
			label = labels.SelectorFromSet(labels.Set(map[string]string{"app.kubernetes.io/component": "dm-worker"}))
			workerList, err := e2epod.PodsCreatedByLabel(
				f.ClientSet,
				ns,
				dc.Name+"-dm-worker",
				3,
				label)
			framework.ExpectNoError(err)

			// dc works now
			// 127.0.0.1:8291

			// TODO normally create/view/modify synchronization tasks through the embedded dm-ctl of image
			// 1. Add source.
			// 2. Add task.
			// 3. check replicate.

			// TODO test scale-in scale-out

			// TODO auto failover of dm cluster can be achieved

			// TODO whether the rolling upgrade of the dm cluster can be performed correctly, and the tasks can still work normally after the upgrade

			ginkgo.By(fmt.Sprintf("Deleting dmcluster %s", dc.Name))
			err = cli.PingcapV1alpha1().DMClusters(ns).Delete(dc.Name, nil)
			framework.ExpectNoError(err)
			ginkgo.By("Waiting pods to be disappear")
			for _, p := range workerList.Items {
				err = e2epod.WaitForPodNotFoundInNamespace(f.ClientSet, p.Name, ns, e2epod.PodDeleteTimeout)
				framework.ExpectNoError(err)
			}
			for _, p := range masterList.Items {
				err = e2epod.WaitForPodNotFoundInNamespace(f.ClientSet, p.Name, ns, e2epod.PodDeleteTimeout)
				framework.ExpectNoError(err)
			}
		})
	})

	return
})

func testTask(f *framework.Framework, dc *v1alpha1.DMCluster) {
	// create source and task
}

func createMysql(f *framework.Framework, name string) (pod *corev1.Pod, err error) {
	pod = &corev1.Pod{}

	var tpl = `
apiVersion: v1
kind: Pod
metadata:
  name: mysql1
  namespace: default
spec:
  containers:
  - env:
    - name: MYSQL_ALLOW_EMPTY_PASSWORD
      value: "true"
    - name: MYSQL_ROOT_PASSWORD
      value: ""
    image: mysql:5.7
    imagePullPolicy: IfNotPresent
    name: mysql
    args: ["--default-authentication-plugin=mysql_native_password", "--log-bin=/var/lib/mysql/mysql-bin.log", "--server-id=1", "--binlog-format=ROW", "--gtid_mode=ON", "--enforce-gtid-consistency=true"]
    ports:
    - containerPort: 3306
      name: mysql
      protocol: TCP
`

	err = yaml.Unmarshal([]byte(tpl), pod)
	if err != nil {
		return nil, err
	}

	pod.Name = name
	pod.Namespace = f.Namespace.Name
	pod, err = f.ClientSet.CoreV1().Pods(pod.Namespace).Create(pod)
	return
}

func newDMC(
	dcName string,
	dcVersion string,
	discoveryAddr string,
	masterReplicas int32,
	workerReplicas int32,
) *v1alpha1.DMCluster {
	dc := &v1alpha1.DMCluster{
		Spec: v1alpha1.DMClusterSpec{
			Version: dcVersion,
			// PVReclaimPolicy: nil,
			Discovery: v1alpha1.DMDiscoverySpec{
				Address: discoveryAddr,
			},
			Master: v1alpha1.MasterSpec{
				BaseImage: "pingcap/dm",
				Replicas:  masterReplicas,
			},
			Worker: &v1alpha1.WorkerSpec{
				BaseImage: "pingcap/dm",
				Replicas:  masterReplicas,
			},
		},
	}

	dc.Name = dcName

	return dc
}
