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
	"strconv"
	"time"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

const (
	// EvictLeaderBeginTime is the key of evict Leader begin time
	EvictLeaderBeginTime = "evictLeaderBeginTime"
)

type TiKVUpgrader interface {
	Upgrade(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error
}

type tikvUpgrader struct {
	deps *controller.Dependencies
}

// NewTiKVUpgrader returns a tikv Upgrader
func NewTiKVUpgrader(deps *controller.Dependencies) TiKVUpgrader {
	return &tikvUpgrader{
		deps: deps,
	}
}

func (u *tikvUpgrader) Upgrade(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := meta.GetNamespace()
	tcName := meta.GetName()

	var status *v1alpha1.TiKVStatus
	switch meta := meta.(type) {
	case *v1alpha1.TidbCluster:
		if meta.Status.TiCDC.Phase == v1alpha1.UpgradePhase ||
			meta.Status.TiFlash.Phase == v1alpha1.UpgradePhase ||
			meta.Status.PD.Phase == v1alpha1.UpgradePhase ||
			meta.TiKVScaling() {
			klog.Infof("TidbCluster: [%s/%s]'s ticdc status is %v, "+
				"tiflash status is %v, pd status is %v, "+
				"tikv status is %v, can not upgrade tikv",
				ns, tcName, meta.Status.TiCDC.Phase, meta.Status.TiFlash.Phase,
				meta.Status.PD.Phase, meta.Status.TiKV.Phase)
			_, podSpec, err := GetLastAppliedConfig(oldSet)
			if err != nil {
				return err
			}
			newSet.Spec.Template.Spec = *podSpec
			return nil
		}
		status = &meta.Status.TiKV
	default:
		return fmt.Errorf("cluster[%s/%s] failed to upgrading tikv due to converting", meta.GetNamespace(), meta.GetName())
	}

	tc, _ := meta.(*v1alpha1.TidbCluster)

	if !status.Synced {
		return fmt.Errorf("cluster: [%s/%s]'s tikv status sync failed, can not to be upgraded", ns, tcName)
	}

	status.Phase = v1alpha1.UpgradePhase
	if !templateEqual(newSet, oldSet) {
		return nil
	}

	if status.StatefulSet.UpdateRevision == status.StatefulSet.CurrentRevision {
		return nil
	}

	if oldSet.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType || oldSet.Spec.UpdateStrategy.RollingUpdate == nil {
		// Manually bypass tidb-operator to modify statefulset directly, such as modify tikv statefulset's RollingUpdate strategy to OnDelete strategy,
		// or set RollingUpdate to nil, skip tidb-operator's rolling update logic in order to speed up the upgrade in the test environment occasionally.
		// If we encounter this situation, we will let the native statefulset controller do the upgrade completely, which may be unsafe for upgrading tikv.
		// Therefore, in the production environment, we should try to avoid modifying the tikv statefulset update strategy directly.
		newSet.Spec.UpdateStrategy = oldSet.Spec.UpdateStrategy
		klog.Warningf("tidbcluster: [%s/%s] tikv statefulset %s UpdateStrategy has been modified manually", ns, tcName, oldSet.GetName())
		return nil
	}

	setUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()
	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		store := getStoreByOrdinal(meta.GetName(), *status, i)
		if store == nil {
			setUpgradePartition(newSet, i)
			continue
		}
		podName := TikvPodName(tcName, i)
		pod, err := u.deps.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return fmt.Errorf("tikvUpgrader.Upgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
		}
		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tikv pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == status.StatefulSet.UpdateRevision {

			if !podutil.IsPodReady(pod) {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded tikv pod: [%s] is not ready", ns, tcName, podName)
			}
			if store.State != v1alpha1.TiKVStateUp {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded tikv pod: [%s] is not all ready", ns, tcName, podName)
			}

			if !u.deps.CLIConfig.PodWebhookEnabled {
				// If pods recreated successfully, endEvictLeader for the store on this Pod.
				storeID, err := strconv.ParseUint(store.ID, 10, 64)
				if err != nil {
					return err
				}
				if err := endEvictLeaderbyStoreID(u.deps, tc, storeID); err != nil {
					return err
				}
			}

			continue
		}

		if u.deps.CLIConfig.PodWebhookEnabled {
			setUpgradePartition(newSet, i)
			return nil
		}

		return u.upgradeTiKVPod(tc, i, newSet)
	}

	return nil
}

func (u *tikvUpgrader) upgradeTiKVPod(tc *v1alpha1.TidbCluster, ordinal int32, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	upgradePodName := TikvPodName(tcName, ordinal)
	upgradePod, err := u.deps.PodLister.Pods(ns).Get(upgradePodName)
	if err != nil {
		return fmt.Errorf("upgradeTiKVPod: failed to get pods %s for cluster %s/%s, error: %s", upgradePodName, ns, tcName, err)
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == upgradePodName {
			storeID, err := strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				return err
			}
			_, evicting := upgradePod.Annotations[EvictLeaderBeginTime]
			if !evicting {
				return u.beginEvictLeader(tc, storeID, upgradePod)
			}

			if u.readyToUpgrade(upgradePod, tc) {
				setUpgradePartition(newSet, ordinal)
				return nil
			}

			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tikv pod: [%s] is evicting leader", ns, tcName, upgradePodName)
		}
	}

	return controller.RequeueErrorf("tidbcluster: [%s/%s] no store status found for tikv pod: [%s]", ns, tcName, upgradePodName)
}

