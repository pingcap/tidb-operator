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

	// maxUnavailable is how many pods may be down at the same time during the
	// rolling update. 1 (the default) is the classic strictly-serial rolling
	// update. Whatever spec.tidb.upgradePolicy.maxUnavailable says, at least
	// one pod is always kept out of the restart window, so a single-replica
	// cluster is always upgraded serially.
	maxUnavailable := 1
	if len(podOrdinals) > 1 {
		maxUnavailable = tc.Spec.TiDB.GetUpgradeMaxUnavailable(len(podOrdinals))
		if maxUnavailable > len(podOrdinals)-1 {
			maxUnavailable = len(podOrdinals) - 1
		}
	}
	if maxUnavailable > 1 {
		return u.upgradeParallel(tc, oldSet, newSet, podOrdinals, minReadySeconds, maxUnavailable)
	}

	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		podName := tidbPodName(tcName, i)
		pod, err := u.deps.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return fmt.Errorf("tidbUpgrader.Upgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
		}
		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == tc.Status.TiDB.StatefulSet.UpdateRevision {
			if !k8s.IsPodAvailable(pod, int32(minReadySeconds), metav1.Now()) {
				readyCond := k8s.GetPodReadyCondition(pod.Status)
				if readyCond == nil || readyCond.Status != corev1.ConditionTrue {
					return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded tidb pod: [%s] is not ready", ns, tcName, podName)

				}
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded tidb pod: [%s] is not available, last transition time is %v", ns, tcName, podName, readyCond.LastTransitionTime)
			}
			if member, exist := tc.Status.TiDB.Members[podName]; !exist || !member.Health {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb upgraded pod: [%s] is not ready", ns, tcName, podName)
			}
			continue
		}
		return u.upgradeTiDBPod(tc, i, newSet)
	}

	return nil
}

func (u *tidbUpgrader) upgradeTiDBPod(tc *v1alpha1.TidbCluster, ordinal int32, newSet *apps.StatefulSet) error {
	mngerutils.SetUpgradePartition(newSet, ordinal)
	return nil
}

// upgradeParallel is the rolling update used when more than one pod may be
// down at the same time (spec.tidb.upgradePolicy.maxUnavailable > 1).
//
// Both the native and the advanced statefulset controller delete only ONE
// outdated pod per sync and wait for its replacement to become healthy, so a
// lower partition alone still restarts pods strictly one at a time. To reach
// the requested concurrency the operator deletes the exposed outdated pods
// itself: their recreation is handled by the statefulset controller, and with
// the default Parallel pod management policy the replacements are created
// concurrently.
//
// Pods of ANY revision that are out of service -- deleted and not recreated,
// terminating, not yet available, or unhealthy members -- count against
// maxUnavailable, so the number of pods simultaneously down never exceeds it.
// Restarting an outdated pod that is already down consumes nothing extra.
func (u *tidbUpgrader) upgradeParallel(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet, podOrdinals []int32, minReadySeconds int, maxUnavailable int) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	// pods already out of service, whatever their revision
	unavailable := 0
	// requeue message for the first (largest ordinal) pod found down
	blocked := ""
	// outdated pods, largest ordinal first
	type candidate struct {
		ordinal   int32
		pod       *corev1.Pod
		available bool
	}
	var pending []candidate

	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		podName := tidbPodName(tcName, i)
		pod, err := u.deps.PodLister.Pods(ns).Get(podName)
		if err != nil {
			if errors.IsNotFound(err) {
				// deleted for the upgrade and not recreated yet
				unavailable++
				if blocked == "" {
					blocked = fmt.Sprintf("tidbcluster: [%s/%s]'s tidb pod: [%s] is being recreated", ns, tcName, podName)
				}
				continue
			}
			return fmt.Errorf("tidbUpgrader.Upgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
		}
		if pod.DeletionTimestamp != nil {
			unavailable++
			if blocked == "" {
				blocked = fmt.Sprintf("tidbcluster: [%s/%s]'s tidb pod: [%s] is terminating", ns, tcName, podName)
			}
			continue
		}
		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		available := k8s.IsPodAvailable(pod, int32(minReadySeconds), metav1.Now())
		if available {
			if member, exist := tc.Status.TiDB.Members[podName]; !exist || !member.Health {
				available = false
			}
		}
		if revision == tc.Status.TiDB.StatefulSet.UpdateRevision {
			if !available {
				unavailable++
				if blocked == "" {
					blocked = fmt.Sprintf("tidbcluster: [%s/%s]'s upgraded tidb pod: [%s] is not ready", ns, tcName, podName)
				}
			}
			continue
		}
		if !available {
			unavailable++
			if blocked == "" {
				blocked = fmt.Sprintf("tidbcluster: [%s/%s]'s outdated tidb pod: [%s] is not ready", ns, tcName, podName)
			}
		}
		pending = append(pending, candidate{ordinal: i, pod: pod, available: available})
	}

	if len(pending) == 0 {
		if blocked != "" {
			// every pod is on the update revision but not all of them are
			// serving yet: keep requeueing until the last one recovers
			return controller.RequeueErrorf("%s", blocked)
		}
		return nil
	}

	// Choose how many of the highest outdated ordinals to expose. The exposed
	// set must be the largest pending ordinals: the partition exposes every
	// ordinal at or above it, and everything above the highest pending pod is
	// already updated. Taking down an available pod costs one unit of the
	// budget; an outdated pod that is already down is free to restart.
	selected := 0
	for _, c := range pending {
		cost := 0
		if c.available {
			cost = 1
		}
		if unavailable+cost > maxUnavailable {
			break
		}
		unavailable += cost
		selected++
	}
	if selected == 0 {
		return controller.RequeueErrorf("%s", blocked)
	}
	mngerutils.SetUpgradePartition(newSet, pending[selected-1].ordinal)

	// The lowered partition reaches the statefulset only after this sync, in
	// UpdateStatefulSetWithPrecheck: a pod deleted while the persisted
	// partition is still above its ordinal would be recreated at the OLD
	// revision. Delete only the pods the persisted partition already exposes;
	// the newly exposed ones are deleted on the next sync.
	persistedPartition := *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition
	for _, c := range pending[:selected] {
		if c.ordinal < persistedPartition {
			continue
		}
		// precondition on the UID the lister observed: if the pod has been
		// deleted and recreated since, the delete fails with a conflict
		// instead of taking down the new pod
		opts := metav1.DeleteOptions{Preconditions: metav1.NewUIDPreconditions(string(c.pod.UID))}
		if err := u.deps.KubeClientset.CoreV1().Pods(ns).Delete(context.TODO(), c.pod.Name, opts); err != nil {
			if errors.IsNotFound(err) || errors.IsConflict(err) {
				klog.Infof("tidbcluster: [%s/%s] tidb pod: [%s] is already gone or replaced, skipping its deletion", ns, tcName, c.pod.Name)
				continue
			}
			return fmt.Errorf("tidbUpgrader.Upgrade: failed to delete pod %s for parallel upgrade of cluster %s/%s, error: %s", c.pod.Name, ns, tcName, err)
		}
		klog.Infof("tidbcluster: [%s/%s] deleted tidb pod: [%s] for parallel rolling update (maxUnavailable=%d)", ns, tcName, c.pod.Name, maxUnavailable)
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
