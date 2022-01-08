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

	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
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
	if tc.Spec.IsEnableIntelligentOperation != nil && *tc.Spec.IsEnableIntelligentOperation {
		// get need to upgrade pod list
		pods, err := u.podsToUpgrade(tc, oldSet)
		if err != nil {
			return err
		}
		// sort candidates by pd leader or no-leader
		sortedCandidates := u.sortCandidates(tc, pods)
		for _, candidate := range sortedCandidates {
			podName := candidate.Name
			pod1Ordinal, err := util.GetOrdinalFromPodName(podName)
			if err != nil {
				continue
			}
			pod, err := u.deps.PodLister.Pods(ns).Get(podName)
			if err != nil {
				return fmt.Errorf("gracefulUpgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
			}

			revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
			if !exist {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
			}

			if revision == tc.Status.PD.StatefulSet.UpdateRevision {
				if member, exist := tc.Status.PD.Members[PdName(tc.Name, pod1Ordinal, tc.Namespace, tc.Spec.ClusterDomain)]; !exist || !member.Health {
					return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd upgraded pod: [%s] is not ready", ns, tcName, podName)
				}
				continue
			}
			u.deps.Recorder.Event(tc, corev1.EventTypeNormal, fmt.Sprintf("PDUpgrade"), fmt.Sprintf("%s:upgrade pod %s", SplitRevision(tc.Status.PD.StatefulSet.UpdateRevision), pod.Name))
			return u.upgradePDPod(tc, pod1Ordinal, newSet)
		}
	} else {
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
				if member, exist := tc.Status.PD.Members[PdName(tc.Name, i, tc.Namespace, tc.Spec.ClusterDomain)]; !exist || !member.Health {
					return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd upgraded pod: [%s] is not ready", ns, tcName, podName)
				}
				continue
			}

			return u.upgradePDPod(tc, i, newSet)
		}
	}
	return nil
}

func (u *pdUpgrader) podsToUpgrade(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) ([]*corev1.Pod, error) {
	sts, err := u.deps.StatefulSetLister.StatefulSets(set.Namespace).Get(set.Name)
	if err != nil {
		return nil, err
	}
	var replica int32
	if sts.Spec.Replicas != nil {
		replica = *sts.Spec.Replicas
	} else {
		replica = 0
	}
	// get upgrade pods
	var toUpgrade []*corev1.Pod
	for idx := replica - 1; idx >= 0; idx-- {
		// Do we need to upgrade that pod?
		podName := fmt.Sprintf("%s-%d", sts.Name, idx)
		pod, err := u.deps.PodLister.Pods(sts.Namespace).Get(podName)
		if err != nil && !errors.IsNotFound(err) {
			break
		}
		if err != nil && errors.IsNotFound(err) {
			// Pod does not exist, continue the loop as the absence will be accounted by the deletion driver
			continue
		}

		if sts.Status.UpdateRevision != pod.Labels["controller-revision-hash"] {
			toUpgrade = append(toUpgrade, pod)
		}
	}

	return toUpgrade, nil
}

func (u *pdUpgrader) sortCandidates(tc *v1alpha1.TidbCluster, allPods []*corev1.Pod) []*corev1.Pod {
	// Step 1. Sort the Pods to get the ones with the higher priority
	candidates := make([]*corev1.Pod, len(allPods))
	copy(candidates, allPods)
	sort.Slice(candidates, func(i, j int) bool {
		pod1 := allPods[i]
		pod2 := allPods[j]
		// check if either is a master node. masters come after all other roles
		tcName := tc.Name
		pod1Ordinal, err := util.GetOrdinalFromPodName(pod1.Name)
		if err != nil {
			return false
		}
		pod1PdName := PdName(tcName, pod1Ordinal, tc.Namespace, tc.Spec.ClusterDomain)
		pod1OrdinalPodName := PdPodName(tcName, pod1Ordinal)

		pod2Ordinal, err := util.GetOrdinalFromPodName(pod2.Name)
		if err != nil {
			return false
		}
		pod2PdName := PdName(tcName, pod2Ordinal, tc.Namespace, tc.Spec.ClusterDomain)
		pod2OrdinalPodName := PdPodName(tcName, pod2Ordinal)
		klog.Infof("sort:%s,%s", pod1PdName, pod2PdName)

		if tc.Status.PD.Leader.Name == pod1PdName || tc.Status.PD.Leader.Name == pod1OrdinalPodName {
			return false
		}

		if tc.Status.PD.Leader.Name == pod2PdName || tc.Status.PD.Leader.Name == pod2OrdinalPodName {
			return true
		}
		return pod1PdName > pod2PdName
	})
	return candidates
}

