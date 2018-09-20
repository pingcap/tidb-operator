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
	"sort"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	// ImagePullBackOff is the pod state of image pull failed
	ImagePullBackOff = "ImagePullBackOff"
	// ErrImagePull is the pod state of image pull failed
	ErrImagePull = "ErrImagePull"
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
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if !tc.Status.PD.SyncSuccess {
		force, err := pu.needForce(tc)
		if err != nil {
			return err
		}
		if force {
			tc.Status.PD.Phase = v1alpha1.UpgradePhase
			setUpgradePartition(newSet, 0)
			return nil
		}

		return fmt.Errorf("tidbcluster: [%s/%s]'s pd status sync failed,can not to be upgraded", ns, tcName)
	}

	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	setUpgradePartition(newSet, int(*oldSet.Spec.UpdateStrategy.RollingUpdate.Partition))

	upgradePod, err := pu.findUpgradePod(tc)
	if upgradePod == nil || err != nil {
		return err
	}

	ordinal, err := getOrdinal(upgradePod)
	if err != nil {
		return err
	}
	if tc.Status.PD.Leader.Name == upgradePod.GetName() {
		var targetName string
		maxName := fmt.Sprintf("%s-%d", controller.PDMemberName(tcName), int(*newSet.Spec.Replicas)-1)
		minName := fmt.Sprintf("%s-%d", controller.PDMemberName(tcName), 0)
		if ordinal == int(*newSet.Spec.Replicas)-1 {
			targetName = minName
		} else {
			targetName = maxName
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

func (pu *pdUpgrader) findUpgradePod(tc *v1alpha1.TidbCluster) (*corev1.Pod, error) {
	selector, err := label.New().Cluster(tc.GetName()).PD().Selector()
	if err != nil {
		return nil, err
	}
	pdPods, err := pu.podLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return nil, err
	}
	sort.Sort(descendingOrdinal(pdPods))
	for _, pod := range pdPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return nil, fmt.Errorf("tidbcluster: [%s/%s]'s pod[%s] have not label: %s", tc.GetNamespace(), tc.GetName(), pod.GetName(), apps.ControllerRevisionHashLabelKey)
		}
		if revisionHash == tc.Status.PD.StatefulSet.UpdateRevision {
			if !tc.Status.PD.Members[pod.GetName()].Health {
				return nil, nil
			}
			continue
		} else {
			return pod, nil
		}
	}
	return nil, nil
}

func (pu *pdUpgrader) needForce(tc *v1alpha1.TidbCluster) (bool, error) {
	selector, err := label.New().Cluster(tc.GetName()).PD().Selector()
	if err != nil {
		return false, err
	}
	pdPods, err := pu.podLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, err
	}

	for _, pod := range pdPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, fmt.Errorf("tidbcluster: [%s/%s]'s pod:[%s] have not label: %s", tc.GetNamespace(), tc.GetName(), pod.GetName(), apps.ControllerRevisionHashLabelKey)
		}
		if revisionHash == tc.Status.PD.StatefulSet.CurrentRevision {
			return imagePullFailed(pod), nil
		}
	}
	return false, nil
}

func (pu *pdUpgrader) transferPDLeaderTo(tc *v1alpha1.TidbCluster, targetName string) error {
	return pu.pdControl.GetPDClient(tc).TransferPDLeader(targetName)
}

/*
func (pu *pdUpgrader) setUpgradePartition(newSet *apps.StatefulSet, upgradeOrdinal int) {
	ordinal := int32(upgradeOrdinal)
	newSet.Spec.UpdateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{Partition: &ordinal}
}*/

type fakePDUpgrader struct{}

// NewFakePDUpgrader returns a fakePDUpgrader
func NewFakePDUpgrader() Upgrader {
	return &fakePDUpgrader{}
}

func (fpu *fakePDUpgrader) Upgrade(tc *v1alpha1.TidbCluster, _ *apps.StatefulSet, _ *apps.StatefulSet) error {
	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	return nil
}
