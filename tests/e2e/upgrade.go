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
	"strings"
	"time"

	. "github.com/onsi/ginkgo" // revive:disable:dot-imports
	. "github.com/onsi/gomega" // revive:disable:dot-imports
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	upgradeVersion = "v2.1.0-rc.3"
)

func testUpgrade() {
	By("When upgrade TiDB cluster to newer version")
	err := wait.Poll(5*time.Second, 5*time.Minute, upgrade)
	Expect(err).NotTo(HaveOccurred())

	By("Then members should be upgrade in order: pd ==> tikv ==> tidb")
	err = wait.Poll(5*time.Second, 10*time.Minute, memberUpgraded)
	Expect(err).NotTo(HaveOccurred())

	By("Then all members should running")
	err = wait.Poll(5*time.Second, 5*time.Minute, allMembersRunning)
	Expect(err).NotTo(HaveOccurred())

	By("And the data is correct")
	err = wait.Poll(5*time.Second, 5*time.Minute, dataIsCorrect)
	Expect(err).NotTo(HaveOccurred())
}

func upgrade() (bool, error) {
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tidbcluster, error: %v", err)
		return false, nil
	}

	tc.Spec.PD.Image = strings.Replace(tc.Spec.PD.Image, getImageTag(tc.Spec.PD.Image), upgradeVersion, -1)
	tc.Spec.TiKV.Image = strings.Replace(tc.Spec.TiKV.Image, getImageTag(tc.Spec.TiKV.Image), upgradeVersion, -1)
	tc.Spec.TiDB.Image = strings.Replace(tc.Spec.TiDB.Image, getImageTag(tc.Spec.TiDB.Image), upgradeVersion, -1)

	tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(tc)
	if err != nil {
		logf("failed to update tidbcluster, error: %v", err)
		return false, nil
	}
	logf("Images after upgraded: PD: %s, TiKV: %s, TiDB: %s", tc.Spec.PD.Image, tc.Spec.TiKV.Image, tc.Spec.TiDB.Image)

	return true, nil
}

func memberUpgraded() (bool, error) {
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tidbcluster: [%s], error: %v", clusterName, err)
		return false, nil
	}

	pdSetName := controller.PDMemberName(clusterName)
	pdSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(pdSetName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get pd statefulset: [%s], error: %v", pdSetName, err)
		return false, nil
	}

	tikvSetName := controller.TiKVMemberName(clusterName)
	tikvSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(tikvSetName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tikvSet statefulset: [%s], error: %v", tikvSetName, err)
		return false, nil
	}

	tidbSetName := controller.TiDBMemberName(clusterName)
	tidbSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(tidbSetName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tidbSet statefulset: [%s], error: %v", tidbSetName, err)
		return false, nil
	}

	if !imageUpgraded(tc, v1alpha1.PDMemberType, pdSet) {
		return false, nil
	}
	if tc.Status.PD.Phase == v1alpha1.UpgradePhase {
		logf("pd is upgrading")
		Expect(tc.Status.TiKV.Phase).NotTo(Equal(v1alpha1.UpgradePhase))
		Expect(tc.Status.TiDB.Phase).NotTo(Equal(v1alpha1.UpgradePhase))
		Expect(imageUpgraded(tc, v1alpha1.PDMemberType, pdSet)).To(BeTrue())
		if !podsUpgraded(pdSet) {
			Expect(imageUpgraded(tc, v1alpha1.TiKVMemberType, tikvSet)).To(BeFalse())
			Expect(imageUpgraded(tc, v1alpha1.TiDBMemberType, tidbSet)).To(BeFalse())
		}
		return false, nil
	} else if tc.Status.TiKV.Phase == v1alpha1.UpgradePhase {
		logf("tikv is upgrading")
		Expect(tc.Status.TiDB.Phase).NotTo(Equal(v1alpha1.UpgradePhase))
		Expect(imageUpgraded(tc, v1alpha1.PDMemberType, pdSet)).To(BeTrue())
		Expect(podsUpgraded(pdSet)).To(BeTrue())
		Expect(imageUpgraded(tc, v1alpha1.TiKVMemberType, tikvSet)).To(BeTrue())
		if !podsUpgraded(tikvSet) {
			Expect(imageUpgraded(tc, v1alpha1.TiDBMemberType, tidbSet)).To(BeFalse())
		}
		return false, nil
	} else if tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
		logf("tidb is upgrading")
		Expect(imageUpgraded(tc, v1alpha1.PDMemberType, pdSet)).To(BeTrue())
		Expect(podsUpgraded(pdSet)).To(BeTrue())
		Expect(imageUpgraded(tc, v1alpha1.TiKVMemberType, tikvSet)).To(BeTrue())
		Expect(podsUpgraded(tikvSet)).To(BeTrue())
		Expect(imageUpgraded(tc, v1alpha1.TiDBMemberType, tidbSet)).To(BeTrue())
		return false, nil
	}
	if !imageUpgraded(tc, v1alpha1.PDMemberType, pdSet) {
		return false, nil
	}
	if !podsUpgraded(pdSet) {
		return false, nil
	}
	if !imageUpgraded(tc, v1alpha1.TiKVMemberType, tikvSet) {
		return false, nil
	}
	if !podsUpgraded(tikvSet) {
		return false, nil
	}
	if !imageUpgraded(tc, v1alpha1.TiDBMemberType, tidbSet) {
		return false, nil
	}
	return podsUpgraded(tidbSet), nil

}

func imageUpgraded(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, set *apps.StatefulSet) bool {
	for _, container := range set.Spec.Template.Spec.Containers {
		if container.Name == memberType.String() {
			if container.Image == getImage(tc, memberType) {
				return true
			}
		}
	}
	return false
}

func podsUpgraded(set *apps.StatefulSet) bool {
	return set.Generation <= *set.Status.ObservedGeneration && set.Status.CurrentRevision == set.Status.UpdateRevision
}

func getImage(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) string {
	switch memberType {
	case v1alpha1.PDMemberType:
		return tc.Spec.PD.Image
	case v1alpha1.TiKVMemberType:
		return tc.Spec.TiKV.Image
	case v1alpha1.TiDBMemberType:
		return tc.Spec.TiDB.Image
	default:
		return ""
	}
}

func getImageTag(image string) string {
	strs := strings.Split(image, ":")
	return strs[len(strs)-1]
}
