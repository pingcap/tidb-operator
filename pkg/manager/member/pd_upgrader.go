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
// limitations under the License.

package member

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	apps "k8s.io/api/apps/v1"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

type pdUpgrader struct {
	deps *controller.Dependencies
}

// NewPDUpgrader returns a pdUpgrader
func NewPDUpgrader(deps *controller.Dependencies) Upgrader {
	return &pdUpgrader{
		deps: deps,
	}
}

func (u *pdUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	return u.gracefulUpgrade(tc, oldSet, newSet)
}

func (u *pdUpgrader) gracefulUpgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	if !tc.Status.PD.Synced {
		return fmt.Errorf("tidbcluster: [%s/%s]'s pd status sync failed, can not to be upgraded", ns, tcName)
	}
	if tc.PDScaling() {
		klog.Infof("TidbCluster: [%s/%s]'s pd status is %v, can not upgrade pd",
			ns, tcName, tc.Status.PD.Phase)
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	if !templateEqual(newSet, oldSet) {
		return nil
	}

	if tc.Status.PD.StatefulSet.UpdateRevision == tc.Status.PD.StatefulSet.CurrentRevision {
		return nil
	}

	if oldSet.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType || oldSet.Spec.UpdateStrategy.RollingUpdate == nil {
		// Manually bypass tidb-operator to modify statefulset directly, such as modify pd statefulset's RollingUpdate straregy to OnDelete strategy,
		// or set RollingUpdate to nil, skip tidb-operator's rolling update logic in order to speed up the upgrade in the test environment occasionally.
		// If we encounter this situation, we will let the native statefulset controller do the upgrade completely, which may be unsafe for upgrading pd.
		// Therefore, in the production environment, we should try to avoid modifying the pd statefulset update strategy directly.
		newSet.Spec.UpdateStrategy = oldSet.Spec.UpdateStrategy
		klog.Warningf("tidbcluster: [%s/%s] pd statefulset %s UpdateStrategy has been modified manually", ns, tcName, oldSet.GetName())
		return nil
	}

	mngerutils.SetUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()
	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		podName := PdPodName(tcName, i)
		pod, err := u.deps.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return fmt.Errorf("gracefulUpgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
		}

		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == tc.Status.PD.StatefulSet.UpdateRevision {
			if !podutil.IsPodReady(pod) {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded pd pod: [%s] is not ready", ns, tcName, podName)
			}
			if member, exist := tc.Status.PD.Members[PdName(tc.Name, i, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)]; !exist || !member.Health {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd upgraded pod: [%s] is not health", ns, tcName, podName)
			}
			continue
		}

		return u.upgradePDPod(tc, i, newSet)
	}

	return nil
}

func (u *pdUpgrader) upgradePDPod(tc *v1alpha1.TidbCluster, ordinal int32, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	upgradePdName := PdName(tcName, ordinal, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)
	upgradePodName := PdPodName(tcName, ordinal)

	// If current pd is leader, transfer leader to other pd
	if tc.Status.PD.Leader.Name == upgradePdName || tc.Status.PD.Leader.Name == upgradePodName {
		targetName := ""

		if tc.PDStsActualReplicas() > 1 {
			targetName = choosePDToTransferFromMembers(tc, newSet, ordinal)
		}

		if targetName == "" {
			targetName = choosePDToTransferFromPeerMembers(tc, upgradePdName)
		}

		if targetName != "" {
			err := u.transferPDLeaderTo(tc, targetName)
			if err != nil {
				klog.Errorf("pd upgrader: failed to transfer pd leader to: %s, %v", targetName, err)
				return err
			}
			klog.Infof("pd upgrader: transfer pd leader to: %s successfully", targetName)
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd member: [%s] is transferring leader to pd member: [%s]", ns, tcName, upgradePdName, targetName)
		} else {
			klog.Warningf("pd upgrader: skip to transfer pd leader, because can not find a suitable pd")
		}
	}

	mngerutils.SetUpgradePartition(newSet, ordinal)
	return nil
}

func (u *pdUpgrader) transferPDLeaderTo(tc *v1alpha1.TidbCluster, targetName string) error {
	return controller.GetPDClient(u.deps.PDControl, tc).TransferPDLeader(targetName)
}

// choosePDToTransferFromMembers choose a pd to transfer leader from members
//
// Assume that current leader ordinal is x, and range is [0, n]
//  1. Find the max suitable ordinal in (x, n], because they have been upgraded
//  2. If no suitable ordinal, find the min suitable ordinal in [0, x) to reduce the count of transfer
func choosePDToTransferFromMembers(tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, ordinal int32) string {
	tcName := tc.GetName()
	ordinals := helper.GetPodOrdinals(*newSet.Spec.Replicas, newSet)

	genPDName := func(targetOrdinal int32) string {
		pdName := PdName(tcName, targetOrdinal, tc.Namespace, tc.Spec.ClusterDomain, tc.Spec.AcrossK8s)
		if _, exist := tc.Status.PD.Members[pdName]; !exist {
			pdName = PdPodName(tcName, targetOrdinal)
		}
		return pdName
	}
	pred := func(pdName string) bool {
		return tc.Status.PD.Members[pdName].Health
	}

	// set ordinal to max ordinal if ordinal isn't exist
	if !ordinals.Has(ordinal) {
		ordinal = helper.GetMaxPodOrdinal(*newSet.Spec.Replicas, newSet)
	}

	targetName := ""
	list := ordinals.List()

	// find the max ordinal which is larger than ordinal
	for i := len(list) - 1; i >= 0 && list[i] > ordinal; i-- {
		curName := genPDName(list[i])
		if pred(curName) {
			targetName = curName
			break
		}
	}

	if targetName == "" {
		// find the min ordinal which is less than ordinal
		for i := 0; i < len(list) && list[i] < ordinal; i++ {
			curName := genPDName(list[i])
			if pred(curName) {
				targetName = curName
				break
			}
		}
	}

	return targetName
}

// choosePDToTransferFromPeerMembers choose a pd to transfer leader from peer members
func choosePDToTransferFromPeerMembers(tc *v1alpha1.TidbCluster, upgradePdName string) string {
	for _, member := range tc.Status.PD.PeerMembers {
		if member.Name != upgradePdName && member.Health {
			return member.Name
		}
	}

	return ""
}

type fakePDUpgrader struct{}

// NewFakePDUpgrader returns a fakePDUpgrader
func NewFakePDUpgrader() Upgrader {
	return &fakePDUpgrader{}
}

func (u *fakePDUpgrader) Upgrade(tc *v1alpha1.TidbCluster, _ *apps.StatefulSet, _ *apps.StatefulSet) error {
	if !tc.Status.PD.Synced {
		return fmt.Errorf("tidbcluster: pd status sync failed, can not to be upgraded")
	}
	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	return nil
}
