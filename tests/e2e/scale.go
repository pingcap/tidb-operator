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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo" // revive:disable:dot-imports
	. "github.com/onsi/gomega" // revive:disable:dot-imports
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	pdScaleOutTo   = 5
	tikvScaleOutTo = 5
	tidbScaleOutTo = 3
	pdScaleInTo    = 3
	tikvScaleInTo  = 3
	tidbScaleInTo  = 2
)

var podUIDsBeforeScale map[string]types.UID

func testScale() {
	By(fmt.Sprintf("When scale out TiDB cluster: pd ==> [%d], tikv ==> [%d], tidb ==> [%d]", pdScaleOutTo, tikvScaleOutTo, tidbScaleOutTo))
	err := wait.Poll(5*time.Second, 5*time.Minute, scaleOut)
	Expect(err).NotTo(HaveOccurred())

	By("Then TiDB cluster should scale out successfully")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaled)
	Expect(err).NotTo(HaveOccurred())

	By("And should scaled out correctly")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaledCorrectly)
	Expect(err).NotTo(HaveOccurred())

	By("And the data is correct")
	err = wait.Poll(5*time.Second, 5*time.Minute, dataIsCorrect)
	Expect(err).NotTo(HaveOccurred())

	By(fmt.Sprintf("When scale in TiDB cluster: pd ==> [%d], tikv ==> [%d], tidb ==> [%d]", pdScaleInTo, tikvScaleInTo, tidbScaleInTo))
	err = wait.Poll(5*time.Second, 5*time.Minute, scaleIn)
	Expect(err).NotTo(HaveOccurred())

	By("Then TiDB cluster scale in securely")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaleInSafely)
	Expect(err).NotTo(HaveOccurred())

	By("And should scale in successfully")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaled)
	Expect(err).NotTo(HaveOccurred())

	By("And should be scaled in correctly")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaledCorrectly)
	Expect(err).NotTo(HaveOccurred())

	By("And the data is correct")
	err = wait.Poll(5*time.Second, 5*time.Minute, dataIsCorrect)
	Expect(err).NotTo(HaveOccurred())

	By(fmt.Sprintf("When scale out TiDB cluster one more time: pd ==> [%d], tikv ==> [%d], tidb ==> [%d]", pdScaleOutTo, tikvScaleOutTo, tidbScaleOutTo))
	err = wait.Poll(5*time.Second, 5*time.Minute, scaleOut)
	Expect(err).NotTo(HaveOccurred())

	By("Then TiDB cluster should scale out successfully")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaled)
	Expect(err).NotTo(HaveOccurred())

	By("And should scaled out correctly")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaledCorrectly)
	Expect(err).NotTo(HaveOccurred())

	By("And the data is correct")
	err = wait.Poll(5*time.Second, 5*time.Minute, dataIsCorrect)
	Expect(err).NotTo(HaveOccurred())

}

func scaleOut() (bool, error) {
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tidbcluster when scale out tidbcluster, error: %v", err)
		return false, nil
	}
	podUIDsBeforeScale, err = getPodsUID()
	if err != nil {
		return false, nil
	}

	tc.Spec.PD.Replicas = pdScaleOutTo
	tc.Spec.TiKV.Replicas = tikvScaleOutTo
	tc.Spec.TiDB.Replicas = tidbScaleOutTo

	tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(tc)
	if err != nil {
		logf("failed to update tidbcluster when scale out tidbcluster, error: %v", err)
		return false, nil
	}
	logf("Replicas after scaled out: PD: %d , TiKV: %d, TiDB: %d", tc.Spec.PD.Replicas, tc.Spec.TiKV.Replicas, tc.Spec.TiDB.Replicas)

	return true, nil
}

func scaled() (bool, error) {
	return allMembersRunning()
}

func scaleIn() (bool, error) {
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tidbcluster when scale in tidbcluster, error: %v", err)
		return false, nil
	}

	if tc.Spec.PD.Replicas <= pdScaleInTo {
		return true, fmt.Errorf("the tidbcluster's pd replicas less then pdScaleInTo: [%d]", pdScaleInTo)
	}
	if tc.Spec.TiKV.Replicas <= tikvScaleInTo {
		return true, fmt.Errorf("the tidbcluster's tikv replicas less then tikvScaleInTo: [%d]", tikvScaleInTo)
	}
	if tc.Spec.TiDB.Replicas <= tidbScaleInTo {
		return true, fmt.Errorf("the tidbcluster's tidb replicas less then tidbScaleInTo: [%d]", tidbScaleInTo)
	}

	podUIDsBeforeScale, err = getPodsUID()
	if err != nil {
		return false, nil
	}
	if err != nil {
		return false, nil
	}

	tc.Spec.PD.Replicas = pdScaleInTo
	tc.Spec.TiKV.Replicas = tikvScaleInTo
	tc.Spec.TiDB.Replicas = tidbScaleInTo

	tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(tc)
	if err != nil {
		logf("failed to update tidbcluster when scale in tidbcluster, error: %v", err)
		return false, nil
	}
	logf("Replicas after scaled in: PD: %d , TiKV: %d, TiDB: %d", tc.Spec.PD.Replicas, tc.Spec.TiKV.Replicas, tc.Spec.TiDB.Replicas)

	return true, nil
}

func scaledCorrectly() (bool, error) {
	podUIDs, err := getPodsUID()
	if err != nil {
		logf("failed to get pd pods's uid, error: %v", err)
		return false, nil
	}

	if len(podUIDsBeforeScale) == len(podUIDs) {
		return false, fmt.Errorf("the length of pods before scale equals the length of pods after scale")
	}

	for podName, uidAfter := range podUIDs {
		if uidBefore, ok := podUIDsBeforeScale[podName]; ok && uidBefore != uidAfter {
			return false, fmt.Errorf("pod: [%s] have be recreated", podName)
		}
	}

	return true, nil
}

// scaleInSafely confirms member scale in safely
func scaleInSafely() (bool, error) {
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tidbcluster when scale in tidbcluster, error: %v", err)
		return false, nil
	}

	tikvSetName := controller.TiKVMemberName(clusterName)
	tikvSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(tikvSetName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tikvSet statefulset: [%s], error: %v", tikvSetName, err)
		return false, nil
	}

	pdClient := controller.NewDefaultPDControl().GetPDClient(tc)
	stores, err := pdClient.GetStores()
	if err != nil {
		logf("pdClient.GetStores failed,error: %v", err)
		return false, nil
	}
	if len(stores.Stores) > int(*tikvSet.Spec.Replicas) {
		logf("stores.Stores: %v", stores.Stores)
		logf("tikvSet.Spec.Replicas: %d", *tikvSet.Spec.Replicas)
		return false, fmt.Errorf("the tikvSet.Spec.Replicas may reduce before tikv complete offline")
	}

	if *tikvSet.Spec.Replicas == tc.Spec.TiKV.Replicas {
		return true, nil
	}

	return false, nil
}

func getPodsUID() (map[string]types.UID, error) {
	result := map[string]types.UID{}

	selector, err := label.New().Cluster(clusterName).Selector()
	if err != nil {
		return nil, err
	}
	pods, err := kubeCli.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	for _, pod := range pods.Items {
		result[pod.GetName()] = pod.GetUID()
	}

	return result, nil
}
