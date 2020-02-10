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
	"k8s.io/kubernetes/test/e2e/framework"
	e2esset "k8s.io/kubernetes/test/e2e/framework/statefulset"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AstsCase struct {
	name        string
	component   string // tikv,pd,tidb
	replicas    int32
	deleteSlots sets.Int32
}

func GetAstsCases() []AstsCase {
	return []AstsCase{
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
			name:        "Scaling in tidb from 2 to 0",
			component:   "tidb",
			replicas:    0,
			deleteSlots: sets.NewInt32(),
		},
		{
			name:        "Scaling out tidb from 0 to 2 by adding pods 0 and 2",
			component:   "tidb",
			replicas:    2,
			deleteSlots: sets.NewInt32(1),
		},
		{
			name:        "Scaling tidb from 2 to 4 by deleting pods 2 and adding pods 3, 4 and 5",
			component:   "tidb",
			replicas:    4,
			deleteSlots: sets.NewInt32(1, 2),
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
		{
			name:        "Scaling out pd from 3 to 5 by adding pods 5 and 6",
			component:   "pd",
			replicas:    5,
			deleteSlots: sets.NewInt32(0, 3),
		},
	}
}

func AstsScalingTest(st AstsCase, clusterName, ns string, c, hc clientset.Interface, cli versioned.Interface, genericCli client.Client) {
	ginkgo.By(st.name)
	replicas := st.replicas
	stsName := fmt.Sprintf("%s-%s", clusterName, st.component)

	sts, err := hc.AppsV1().StatefulSets(ns).Get(stsName, metav1.GetOptions{})
	framework.ExpectNoError(err)

	oldPodList := e2esset.GetPodList(c, sts)

	ginkgo.By(fmt.Sprintf("Scaling sts %s/%s to replicas %d and setting deleting pods to %v (old replicas: %d, old delete slots: %v)", ns, stsName, replicas, st.deleteSlots.List(), *sts.Spec.Replicas, helper.GetDeleteSlots(sts).List()))
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	framework.ExpectNoError(err)
	err = controller.GuaranteedUpdate(genericCli, tc, func() error {
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
			return fmt.Errorf("unsupported component: %v", st.component)
		}
		return nil
	})
	framework.ExpectNoError(err)

	ginkgo.By(fmt.Sprintf("Waiting for all pods of tidb cluster component %s (sts: %s/%s) are in desired state (replicas: %d, delete slots: %v)", st.component, ns, stsName, st.replicas, st.deleteSlots.List()))
	err = wait.PollImmediate(time.Second*5, time.Minute*20, func() (bool, error) {
		// check replicas and delete slots are synced
		sts, err = hc.AppsV1().StatefulSets(ns).Get(stsName, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		if *sts.Spec.Replicas != st.replicas {
			klog.Infof("replicas of sts %s/%s is %d, expects %d", ns, stsName, sts.Spec.Replicas, st.replicas)
			return false, nil
		}
		if !helper.GetDeleteSlots(sts).Equal(st.deleteSlots) {
			klog.Infof("delete slots of sts %s/%s is %v, expects %v", ns, stsName, helper.GetDeleteSlots(sts).List(), st.deleteSlots.List())
			return false, nil
		}
		// check all desired pods are running and ready
		return utilstatefulset.IsAllDesiredPodsRunningAndReady(hc, sts), nil
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