func (u *pdUpgrader) upgradePDPod(tc *v1alpha1.TidbCluster, ordinal int32, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	upgradePdName := PdName(tcName, ordinal, tc.Namespace, tc.Spec.ClusterDomain)
	upgradePodName := PdPodName(tcName, ordinal)
	klog.Infof("cwtest pd transfer 1: ")
	if tc.Status.PD.Leader.Name == upgradePdName || tc.Status.PD.Leader.Name == upgradePodName {
		klog.Infof("cwtest pd transfer 2: ")
		var targetName string
		if tc.PDStsActualReplicas() > 1 {
			targetOrdinal := helper.GetMaxPodOrdinal(*newSet.Spec.Replicas, newSet)
			klog.Infof("cwtest pd upgrader: targetOrdinal:%d,ordinal:%d", targetOrdinal, ordinal)
			if ordinal == targetOrdinal {
				targetOrdinal = helper.GetMinPodOrdinal(*newSet.Spec.Replicas, newSet)
			}
			targetName = PdName(tcName, targetOrdinal, tc.Namespace, tc.Spec.ClusterDomain)
			if _, exist := tc.Status.PD.Members[targetName]; !exist {
				targetName = PdPodName(tcName, targetOrdinal)
			}
		} else {
			for _, member := range tc.Status.PD.PeerMembers {
				if member.Name != upgradePdName && member.Health {
					targetName = member.Name
					break
				}
			}
		}
		if len(targetName) > 0 {
			u.deps.Recorder.Event(tc, corev1.EventTypeNormal, fmt.Sprintf("PDUpgrade"), fmt.Sprintf("%s:leader transfer to %s", SplitRevision(tc.Status.PD.StatefulSet.UpdateRevision), targetName))
			err := u.transferPDLeaderTo(tc, targetName)
			if err != nil {
				klog.Errorf("pd upgrader: failed to transfer pd leader to: %s, %v", targetName, err)
				return err
			}

			klog.Infof("pd upgrader: transfer pd leader to: %s successfully", targetName)
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd member: [%s] is transferring leader to pd member: [%s]", ns, tcName, upgradePdName, targetName)
		}
	}
	if tc.Spec.IsEnableIntelligentOperation != nil && *tc.Spec.IsEnableIntelligentOperation {
		pod, err := u.deps.PodLister.Pods(ns).Get(upgradePodName)
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("pd upgrade[get pod]: failed to upgrade pod %s/%s for tc %s/%s, error: %s", ns, upgradePodName, ns, tcName, err)
		}
		if pod != nil {
			if pod.DeletionTimestamp == nil {
				if err := u.deps.PodControl.DeletePod(tc, pod); err != nil {
					return err
				}
			}
		} else {
			klog.Infof("pd upgrade: get pod %s/%s not found, skip", ns, upgradePodName)
		}
	} else {
		mngerutils.SetUpgradePartition(newSet, ordinal)
	}
	return nil
}

func (u *pdUpgrader) transferPDLeaderTo(tc *v1alpha1.TidbCluster, targetName string) error {
	return controller.GetPDClient(u.deps.PDControl, tc).TransferPDLeader(targetName)
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
