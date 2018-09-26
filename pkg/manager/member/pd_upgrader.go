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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type pdUpgrader struct {
	pdControl  controller.PDControlInterface
	podControl controller.PodControlInterface
	podLister  corelisters.PodLister
}

// NewPDUpgrader returns a pdUpgrader
func NewPDUpgrader(pdControl controller.PDControlInterface,
	podControl controller.PodControlInterface,
	podLister corelisters.PodLister) Upgrader {
	return &pdUpgrader{
		pdControl:  pdControl,
		podControl: podControl,
		podLister:  podLister,
	}
}

func (pu *pdUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	force, err := pu.needForceUpgrade(tc)
	if err != nil {
		return err
	}
	if force {
		return pu.forceUpgrade(tc, oldSet, newSet)
	}

	return pu.gracefulUpgrade(tc, oldSet, newSet)
}

func (pu *pdUpgrader) forceUpgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	setUpgradePartition(newSet, 0)
	return nil
}

func (pu *pdUpgrader) gracefulUpgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	if !tc.Status.PD.SyncSuccess {
		return fmt.Errorf("tidbcluster: [%s/%s]'s pd status sync failed,can not to be upgraded", ns, tcName)
	}

	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	setUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)

	if tc.Status.PD.StatefulSet.CurrentReplicas == 0 {
		return nil
	}

	ordinal, err := pu.getUpgradeOrdinal(tc)
	if err != nil {
		return err
	}
	upgradePodName := pdPodName(tc, ordinal)
	if tc.Status.PD.Leader.Name == upgradePodName {
		var targetName string
		if ordinal == *newSet.Spec.Replicas-1 {
			targetName = pdPodName(tc, 0)
		} else {
			targetName = pdPodName(tc, *newSet.Spec.Replicas-1)
		}
		err := pu.transferPDLeaderTo(tc, targetName)
		if err != nil {
			return err
		}
	} else {
		setUpgradePartition(newSet, ordinal)
	}

	return nil
}

func (pu *pdUpgrader) getUpgradeOrdinal(tc *v1alpha1.TidbCluster) (int32, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Status.PD.StatefulSet.UpdatedReplicas+tc.Status.PD.StatefulSet.CurrentReplicas != tc.Status.PD.StatefulSet.Replicas {
		return -1, controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd is not ", ns, tcName)
	}

	for i := tc.Status.PD.StatefulSet.Replicas; i > tc.Status.PD.StatefulSet.CurrentReplicas; i-- {
		if member, exist := tc.Status.PD.Members[pdPodName(tc, i-1)]; !exist || !member.Health {
			return -1, controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded pods are not all ready", ns, tcName)
		}
	}

	return tc.Status.PD.StatefulSet.CurrentReplicas - 1, nil
}

func (pu *pdUpgrader) needForceUpgrade(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	selector, err := label.New().Cluster(tcName).PD().Selector()
	if err != nil {
		return false, err
	}
	pdPods, err := pu.podLister.Pods(ns).List(selector)
	if err != nil {
		return false, err
	}

	imagePullFailedCount := 0
	for _, pod := range pdPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, fmt.Errorf("tidbcluster: [%s/%s]'s pod:[%s] doesn't have label: %s", ns, tcName, pod.GetName(), apps.ControllerRevisionHashLabelKey)
		}
		if revisionHash == tc.Status.PD.StatefulSet.CurrentRevision {
			if imagePullFailed(pod) {
				imagePullFailedCount++
			}
		}
	}

	if imagePullFailedCount >= int(tc.Status.PD.StatefulSet.Replicas)/2+1 {
		return true, nil
	}
	return false, nil
}

func (pu *pdUpgrader) transferPDLeaderTo(tc *v1alpha1.TidbCluster, targetName string) error {
	return pu.pdControl.GetPDClient(tc).TransferPDLeader(targetName)
}

type fakePDUpgrader struct{}

// NewFakePDUpgrader returns a fakePDUpgrader
func NewFakePDUpgrader() Upgrader {
	return &fakePDUpgrader{}
}

func (fpu *fakePDUpgrader) Upgrade(tc *v1alpha1.TidbCluster, _ *apps.StatefulSet, _ *apps.StatefulSet) error {
	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	return nil
}
