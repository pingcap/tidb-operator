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

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
)

type pdUpgrader struct {
	pdControl  pdapi.PDControlInterface
	podControl controller.PodControlInterface
	podLister  corelisters.PodLister
}

// NewPDUpgrader returns a pdUpgrader
func NewPDUpgrader(pdControl pdapi.PDControlInterface,
	podControl controller.PodControlInterface,
	podLister corelisters.PodLister) Upgrader {
	return &pdUpgrader{
		pdControl:  pdControl,
		podControl: podControl,
		podLister:  podLister,
	}
}

func (pu *pdUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	return pu.gracefulUpgrade(tc, oldSet, newSet)
}

func (pu *pdUpgrader) gracefulUpgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	if !tc.Status.PD.Synced {
		return fmt.Errorf("tidbcluster: [%s/%s]'s pd status sync failed,can not to be upgraded", ns, tcName)
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

	setUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()
	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		podName := PdPodName(tcName, i)
		pod, err := pu.podLister.Pods(ns).Get(podName)
		if err != nil {
			return err
		}

		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == tc.Status.PD.StatefulSet.UpdateRevision {
			if member, exist := tc.Status.PD.Members[podName]; !exist || !member.Health {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd upgraded pod: [%s] is not ready", ns, tcName, podName)
			}
			continue
		}

		if controller.PodWebhookEnabled {
			setUpgradePartition(newSet, i)
			return nil
		}

		return pu.upgradePDPod(tc, i, newSet)
	}

	return nil
}

func (pu *pdUpgrader) upgradePDPod(tc *v1alpha1.TidbCluster, ordinal int32, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	upgradePodName := PdPodName(tcName, ordinal)
	if tc.Status.PD.Leader.Name == upgradePodName && tc.PDStsActualReplicas() > 1 {
		lastOrdinal := tc.PDStsActualReplicas() - 1
		var targetName string
		if ordinal == lastOrdinal {
			targetName = PdPodName(tcName, 0)
		} else {
			targetName = PdPodName(tcName, lastOrdinal)
		}
		err := pu.transferPDLeaderTo(tc, targetName)
		if err != nil {
			klog.Errorf("pd upgrader: failed to transfer pd leader to: %s, %v", targetName, err)
			return err
		}
		klog.Infof("pd upgrader: transfer pd leader to: %s successfully", targetName)
		return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd member: [%s] is transferring leader to pd member: [%s]", ns, tcName, upgradePodName, targetName)
	}

	setUpgradePartition(newSet, ordinal)
	return nil
}

func (pu *pdUpgrader) transferPDLeaderTo(tc *v1alpha1.TidbCluster, targetName string) error {
	return controller.GetPDClient(pu.pdControl, tc).TransferPDLeader(targetName)
}

type fakePDUpgrader struct{}

// NewFakePDUpgrader returns a fakePDUpgrader
func NewFakePDUpgrader() Upgrader {
	return &fakePDUpgrader{}
}

func (fpu *fakePDUpgrader) Upgrade(tc *v1alpha1.TidbCluster, _ *apps.StatefulSet, _ *apps.StatefulSet) error {
	if !tc.Status.PD.Synced {
		return fmt.Errorf("tidbcluster: pd status sync failed,can not to be upgraded")
	}
	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	return nil
}
