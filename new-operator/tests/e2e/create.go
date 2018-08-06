// Copyright 2018 PingCAP, Inc.
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

package e2e

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util/label"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

func testCreate() {
	By("All members should running")
	err := wait.Poll(5*time.Second, 5*time.Minute, allRunning)
	Expect(err).NotTo(HaveOccurred())
}

func allRunning() (bool, error) {
	tc, err := cli.PingcapV1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		return false, nil
	}

	running, err := pdMemberRunning(tc)
	if err != nil || !running {
		return false, nil
	}

	running, err = tikvMemberRunning(tc)
	if err != nil || !running {
		return false, nil
	}

	running, err = tidbMemberRunning(tc)
	if err != nil || !running {
		return false, nil
	}

	synced, err := reclaimPolicySynced(tc)
	if err != nil || !synced {
		return false, nil
	}

	// TODO meta information synced

	return true, nil
}

func pdMemberRunning(tc *v1.TidbCluster) (bool, error) {
	pdSetName := controller.PDMemberName(clusterName)
	pdSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(pdSetName, metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	logf("pdSet.Status: %+v", pdSet.Status)

	if tc.Status.PD.StatefulSet == nil {
		logf("tc.Status.PD.StatefulSet is nil")
		return false, nil
	}

	if pdSet.Status.ReadyReplicas != tc.Spec.PD.Replicas {
		logf("pdSet.Status.ReadyReplicas(%d) != %d",
			pdSet.Status.ReadyReplicas, tc.Spec.PD.Replicas)
		return false, nil
	}

	if len(tc.Status.PD.Members) != int(tc.Spec.PD.Replicas) {
		logf("tc.Status.PD.Members count(%d) != %d",
			len(tc.Status.PD.Members), tc.Spec.PD.Replicas)
		return false, nil
	}

	for _, member := range tc.Status.PD.Members {
		if !member.Health {
			logf("pd member(%s) is not health", member.ID)
			return false, nil
		}
	}

	_, err = kubeCli.CoreV1().Services(ns).Get(controller.PDMemberName(clusterName), metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}
	_, err = kubeCli.CoreV1().Services(ns).Get(controller.PDPeerMemberName(clusterName), metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	return true, nil
}

func tikvMemberRunning(tc *v1.TidbCluster) (bool, error) {
	tikvSetName := controller.TiKVMemberName(clusterName)

	tikvSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(tikvSetName, metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	logf("tikvSet.Status: %+v", tikvSet.Status)

	if tc.Status.TiKV.StatefulSet == nil {
		logf("tc.Status.TiKV.StatefulSet is nil")
		return false, nil
	}

	if tikvSet.Status.ReadyReplicas != tc.Spec.TiKV.Replicas {
		logf("tikvSet.Status.ReadyReplicas(%d) != %d",
			tikvSet.Status.ReadyReplicas, tc.Spec.TiKV.Replicas)
		return false, nil
	}

	if len(tc.Status.TiKV.Stores.CurrentStores) != int(tc.Spec.TiKV.Replicas) {
		logf("tc.Status.TiKV.Stores.CurrentStores count(%d) != %d",
			len(tc.Status.TiKV.Stores.CurrentStores), tc.Spec.TiKV.Replicas)
		return false, nil
	}

	for _, store := range tc.Status.TiKV.Stores.CurrentStores {
		if store.State != util.StoreUpState {
			logf("store(%s) state != %s", store.ID, util.StoreUpState)
			return false, nil
		}
	}

	_, err = kubeCli.CoreV1().Services(ns).Get(controller.TiKVPeerMemberName(clusterName), metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	return true, nil
}

func tidbMemberRunning(tc *v1.TidbCluster) (bool, error) {
	tidbSetName := controller.TiDBMemberName(clusterName)
	tidbSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(tidbSetName, metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	logf("tidbSet.Status: %+v", tidbSet.Status)

	if tc.Status.TiDB.StatefulSet == nil {
		logf("tc.Status.TiDB.StatefulSet is nil")
		return false, nil
	}

	if tidbSet.Status.ReadyReplicas != tc.Spec.TiDB.Replicas {
		logf("tidbSet.Status.ReadyReplicas(%d) != %d",
			tidbSet.Status.ReadyReplicas, tc.Spec.TiDB.Replicas)
		return false, nil
	}

	_, err = kubeCli.CoreV1().Services(ns).Get(controller.TiDBMemberName(clusterName), metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	return true, nil
}

func reclaimPolicySynced(tc *v1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	labelSelector := label.New().Cluster(tcName)
	pvcList, err := kubeCli.CoreV1().PersistentVolumeClaims(ns).List(
		metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(
				labelSelector.Labels(),
			).String(),
		},
	)
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	for _, pvc := range pvcList.Items {
		pv, err := kubeCli.CoreV1().PersistentVolumes().Get(pvc.Spec.VolumeName, metav1.GetOptions{})
		if err != nil {
			logf(err.Error())
			return false, nil
		}

		logf("pv: %s's persistentVolumeReclaimPolicy is %s", pv.GetName(), pv.Spec.PersistentVolumeReclaimPolicy)
		if pv.Spec.PersistentVolumeReclaimPolicy != tc.Spec.PVReclaimPolicy {
			return false, nil
		}
	}

	return true, nil
}
