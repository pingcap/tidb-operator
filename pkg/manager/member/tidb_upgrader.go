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
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"sort"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	apps "k8s.io/api/apps/v1"
	"k8s.io/klog/v2"
)

type tidbUpgrader struct {
	deps *controller.Dependencies
}

// NewTiDBUpgrader returns a tidb Upgrader
func NewTiDBUpgrader(deps *controller.Dependencies) Upgrader {
	return &tidbUpgrader{
		deps: deps,
	}
}

func (u *tidbUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	// when scale replica to 0 , all nodes crash and tidb is in upgrade phase, this method will throw error about pod is upgrade.
	// so  directly return nil when scale replica to 0.
	if tc.Spec.TiDB.Replicas == int32(0) {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Status.PD.Phase == v1alpha1.UpgradePhase ||
		tc.Status.TiKV.Phase == v1alpha1.UpgradePhase ||
		tc.Status.TiFlash.Phase == v1alpha1.UpgradePhase ||
		tc.Status.Pump.Phase == v1alpha1.UpgradePhase ||
		tc.TiDBScaling() {
		klog.Infof("TidbCluster: [%s/%s]'s pd status is %s, "+
			"tikv status is %s, tiflash status is %s, pump status is %s, "+
			"tidb status is %s, can not upgrade tidb",
			ns, tcName,
			tc.Status.PD.Phase, tc.Status.TiKV.Phase, tc.Status.TiFlash.Phase,
			tc.Status.Pump.Phase, tc.Status.TiDB.Phase)
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
	if !templateEqual(newSet, oldSet) {
		return nil
	}

	if tc.Status.TiDB.StatefulSet.UpdateRevision == tc.Status.TiDB.StatefulSet.CurrentRevision {
		return nil
	}

	if tc.IsEnableIntelligentOperation() {
		//get need to upgrade pod list
		pods, err := GetPodsToUpgrade(u.deps, oldSet)
		if err != nil {
			return err
		}
		// sort candidates
		sortedCandidates := u.sortCandidates(tc, pods)
		klog.Infof("sortedCandidates:%v", sortedCandidates)
		for _, candidate := range sortedCandidates {
			podName := candidate.Name
			pod, err := u.deps.PodLister.Pods(ns).Get(podName)
			if err != nil {
				return fmt.Errorf("tidbUpgrader.Upgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
			}
			pod1Ordinal, err := util.GetOrdinalFromPodName(podName)
			i := pod1Ordinal

			revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
			if !exist {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
			}

			if revision == tc.Status.TiDB.StatefulSet.UpdateRevision {
				if member, exist := tc.Status.TiDB.Members[podName]; !exist || !member.Health {
					return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb upgraded pod: [%s] is not ready", ns, tcName, podName)
				}
				continue
			}
			u.deps.Recorder.Event(tc, corev1.EventTypeNormal, fmt.Sprintf("TiDBUpgrade"), fmt.Sprintf("%s:upgrade pod %s QPS:%f", SplitRevision(tc.Status.PD.StatefulSet.UpdateRevision), pod.Name, u.deps.MetricCache.GetTiDBQPSRate(fmt.Sprintf("%s-%s", tc.Namespace, tcName), podName)))

			return u.upgradeTiDBPod(tc, i, newSet)
		}
	} else {
		if oldSet.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType || oldSet.Spec.UpdateStrategy.RollingUpdate == nil {
			// Manually bypass tidb-operator to modify statefulset directly, such as modify tidb statefulset's RollingUpdate strategy to OnDelete strategy,
			// or set RollingUpdate to nil, skip tidb-operator's rolling update logic in order to speed up the upgrade in the test environment occasionally.
			// If we encounter this situation, we will let the native statefulset controller do the upgrade completely, which may be unsafe for upgrading tidb.
			// Therefore, in the production environment, we should try to avoid modifying the tidb statefulset update strategy directly.
			newSet.Spec.UpdateStrategy = oldSet.Spec.UpdateStrategy
			klog.Warningf("tidbcluster: [%s/%s] tidb statefulset %s UpdateStrategy has been modified manually", ns, tcName, oldSet.GetName())
			return nil
		}

		mngerutils.SetUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
		podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()
		for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
			i := podOrdinals[_i]
			podName := TidbPodName(tcName, i)
			pod, err := u.deps.PodLister.Pods(ns).Get(podName)
			if err != nil {
				return fmt.Errorf("tidbUpgrader.Upgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
			}
			revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
			if !exist {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
			}

			if revision == tc.Status.TiDB.StatefulSet.UpdateRevision {
				if member, exist := tc.Status.TiDB.Members[podName]; !exist || !member.Health {
					return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb upgraded pod: [%s] is not ready", ns, tcName, podName)
				}
				continue
			}
			return u.upgradeTiDBPod(tc, i, newSet)
		}
	}

	return nil
}

func (u *tidbUpgrader) sortCandidates(tc *v1alpha1.TidbCluster, allPods []*corev1.Pod) []*corev1.Pod {
	// Step 1. Sort the Pods to get the ones with the higher priority
	candidates := make([]*corev1.Pod, len(allPods))
	copy(candidates, allPods)
	tcName := fmt.Sprintf("%s-%s", tc.Namespace, tc.Name)

	sort.Slice(candidates, func(i, j int) bool {
		pod1 := allPods[i]
		pod2 := allPods[j]
		// compare client traffic
		pod1QPS := u.deps.MetricCache.GetTiDBQPSRate(fmt.Sprintf("%s-%s", tc.Namespace, tcName), pod1.Name)
		pod2QPS := u.deps.MetricCache.GetTiDBQPSRate(fmt.Sprintf("%s-%s", tc.Namespace, tcName), pod2.Name)
		klog.Infof("sort %s compare %s  %f:%f", pod1.Name, pod2.Name, pod1QPS, pod2QPS)
		return pod1QPS < pod2QPS
	})
	return candidates
}

func (u *tidbUpgrader) upgradeTiDBPod(tc *v1alpha1.TidbCluster, ordinal int32, newSet *apps.StatefulSet) error {
	if tc.IsEnableIntelligentOperation() {
		ns := tc.GetNamespace()
		tcName := tc.GetName()
		upgradePodName := TidbPodName(tcName, ordinal)
		pod, err := u.deps.PodLister.Pods(ns).Get(upgradePodName)
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("tikv upgrade[get pod]: failed to upgrade tikv %s/%s for tc %s/%s, error: %s", ns, upgradePodName, ns, tcName, err)
		}
		if pod != nil {
			if pod.DeletionTimestamp == nil {
				if err := u.deps.PodControl.DeletePod(tc, pod); err != nil {
					return err
				}
			}
		} else {
			klog.Infof("tikv upgrade: get pod %s/%s not found, skip", ns, upgradePodName)
		}
	} else {
		mngerutils.SetUpgradePartition(newSet, ordinal)
	}

	return nil
}

type fakeTiDBUpgrader struct{}

// NewFakeTiDBUpgrader returns a fake tidb upgrader
func NewFakeTiDBUpgrader() Upgrader {
	return &fakeTiDBUpgrader{}
}

func (u *fakeTiDBUpgrader) Upgrade(tc *v1alpha1.TidbCluster, _ *apps.StatefulSet, _ *apps.StatefulSet) error {
	tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
	return nil
}
