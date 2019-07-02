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

	"github.com/pingcap/tidb-operator/pkg/apis/pdapi"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
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
	if !tc.Status.PD.Synced {
		return fmt.Errorf("tidbcluster: [%s/%s]'s pd status sync failed,can not to be upgraded", ns, tcName)
	}

	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	if !templateEqual(newSet.Spec.Template, oldSet.Spec.Template) {
		return nil
	}

	setUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	for i := tc.Status.PD.StatefulSet.Replicas - 1; i >= 0; i-- {
		podName := pdPodName(tcName, i)
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

		return pu.upgradePDPod(tc, i, newSet)
	}

	return nil
}

func (pu *pdUpgrader) needForceUpgrade(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	instanceName := tc.GetLabels()[label.InstanceLabelKey]
	selector, err := label.New().Instance(instanceName).PD().Selector()
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

	return imagePullFailedCount >= int(tc.Status.PD.StatefulSet.Replicas)/2+1, nil
}

func (pu *pdUpgrader) upgradePDPod(tc *v1alpha1.TidbCluster, ordinal int32, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	upgradePodName := pdPodName(tcName, ordinal)
	if tc.Status.PD.Leader.Name == upgradePodName && tc.Status.PD.StatefulSet.Replicas > 1 {
		lastOrdinal := tc.Status.PD.StatefulSet.Replicas - 1
		var targetName string
		if ordinal == lastOrdinal {
			targetName = pdPodName(tcName, 0)
		} else {
			targetName = pdPodName(tcName, lastOrdinal)
		}
		err := pu.transferPDLeaderTo(tc, targetName)
		if err != nil {
			return err
		}
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
	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	return nil
}
