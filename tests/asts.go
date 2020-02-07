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

package tests

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	utilstatefulset "github.com/pingcap/tidb-operator/tests/e2e/util/statefulset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/kubernetes/test/e2e/framework"
	e2esset "k8s.io/kubernetes/test/e2e/framework/statefulset"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AstsCase struct {
	Name        string
	Component   string // tikv,pd,tidb
	Replicas    int32
	DeleteSlots sets.Int32
}

func GetAstsCases() []AstsCase {
	return []AstsCase{
		{
			Name:        "Scaling in tikv from 5 to 3 by deleting pods 1 and 3",
			Component:   "tikv",
			Replicas:    3,
			DeleteSlots: sets.NewInt32(1, 3),
		},
		{
			Name:        "Scaling out tikv from 3 to 4 by adding pod 3",
			Component:   "tikv",
			Replicas:    4,
			DeleteSlots: sets.NewInt32(1),
		},
		{
			Name:        "Scaling tikv by adding pod 1 and deleting pod 2",
			Component:   "tikv",
			Replicas:    4,
			DeleteSlots: sets.NewInt32(2),
		},
		{
			Name:        "Scaling in tidb from 3 to 2 by deleting pod 1",
			Component:   "tidb",
			Replicas:    2,
			DeleteSlots: sets.NewInt32(1),
		},
		{
			Name:        "Scaling in tidb from 2 to 0",
			Component:   "tidb",
			Replicas:    0,
			DeleteSlots: sets.NewInt32(),
		},
		{
			Name:        "Scaling out tidb from 0 to 2 by adding pods 0 and 2",
			Component:   "tidb",
			Replicas:    2,
			DeleteSlots: sets.NewInt32(1),
		},
		{
			Name:        "Scaling tidb from 2 to 4 by deleting pods 2 and adding pods 3, 4 and 5",
			Component:   "tidb",
			Replicas:    4,
			DeleteSlots: sets.NewInt32(1, 2),
		},
		{
			Name:        "Scaling out pd from 3 to 5 by adding pods 3, 4",
			Component:   "pd",
			Replicas:    5,
			DeleteSlots: sets.NewInt32(),
		},
		{
			Name:        "Scaling in pd from 5 to 3 by deleting pods 0 and 3",
			Component:   "pd",
			Replicas:    3,
			DeleteSlots: sets.NewInt32(0, 3),
		},
		{
			Name:        "Scaling out pd from 3 to 5 by adding pods 5 and 6",
			Component:   "pd",
			Replicas:    5,
			DeleteSlots: sets.NewInt32(0, 3),
		},
	}
}

func AstsScalingTest(st AstsCase, clusterName, ns string, c, hc clientset.Interface, cli versioned.Interface, genericCli client.Client) {
	ginkgo.By(st.Name)
	replicas := st.Replicas
	stsName := fmt.Sprintf("%s-%s", clusterName, st.Component)

	sts, err := hc.AppsV1().StatefulSets(ns).Get(stsName, metav1.GetOptions{})
	framework.ExpectNoError(err)

	oldPodList := e2esset.GetPodList(c, sts)

	ginkgo.By(fmt.Sprintf("Scaling sts %s/%s to replicas %d and setting deleting pods to %v (old replicas: %d, old delete slots: %v)", ns, stsName, replicas, st.DeleteSlots.List(), *sts.Spec.Replicas, helper.GetDeleteSlots(sts).List()))
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	framework.ExpectNoError(err)
	err = controller.GuaranteedUpdate(genericCli, tc, func() error {
		if tc.Annotations == nil {
			tc.Annotations = map[string]string{}
		}
		if st.Component == "tikv" {
			tc.Annotations[label.AnnTiKVDeleteSlots] = mustToString(st.DeleteSlots)
			tc.Spec.TiKV.Replicas = replicas
		} else if st.Component == "pd" {
			tc.Annotations[label.AnnPDDeleteSlots] = mustToString(st.DeleteSlots)
			tc.Spec.PD.Replicas = replicas
		} else if st.Component == "tidb" {
			tc.Annotations[label.AnnTiDBDeleteSlots] = mustToString(st.DeleteSlots)
			tc.Spec.TiDB.Replicas = replicas
		} else {
			return fmt.Errorf("unsupported component: %v", st.Component)
		}
		return nil
	})
	framework.ExpectNoError(err)

	ginkgo.By(fmt.Sprintf("Waiting for all pods of tidb cluster component %s (sts: %s/%s) are in desired state (replicas: %d, delete slots: %v)", st.Component, ns, stsName, st.Replicas, st.DeleteSlots.List()))
	err = wait.PollImmediate(time.Second*5, time.Minute*20, func() (bool, error) {
		// check delete slots annotation
		sts, err = hc.AppsV1().StatefulSets(ns).Get(stsName, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		if !helper.GetDeleteSlots(sts).Equal(st.DeleteSlots) {
			klog.Infof("delete slots of sts %s/%s is %v, expects: %v", ns, stsName, helper.GetDeleteSlots(sts).List(), st.DeleteSlots.List())
			return false, nil
		}
		// check all pod ordinals
		actualPodList := e2esset.GetPodList(c, sts)
		actualPodOrdinals := sets.NewInt32()
		for _, pod := range actualPodList.Items {
			actualPodOrdinals.Insert(int32(utilstatefulset.GetStatefulPodOrdinal(pod.Name)))
		}
		desiredPodOrdinals := helper.GetPodOrdinalsFromReplicasAndDeleteSlots(st.Replicas, st.DeleteSlots)
		if !actualPodOrdinals.Equal(desiredPodOrdinals) {
			klog.Infof("pod ordinals of sts %s/%s is %v, expects: %v", ns, stsName, actualPodOrdinals.List(), desiredPodOrdinals.List())
			return false, nil
		}
		for _, pod := range actualPodList.Items {
			if !podutil.IsPodReady(&pod) {
				klog.Infof("pod %s of sts %s/%s is not ready, got: %v", pod.Name, ns, stsName, podutil.GetPodReadyCondition(pod.Status))
				return false, nil
			}
		}
		return true, nil
	})
	framework.ExpectNoError(err)

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

func mustToString(set sets.Int32) string {
	b, err := json.Marshal(set.List())
	if err != nil {
		panic(err)
	}
	return string(b)
}
