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
	"context"
	"fmt"
	"strconv"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/third_party/k8s"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const (
	// TODO: change to use minReadySeconds in sts spec
	// See https://kubernetes.io/blog/2021/08/27/minreadyseconds-statefulsets/
	annoKeyTiDBMinReadySeconds = "tidb.pingcap.com/tidb-min-ready-seconds"
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

	if tc.Status.PD.Phase == v1alpha1.UpgradePhase || tc.Status.PD.Phase == v1alpha1.ScalePhase ||
		tc.Status.TiKV.Phase == v1alpha1.UpgradePhase || tc.Status.TiKV.Phase == v1alpha1.ScalePhase ||
		tc.Status.TiFlash.Phase == v1alpha1.UpgradePhase || tc.Status.TiFlash.Phase == v1alpha1.ScalePhase ||
		tc.Status.Pump.Phase == v1alpha1.UpgradePhase || tc.Status.Pump.Phase == v1alpha1.ScalePhase ||
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
	if err := u.ensureSmoothUpgradeStarted(tc, oldSet, newSet); err != nil {
		return err
	}
	if !templateEqual(newSet, oldSet) {
		return nil
	}

	if oldSet.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType || oldSet.Spec.UpdateStrategy.RollingUpdate == nil {
		// Manually bypass tidb-operator to modify statefulset directly, such as modify tidb statefulset's RollingUpdate strategy to OnDelete strategy,
		// or set RollingUpdate to nil, skip tidb-operator's rolling update logic in order to speed up the upgrade in the test environment occasionally.
		// If we encounter this situation, we will let the native statefulset controller do the upgrade completely, which may be unsafe for upgrading tidb.
		// Therefore, in the production environment, we should try to avoid modifying the tidb statefulset update strategy directly.
		newSet.Spec.UpdateStrategy = oldSet.Spec.UpdateStrategy
		klog.Warningf("tidbcluster: [%s/%s] tidb statefulset %s UpdateStrategy has been modified manually", ns, tcName, oldSet.GetName())
		return nil
	}

	minReadySeconds := 0
	s, ok := tc.Annotations[annoKeyTiDBMinReadySeconds]
	if ok {
		i, err := strconv.Atoi(s)
		if err != nil {
			klog.Warningf("tidbcluster: [%s/%s] annotation %s should be an integer: %v", ns, tcName, annoKeyTiDBMinReadySeconds, err)
		} else {
			minReadySeconds = i
		}
	}

	mngerutils.SetUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()

	// budget is how many pods may be down at the same time during the rolling
	// update. 1 (the default) is the classic strictly-serial behavior.
	// Whatever spec.tidb.upgradePolicy.maxUnavailable says, at least one pod
	// is always kept out of the restart window.
	maxUnavailable := tc.Spec.TiDB.GetUpgradeMaxUnavailable(len(podOrdinals))
	if len(podOrdinals) > 1 && maxUnavailable > len(podOrdinals)-1 {
		maxUnavailable = len(podOrdinals) - 1
	}

	budget := maxUnavailable
	// blocked keeps the requeue message for the first in-flight pod (largest
	// ordinal), which with maxUnavailable=1 is exactly the message the serial
	// upgrade returned.
	blocked := ""
	// outdated pods that still need the update, largest ordinal first
	var pending []int32

	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		podName := tidbPodName(tcName, i)
		pod, err := u.deps.PodLister.Pods(ns).Get(podName)
		if err != nil {
			if errors.IsNotFound(err) {
				// Deleted for the upgrade and not recreated yet. Its ordinal
				// is at or above the partition, so it comes back at the
				// update revision: count it as in flight.
				budget--
				if blocked == "" {
					blocked = fmt.Sprintf("tidbcluster: [%s/%s]'s upgraded tidb pod: [%s] is being recreated", ns, tcName, podName)
				}
				continue
			}
			return fmt.Errorf("tidbUpgrader.Upgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
		}
		if pod.DeletionTimestamp != nil {
			budget--
			if blocked == "" {
				blocked = fmt.Sprintf("tidbcluster: [%s/%s]'s tidb pod: [%s] is terminating", ns, tcName, podName)
			}
			continue
		}
		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == tc.Status.TiDB.StatefulSet.UpdateRevision {
			if !k8s.IsPodAvailable(pod, int32(minReadySeconds), metav1.Now()) {
				budget--
				if blocked == "" {
					readyCond := k8s.GetPodReadyCondition(pod.Status)
					if readyCond == nil || readyCond.Status != corev1.ConditionTrue {
						blocked = fmt.Sprintf("tidbcluster: [%s/%s]'s upgraded tidb pod: [%s] is not ready", ns, tcName, podName)
					} else {
						blocked = fmt.Sprintf("tidbcluster: [%s/%s]'s upgraded tidb pod: [%s] is not available, last transition time is %v", ns, tcName, podName, readyCond.LastTransitionTime)
					}
				}
			} else if member, exist := tc.Status.TiDB.Members[podName]; !exist || !member.Health {
				budget--
				if blocked == "" {
					blocked = fmt.Sprintf("tidbcluster: [%s/%s]'s tidb upgraded pod: [%s] is not ready", ns, tcName, podName)
				}
			}
			continue
		}
		pending = append(pending, i)
	}

	if len(pending) == 0 {
		if blocked != "" {
			// every pod is already on the update revision, but not all of
			// them are serving yet: keep requeueing, as the serial upgrade
			// did, until the last pod is available and healthy
			return controller.RequeueErrorf("%s", blocked)
		}
		return nil
	}
	if budget <= 0 {
		return controller.RequeueErrorf("%s", blocked)
	}
	if budget > len(pending) {
		budget = len(pending)
	}
	return u.upgradeTiDBPods(tc, pending[:budget], newSet, maxUnavailable)
}

// upgradeTiDBPods exposes the given outdated pods (largest ordinal first) to
// the update revision by lowering the StatefulSet partition to the smallest
// chosen ordinal.
//
// Both the native and the advanced statefulset controller delete only ONE
// outdated pod per sync and wait for its replacement to become healthy, so a
// lower partition alone still restarts pods strictly one at a time. When more
// than one pod may be taken down, delete the exposed pods directly: their
// recreation is handled by the statefulset controller, and with the default
// Parallel pod management policy the replacements are created concurrently.
func (u *tidbUpgrader) upgradeTiDBPods(tc *v1alpha1.TidbCluster, ordinals []int32, newSet *apps.StatefulSet, maxUnavailable int) error {
	mngerutils.SetUpgradePartition(newSet, ordinals[len(ordinals)-1])
	if maxUnavailable <= 1 {
		// classic behavior: leave the deletion to the statefulset controller
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	for _, ordinal := range ordinals {
		podName := tidbPodName(tcName, ordinal)
		if err := u.deps.KubeClientset.CoreV1().Pods(ns).Delete(context.TODO(), podName, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("tidbUpgrader.Upgrade: failed to delete pod %s for parallel upgrade of cluster %s/%s, error: %s", podName, ns, tcName, err)
		}
		klog.Infof("tidbcluster: [%s/%s] deleted tidb pod: [%s] for parallel rolling update (maxUnavailable=%d)", ns, tcName, podName, maxUnavailable)
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