func (u *tikvUpgrader) readyToUpgrade(upgradePod *corev1.Pod, tc *v1alpha1.TidbCluster) bool {
	evictLeaderTimeout := tc.TiKVEvictLeaderTimeout()

	if evictLeaderBeginTimeStr, evicting := upgradePod.Annotations[EvictLeaderBeginTime]; evicting {
		evictLeaderBeginTime, err := time.Parse(time.RFC3339, evictLeaderBeginTimeStr)
		if err != nil {
			klog.Errorf("parse annotation:[%s] to time failed.", EvictLeaderBeginTime)
			return false
		}
		if time.Now().After(evictLeaderBeginTime.Add(evictLeaderTimeout)) {
			klog.Infof("Evict region leader timeout (threshold: %v) for Pod %s/%s", evictLeaderTimeout, upgradePod.Namespace, upgradePod.Name)
			return true
		}
	}

	tlsEnabled := tc.IsTLSClusterEnabled()
	leaderCount, err := u.deps.TiKVControl.GetTiKVPodClient(tc.Namespace, tc.Name, upgradePod.Name, tlsEnabled).GetLeaderCount()
	if err != nil {
		klog.Warningf("Fail to get region leader count for Pod %s/%s, error: %v", upgradePod.Namespace, upgradePod.Name, err)
		return false
	}

	if leaderCount == 0 {
		klog.Infof("Region leader count is 0 for Pod %s/%s", upgradePod.Namespace, upgradePod.Name)
		return true
	}

	klog.Infof("Region leader count is %d for Pod %s/%s", leaderCount, upgradePod.Namespace, upgradePod.Name)

	return false
}

func (u *tikvUpgrader) beginEvictLeader(tc *v1alpha1.TidbCluster, storeID uint64, pod *corev1.Pod) error {
	ns := tc.GetNamespace()
	podName := pod.GetName()
	err := controller.GetPDClient(u.deps.PDControl, tc).BeginEvictLeader(storeID)
	if err != nil {
		klog.Errorf("tikv upgrader: failed to begin evict leader: %d, %s/%s, %v",
			storeID, ns, podName, err)
		return err
	}
	klog.Infof("tikv upgrader: begin evict leader: %d, %s/%s successfully", storeID, ns, podName)
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	now := time.Now().Format(time.RFC3339)
	pod.Annotations[EvictLeaderBeginTime] = now
	_, err = u.deps.PodControl.UpdatePod(tc, pod)
	if err != nil {
		klog.Errorf("tikv upgrader: failed to set pod %s/%s annotation %s to %s, %v",
			ns, podName, EvictLeaderBeginTime, now, err)
		return err
	}
	klog.Infof("tikv upgrader: set pod %s/%s annotation %s to %s successfully",
		ns, podName, EvictLeaderBeginTime, now)
	return nil
}

func endEvictLeader(deps *controller.Dependencies, tc *v1alpha1.TidbCluster, ordinal int32) error {
	store := getStoreByOrdinal(tc.GetName(), tc.Status.TiKV, ordinal)
	if store == nil {
		klog.Errorf("tikv: no store found for TiKV ordinal %v of %s/%s", ordinal, tc.Namespace, tc.Name)
		return nil
	}
	storeID, err := strconv.ParseUint(store.ID, 10, 64)
	if err != nil {
		return err
	}

	return endEvictLeaderbyStoreID(deps, tc, storeID)
}

func endEvictLeaderbyStoreID(deps *controller.Dependencies, tc *v1alpha1.TidbCluster, storeID uint64) error {
	// wait 5 second before delete evict scheduler，it is for auto test can catch these info
	if deps.CLIConfig.TestMode {
		time.Sleep(5 * time.Second)
	}
	var err error

	if tc.HeterogeneousWithoutLocalPD() {
		err = deps.PDControl.GetPDClient(pdapi.Namespace(tc.Spec.Cluster.Namespace), tc.Spec.Cluster.Name, tc.IsTLSClusterEnabled()).EndEvictLeader(storeID)
	} else {
		err = deps.PDControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled()).EndEvictLeader(storeID)
	}

	if err != nil {
		klog.Errorf("tikv: failed to end evict leader for store: %d of %s/%s, error: %v", storeID, tc.Namespace, tc.Name, err)
		return err
	}
	klog.Infof("tikv: end evict leader for store: %d of %s/%s successfully", storeID, tc.Namespace, tc.Name)
	return nil
}

func getStoreByOrdinal(name string, status v1alpha1.TiKVStatus, ordinal int32) *v1alpha1.TiKVStore {
	podName := TikvPodName(name, ordinal)
	for _, store := range status.Stores {
		if store.PodName == podName {
			return &store
		}
	}
	return nil
}

type fakeTiKVUpgrader struct{}

// NewFakeTiKVUpgrader returns a fake tikv upgrader
func NewFakeTiKVUpgrader() TiKVUpgrader {
	return &fakeTiKVUpgrader{}
}

func (u *fakeTiKVUpgrader) Upgrade(meta metav1.Object, _ *apps.StatefulSet, _ *apps.StatefulSet) error {
	tc := meta.(*v1alpha1.TidbCluster)
	tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
	return nil
}
