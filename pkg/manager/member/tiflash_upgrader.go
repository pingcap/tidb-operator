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
	apps "k8s.io/api/apps/v1"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

type tiflashUpgrader struct {
	deps *controller.Dependencies
}

// NewTiFlashUpgrader returns a tiflash Upgrader
func NewTiFlashUpgrader(deps *controller.Dependencies) Upgrader {
	return &tiflashUpgrader{
		deps: deps,
	}
}

func (u *tiflashUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Status.TiCDC.Phase == v1alpha1.UpgradePhase ||
		tc.Status.PD.Phase == v1alpha1.UpgradePhase ||
		tc.TiFlashScaling() {
		klog.Infof("TidbCluster: [%s/%s]'s ticdc status is %s, "+
			"tiflash status is %s, can not upgrade tiflash", ns, tcName,
			tc.Status.TiCDC.Phase, tc.Status.TiFlash.Phase)
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	if !tc.Status.TiFlash.Synced {
		return fmt.Errorf("cluster: [%s/%s]'s TiFlash status is not synced, can not upgrade", ns, tcName)
	}

	tc.Status.TiFlash.Phase = v1alpha1.UpgradePhase
	if !templateEqual(newSet, oldSet) {
		return nil
	}

	if tc.Status.TiFlash.StatefulSet.UpdateRevision == tc.Status.TiFlash.StatefulSet.CurrentRevision {
		return nil
	}

	if oldSet.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType || oldSet.Spec.UpdateStrategy.RollingUpdate == nil {
		// Manually bypass tidb-operator to modify statefulset directly, such as modify tikv statefulset's RollingUpdate strategy to OnDelete strategy,
		// or set RollingUpdate to nil, skip tidb-operator's rolling update logic in order to speed up the upgrade in the test environment occasionally.
		// If we encounter this situation, we will let the native statefulset controller do the upgrade completely, which may be unsafe for upgrading tikv.
		// Therefore, in the production environment, we should try to avoid modifying the tikv statefulset update strategy directly.
		newSet.Spec.UpdateStrategy = oldSet.Spec.UpdateStrategy
		klog.Warningf("tidbcluster: [%s/%s] TiFlash statefulset %s UpdateStrategy has been modified manually", ns, tcName, oldSet.GetName())
		return nil
	}

	setUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()
	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		store := getTiFlashStoreByOrdinal(tc.GetName(), tc.Status.TiFlash, i)
		if store == nil {
			setUpgradePartition(newSet, i)
			continue
		}
		podName := TiFlashPodName(tcName, i)
		pod, err := u.deps.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return fmt.Errorf("TiFlashUpgrader.Upgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
		}
		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s TiFlash pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == tc.Status.TiFlash.StatefulSet.UpdateRevision {
			if !podutil.IsPodReady(pod) {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded TiFlash pod: [%s] is not ready", ns, tcName, podName)
			}
			if store.State != v1alpha1.TiKVStateUp {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded TiFlash pod: [%s], store state is not UP", ns, tcName, podName)
			}
			continue
		}

		setUpgradePartition(newSet, i)
		return nil
	}

	return nil
}

func getTiFlashStoreByOrdinal(name string, status v1alpha1.TiFlashStatus, ordinal int32) *v1alpha1.TiKVStore {
	podName := TiFlashPodName(name, ordinal)
	for _, store := range status.Stores {
		if store.PodName == podName {
			return &store
		}
	}
	return nil
}

type fakeTiFlashUpgrader struct{}

// NewFakeTiFlashUpgrader returns a fake tiflash upgrader
func NewFakeTiFlashUpgrader() Upgrader {
	return &fakeTiFlashUpgrader{}
}

func (u *fakeTiFlashUpgrader) Upgrade(tc *v1alpha1.TidbCluster, _ *apps.StatefulSet, _ *apps.StatefulSet) error {
	return nil
}
