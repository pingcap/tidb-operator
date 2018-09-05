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
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // init mysql driver
	. "github.com/onsi/ginkgo"         // revive:disable:dot-imports
	. "github.com/onsi/gomega"         // revive:disable:dot-imports
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

func testCreate() {
	By("Then all members should running")
	err := wait.Poll(5*time.Second, 5*time.Minute, allMembersRunning)
	Expect(err).NotTo(HaveOccurred())

	By("When create a table and add some data to this table")
	err = wait.Poll(5*time.Second, 5*time.Minute, addDataToCluster)
	Expect(err).NotTo(HaveOccurred())

	By("Then the data is correct")
	err = wait.Poll(5*time.Second, 5*time.Minute, dataIsCorrect)
	Expect(err).NotTo(HaveOccurred())
}

func allMembersRunning() (bool, error) {
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
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

func addDataToCluster() (bool, error) {
	db, err := sql.Open("mysql", fmt.Sprintf("root:@(%s-tidb.%s:4000)/test?charset=utf8", clusterName, ns))
	if err != nil {
		logf("can't open connection to mysql: %v", err)
		return false, nil
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE TABLE %s (name VARCHAR(64))", testTableName))
	if err != nil {
		logf("can't create table to mysql: %v", err)
		return false, nil
	}

	_, err = db.Exec(fmt.Sprintf("INSERT INTO %s VALUES (?)", testTableName), testTableVal)
	if err != nil {
		logf("can't insert data to mysql: %v", err)
		return false, nil
	}

	return true, nil
}

func dataIsCorrect() (bool, error) {
	db, err := sql.Open("mysql", fmt.Sprintf("root:@(%s-tidb.%s:4000)/test?charset=utf8", clusterName, ns))
	if err != nil {
		return false, nil
	}

	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", testTableName))
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	for rows.Next() {
		var v string
		err := rows.Scan(&v)
		if err != nil {
			logf(err.Error())
		}

		if v == testTableVal {
			return true, nil
		}

		return true, fmt.Errorf("val should equal: %s", testTableVal)
	}

	return false, nil
}

func pdMemberRunning(tc *v1alpha1.TidbCluster) (bool, error) {
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

	if *pdSet.Spec.Replicas != tc.Spec.PD.Replicas {
		logf("pdSet.Spec.Replicas(%d) != tc.Spec.PD.Replicas(%d)",
			*pdSet.Spec.Replicas, tc.Spec.PD.Replicas)
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

	if pdSet.Status.ReadyReplicas != pdSet.Status.Replicas {
		logf("pdSet.Status.ReadyReplicas(%d) != pdSet.Status.Replicas(%d)",
			pdSet.Status.ReadyReplicas, pdSet.Status.Replicas)
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

func tikvMemberRunning(tc *v1alpha1.TidbCluster) (bool, error) {
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

	if *tikvSet.Spec.Replicas != tc.Spec.TiKV.Replicas {
		logf("tikvSet.Spec.Replicas(%d) != tc.Spec.TiKV.Replicas(%d)",
			*tikvSet.Spec.Replicas, tc.Spec.TiKV.Replicas)
		return false, nil
	}

	if tikvSet.Status.ReadyReplicas != tc.Spec.TiKV.Replicas {
		logf("tikvSet.Status.ReadyReplicas(%d) != %d",
			tikvSet.Status.ReadyReplicas, tc.Spec.TiKV.Replicas)
		return false, nil
	}

	if len(tc.Status.TiKV.Stores) != int(tc.Spec.TiKV.Replicas) {
		logf("tc.Status.TiKV.Stores.count(%d) != %d",
			len(tc.Status.TiKV.Stores), tc.Spec.TiKV.Replicas)
		return false, nil
	}

	if tikvSet.Status.ReadyReplicas != tikvSet.Status.Replicas {
		logf("tikvSet.Status.ReadyReplicas(%d) != tikvSet.Status.Replicas(%d)",
			tikvSet.Status.ReadyReplicas, tikvSet.Status.Replicas)
		return false, nil
	}

	for _, store := range tc.Status.TiKV.Stores {
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

func tidbMemberRunning(tc *v1alpha1.TidbCluster) (bool, error) {
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

	if *tidbSet.Spec.Replicas != tc.Spec.TiDB.Replicas {
		logf("tidbSet.Spec.Replicas(%d) != tc.Spec.TiDB.Replicas(%d)",
			*tidbSet.Spec.Replicas, tc.Spec.TiDB.Replicas)
		return false, nil
	}

	if tidbSet.Status.ReadyReplicas != tc.Spec.TiDB.Replicas {
		logf("tidbSet.Status.ReadyReplicas(%d) != %d",
			tidbSet.Status.ReadyReplicas, tc.Spec.TiDB.Replicas)
		return false, nil
	}

	if tidbSet.Status.ReadyReplicas != tidbSet.Status.Replicas {
		logf("tidbSet.Status.ReadyReplicas(%d) != tidbSet.Status.Replicas(%d)",
			tidbSet.Status.ReadyReplicas, tidbSet.Status.Replicas)
		return false, nil
	}

	_, err = kubeCli.CoreV1().Services(ns).Get(controller.TiDBMemberName(clusterName), metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	return true, nil
}

func reclaimPolicySynced(tc *v1alpha1.TidbCluster) (bool, error) {
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
