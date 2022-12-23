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
	"strconv"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/tiflashapi"
	"github.com/pingcap/tidb-operator/pkg/util/cmpver"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

const (
	// TODO: change to use minReadySeconds in sts spec
	// See https://kubernetes.io/blog/2021/08/27/minreadyseconds-statefulsets/
	annoKeyTiFlashMinReadySeconds = "tidb.pingcap.com/tiflash-min-ready-seconds"
)

var (
	// the first version that tiflash support `tiflash/store-status` api.
	// https://github.com/pingcap/tidb-operator/issues/4159
	tiflashEqualOrGreaterThanV512, _ = cmpver.NewConstraint(cmpver.GreaterOrEqual, "v5.1.2")
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

	if tc.Status.PD.Phase == v1alpha1.UpgradePhase || tc.Status.PD.Phase == v1alpha1.ScalePhase ||
		tc.TiFlashScaling() {
		klog.Infof("TidbCluster: [%s/%s]'s pd status is %s, tiflash status is %s, can not upgrade tiflash",
			ns, tcName,
			tc.Status.PD.Phase, tc.Status.TiFlash.Phase)
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

	minReadySeconds := 0
	s, ok := tc.Annotations[annoKeyTiFlashMinReadySeconds]
	if ok {
		i, err := strconv.Atoi(s)
		if err != nil {
			klog.Warningf("tidbcluster: [%s/%s] annotation %s should be an integer: %v", ns, tcName, annoKeyTiFlashMinReadySeconds, err)
		} else {
			minReadySeconds = i
		}
	}

	mngerutils.SetUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()
	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		store := getTiFlashStoreByOrdinal(tc.GetName(), tc.Status.TiFlash, i)
		if store == nil {
			mngerutils.SetUpgradePartition(newSet, i)
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
			if !podutil.IsPodAvailable(pod, int32(minReadySeconds), metav1.Now()) {
				readyCond := podutil.GetPodReadyCondition(pod.Status)
				if readyCond == nil || readyCond.Status != corev1.ConditionTrue {
					return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded tiflash pod: [%s] is not ready", ns, tcName, podName)

				}
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded TiFlash pod: [%s] is not available, last transition time is %v", ns, tcName, podName, readyCond.LastTransitionTime)
			}
			if store.State != v1alpha1.TiKVStateUp {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded TiFlash pod: [%s], store state is not UP", ns, tcName, podName)
			}

			if larger, err := tiflashEqualOrGreaterThanV512.Check(tc.TiFlashVersion()); err == nil && larger {
				status, err := u.deps.TiFlashControl.GetTiFlashPodClient(tc.Namespace, tc.Name, podName, tc.IsTLSClusterEnabled()).GetStoreStatus()
				if err != nil {
					return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded TiFlash pod: [%s], get store status failed: %s", ns, tcName, podName, err)
				}

				if status != tiflashapi.Running {
					return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded TiFlash pod: [%s], store status is %s instead of Running", ns, tcName, podName, status)
				}
			}

			continue
		}

		mngerutils.SetUpgradePartition(newSet, i)
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
