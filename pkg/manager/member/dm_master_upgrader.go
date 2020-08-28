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

package member

import (
	"fmt"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/dmapi"

	apps "k8s.io/api/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
)

type masterUpgrader struct {
	masterControl dmapi.MasterControlInterface
	podLister     corelisters.PodLister
}

// NewMasterUpgrader returns a masterUpgrader
func NewMasterUpgrader(masterControl dmapi.MasterControlInterface,
	podLister corelisters.PodLister) DMUpgrader {
	return &masterUpgrader{
		masterControl: masterControl,
		podLister:     podLister,
	}
}

func (mu *masterUpgrader) Upgrade(dc *v1alpha1.DMCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	return mu.gracefulUpgrade(dc, oldSet, newSet)
}

func (mu *masterUpgrader) gracefulUpgrade(dc *v1alpha1.DMCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := dc.GetNamespace()
	dcName := dc.GetName()
	if !dc.Status.Master.Synced {
		return fmt.Errorf("dmcluster: [%s/%s]'s dm-master status sync failed, can not to be upgraded", ns, dcName)
	}
	if dc.MasterScaling() {
		klog.Infof("DMCluster: [%s/%s]'s dm-master is scaling, can not upgrade dm-master", ns, dcName)
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	dc.Status.Master.Phase = v1alpha1.UpgradePhase
	if !templateEqual(newSet, oldSet) {
		return nil
	}

	if dc.Status.Master.StatefulSet.UpdateRevision == dc.Status.Master.StatefulSet.CurrentRevision {
		return nil
	}

	if oldSet.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType || oldSet.Spec.UpdateStrategy.RollingUpdate == nil {
		// Manually bypass tidb-operator to modify statefulset directly, such as modify dm-master statefulset's RollingUpdate straregy to OnDelete strategy,
		// or set RollingUpdate to nil, skip tidb-operator's rolling update logic in order to speed up the upgrade in the test environment occasionally.
		// If we encounter this situation, we will let the native statefulset controller do the upgrade completely, which may be unsafe for upgrading dm-master.
		// Therefore, in the production environment, we should try to avoid modifying the dm-master statefulset update strategy directly.
		newSet.Spec.UpdateStrategy = oldSet.Spec.UpdateStrategy
		klog.Warningf("dmcluster: [%s/%s] dm-master statefulset %s UpdateStrategy has been modified manually", ns, dcName, oldSet.GetName())
		return nil
	}

	setUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()
	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		podName := DMMasterPodName(dcName, i)
		pod, err := mu.podLister.Pods(ns).Get(podName)
		if err != nil {
			return fmt.Errorf("gracefulUpgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, dcName, err)
		}

		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("dmcluster: [%s/%s]'s dm-master pod: [%s] has no label: %s", ns, dcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == dc.Status.Master.StatefulSet.UpdateRevision {
			if member, exist := dc.Status.Master.Members[podName]; !exist || !member.Health {
				return controller.RequeueErrorf("dmcluster: [%s/%s]'s dm-master upgraded pod: [%s] is not ready", ns, dcName, podName)
			}
			continue
		}

		//if controller.PodWebhookEnabled {
		//	setUpgradePartition(newSet, i)
		//	return nil
		//}

		return mu.upgradeMasterPod(dc, i, newSet)
	}

	return nil
}

func (mu *masterUpgrader) upgradeMasterPod(dc *v1alpha1.DMCluster, ordinal int32, newSet *apps.StatefulSet) error {
	ns := dc.GetNamespace()
	dcName := dc.GetName()
	upgradePodName := DMMasterPodName(dcName, ordinal)
	if dc.Status.Master.Leader.Name == upgradePodName && dc.MasterStsActualReplicas() > 1 {
		err := mu.evictMasterLeader(dc, upgradePodName)
		if err != nil {
			klog.Errorf("dm-master upgrader: failed to evict dm-master %s's leader: %v", upgradePodName, err)
			return err
		}
		klog.Infof("dm-master upgrader: evict dm-master %s's leader successfully", upgradePodName)
		return controller.RequeueErrorf("dmcluster: [%s/%s]'s dm-master member: evicting [%s]'s leader", ns, dcName, upgradePodName)
	}

	setUpgradePartition(newSet, ordinal)
	return nil
}

func (mu *masterUpgrader) evictMasterLeader(dc *v1alpha1.DMCluster, podName string) error {
	return controller.GetMasterPeerClient(mu.masterControl, dc, podName).EvictLeader()
}

type fakeMasterUpgrader struct{}

// NewFakeMasterUpgrader returns a fakeMasterUpgrader
func NewFakeMasterUpgrader() DMUpgrader {
	return &fakeMasterUpgrader{}
}

func (fmu *fakeMasterUpgrader) Upgrade(dc *v1alpha1.DMCluster, _ *apps.StatefulSet, _ *apps.StatefulSet) error {
	if !dc.Status.Master.Synced {
		return fmt.Errorf("dmcluster: dm-master status sync failed,can not to be upgraded")
	}
	dc.Status.Master.Phase = v1alpha1.UpgradePhase
	return nil
}
