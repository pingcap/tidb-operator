// Copyright 2019 PingCAP, Inc.
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
// limitations under the License.package spec

package tidbcluster

import (
	"context"
	"encoding/json"
	"fmt"
	_ "net/http/pprof"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	utilstatefulset "github.com/pingcap/tidb-operator/tests/e2e/util/statefulset"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/kubernetes/test/e2e/framework"
	e2esset "k8s.io/kubernetes/test/e2e/framework/statefulset"
)

func mustToString(set sets.Int32) string {
	b, err := json.Marshal(set.List())
	if err != nil {
		panic(err)
	}
	return string(b)
}

var _ = ginkgo.Describe("[tidb-operator][Serial]", func() {
	f := framework.NewDefaultFramework("serial")

	var ns string
	var c clientset.Interface
	var cli versioned.Interface
	var asCli asclientset.Interface
	var hc clientset.Interface
	var cfg *tests.Config
	var config *restclient.Config
	var fw portforward.PortForward
	var fwCancel context.CancelFunc

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		c = f.ClientSet
		var err error
		config, err = framework.LoadConfig()
		framework.ExpectNoError(err, "failed to load config")
		cli, err = versioned.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		asCli, err = asclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		clientRawConfig, err := e2econfig.LoadClientRawConfig()
		framework.ExpectNoError(err, "failed to load raw config")
		hc = helper.NewHijackClient(c, asCli)
		ctx, cancel := context.WithCancel(context.Background())
		fw, err = portforward.NewPortForwarder(ctx, e2econfig.NewSimpleRESTClientGetter(clientRawConfig))
		framework.ExpectNoError(err, "failed to create port forwarder")
		fwCancel = cancel
		cfg = e2econfig.TestConfig
	})

	ginkgo.AfterEach(func() {
		if fwCancel != nil {
			fwCancel()
		}
	})

	// tidb-operator with AdvancedStatefulSet feature enabled
	ginkgo.Context("[Feature: AdvancedStatefulSet]", func() {
		var ocfg *tests.OperatorConfig
		var oa tests.OperatorActions

		ginkgo.BeforeEach(func() {
			ocfg = &tests.OperatorConfig{
				Namespace:      "pingcap",
				ReleaseName:    "operator",
				Image:          cfg.OperatorImage,
				Tag:            cfg.OperatorTag,
				SchedulerImage: "k8s.gcr.io/kube-scheduler",
				Features: []string{
					"StableScheduling=true",
					"AdvancedStatefulSet=true",
				},
				LogLevel:        "4",
				ImagePullPolicy: v1.PullIfNotPresent,
				TestMode:        true,
			}
			oa = tests.NewOperatorActions(cli, c, asCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.InstallCRDOrDie()
			ginkgo.By("Installing tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			oa.DeployOperatorOrDie(ocfg)
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Uninstalling CRDs")
			oa.CleanCRDOrDie()
			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
		})

		ginkgo.It("able to deploy TiDB Cluster with advanced statefulset", func() {
			clusterName := "deploy"
			cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, clusterName, "", "")
			cluster.Resources["pd.replicas"] = "3"
			cluster.Resources["tikv.replicas"] = "5"
			cluster.Resources["tidb.replicas"] = "3"
			oa.DeployTidbClusterOrDie(&cluster)
			oa.CheckTidbClusterStatusOrDie(&cluster)

			scalingTests := []struct {
				name        string
				component   string // tikv,pd,tidb
				replicas    int32
				deleteSlots sets.Int32
			}{
				{
					name:        "Scaling in tikv from 5 to 3 by deleting pods 1 and 3",
					component:   "tikv",
					replicas:    3,
					deleteSlots: sets.NewInt32(1, 3),
				},
				{
					name:        "Scaling out tikv from 3 to 4 by adding pod 3",
					component:   "tikv",
					replicas:    4,
					deleteSlots: sets.NewInt32(1),
				},
				{
					name:        "Scaling tikv by adding pod 1 and deleting pod 2",
					component:   "tikv",
					replicas:    4,
					deleteSlots: sets.NewInt32(2),
				},
				{
					name:        "Scaling in tidb from 3 to 2 by deleting pod 1",
					component:   "tidb",
					replicas:    2,
					deleteSlots: sets.NewInt32(1),
				},
				{
					name:        "Scaling out tidb from 2 to 4 by adding pods 3 and 4",
					component:   "tidb",
					replicas:    4,
					deleteSlots: sets.NewInt32(1),
				},
				{
					name:        "Scaling out pd from 3 to 5 by adding pods 3, 4",
					component:   "pd",
					replicas:    5,
					deleteSlots: sets.NewInt32(),
				},
				{
					name:        "Scaling in pd from 5 to 3 by deleting pods 0 and 3",
					component:   "pd",
					replicas:    3,
					deleteSlots: sets.NewInt32(0, 3),
				},
			}

			for _, st := range scalingTests {
				ginkgo.By(st.name)
				replicas := st.replicas
				stsName := fmt.Sprintf("%s-%s", clusterName, st.component)

				sts, err := hc.AppsV1().StatefulSets(ns).Get(stsName, metav1.GetOptions{})
				framework.ExpectNoError(err)

				oldPodList := e2esset.GetPodList(c, sts)

				ginkgo.By(fmt.Sprintf("Scaling sts %s/%s to replicas %d and setting deleting pods to %v (old replicas: %d, old delete slots: %v)", ns, stsName, replicas, st.deleteSlots.List(), *sts.Spec.Replicas, helper.GetDeleteSlots(sts).List()))
				tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
				framework.ExpectNoError(err)
				if tc.Annotations == nil {
					tc.Annotations = map[string]string{}
				}
				if st.component == "tikv" {
					tc.Annotations[label.AnnTiKVDeleteSlots] = mustToString(st.deleteSlots)
					tc.Spec.TiKV.Replicas = replicas
				} else if st.component == "pd" {
					tc.Annotations[label.AnnPDDeleteSlots] = mustToString(st.deleteSlots)
					tc.Spec.PD.Replicas = replicas
				} else if st.component == "tidb" {
					tc.Annotations[label.AnnTiDBDeleteSlots] = mustToString(st.deleteSlots)
					tc.Spec.TiDB.Replicas = replicas
				} else {
					framework.Failf("unsupported component: %v", st.component)
				}
				tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(tc)
				framework.ExpectNoError(err)
				utilstatefulset.WaitForStatusReplicas(hc, sts, replicas)

				maxReplicaCount, deleteSlots := helper.GetMaxReplicaCountAndDeleteSlots(st.replicas, st.deleteSlots)
				ginkgo.By(fmt.Sprintf("Waiting for all pods of tidb cluster component %s (sts: %s/%s) are ready (maxReplicaCount: %d, delete slots: %v)", st.component, ns, stsName, maxReplicaCount, deleteSlots.List()))
				// workaround solution for https://github.com/pingcap/advanced-statefulset/issues/54
				err = wait.PollImmediate(time.Second, time.Minute*5, func() (bool, error) {
					for i := int32(0); i < maxReplicaCount; i++ {
						pod, err := c.CoreV1().Pods(ns).Get(fmt.Sprintf("%s-%d", stsName, i), metav1.GetOptions{})
						if err != nil && !apierrors.IsNotFound(err) {
							return false, nil
						}
						found := err == nil
						if deleteSlots.Has(i) && found {
							// pod expects to be absent but exists
							return false, nil
						} else if !deleteSlots.Has(i) {
							// pod expects to be present and ready but does not exist or is not ready
							if !found {
								return false, nil
							}
							return podutil.IsPodReady(pod), nil
						}
					}
					return true, nil
				})

				ginkgo.By(fmt.Sprintf("Verify delete slots of sts %s/%s is %v", ns, stsName, st.deleteSlots.List()))
				sts, err = hc.AppsV1().StatefulSets(ns).Get(stsName, metav1.GetOptions{})
				framework.ExpectNoError(err)
				framework.ExpectEqual(helper.GetDeleteSlots(sts), st.deleteSlots)

				ginkgo.By(fmt.Sprintf("Verify other pods of sts %s/%s should not be affected", ns, stsName))
				newPodList := e2esset.GetPodList(c, sts)
				framework.ExpectEqual(len(newPodList.Items), int(*sts.Spec.Replicas))
				for _, newPod := range newPodList.Items {
					for _, oldPod := range oldPodList.Items {
						// if the pod is not new or deleted in scaling, it should not be affected
						if oldPod.Name == newPod.Name && oldPod.UID != newPod.UID {
							framework.Failf("pod %s/%s should not be affected (UID: %s, OLD UID: %s)", newPod.Namespace, newPod.Name, newPod.UID, oldPod.UID)
						}
					}
				}
			}

			oa.CheckTidbClusterStatusOrDie(&cluster)
		})
	})

})
