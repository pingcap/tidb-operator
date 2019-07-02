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

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pdapi"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	// EvictLeaderBeginTime is the key of evict Leader begin time
	EvictLeaderBeginTime = "evictLeaderBeginTime"
	// EvictLeaderTimeout is the timeout limit of evict leader
	EvictLeaderTimeout = 3 * time.Minute
)

type tikvUpgrader struct {
	pdControl  pdapi.PDControlInterface
	podControl controller.PodControlInterface
	podLister  corelisters.PodLister
}

// NewTiKVUpgrader returns a tikv Upgrader
func NewTiKVUpgrader(pdControl pdapi.PDControlInterface,
	podControl controller.PodControlInterface,
	podLister corelisters.PodLister) Upgrader {
	return &tikvUpgrader{
		pdControl:  pdControl,
		podControl: podControl,
		podLister:  podLister,
	}
}

func (tku *tikvUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	if tc.Status.PD.Phase == v1alpha1.UpgradePhase {
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	if !tc.Status.TiKV.Synced {
		return fmt.Errorf("Tidbcluster: [%s/%s]'s tikv status sync failed, can not to be upgraded", ns, tcName)
	}

	tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
	if !templateEqual(newSet.Spec.Template, oldSet.Spec.Template) {
		return nil
	}

	setUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	for i := tc.Status.TiKV.StatefulSet.Replicas - 1; i >= 0; i-- {
		store := tku.getStoreByOrdinal(tc, i)
		if store == nil {
			continue
		}
		podName := tikvPodName(tcName, i)
		pod, err := tku.podLister.Pods(ns).Get(podName)
		if err != nil {
			return err
		}
		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tikv pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == tc.Status.TiKV.StatefulSet.UpdateRevision {

			if pod.Status.Phase != corev1.PodRunning {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded tikv pod: [%s] is not running", ns, tcName, podName)
			}
			if store.State != v1alpha1.TiKVStateUp {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded tikv pod: [%s] is not all ready", ns, tcName, podName)
			}

			continue
		}

		return tku.upgradeTiKVPod(tc, i, newSet)
	}

	return nil
}

func (tku *tikvUpgrader) upgradeTiKVPod(tc *v1alpha1.TidbCluster, ordinal int32, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	upgradePodName := tikvPodName(tcName, ordinal)
	upgradePod, err := tku.podLister.Pods(ns).Get(upgradePodName)
	if err != nil {
		return err
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == upgradePodName {
			storeID, err := strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				return err
			}
			_, evicting := upgradePod.Annotations[EvictLeaderBeginTime]

			if tku.readyToUpgrade(upgradePod, store) {
				err := tku.endEvictLeader(tc, ordinal)
				if err != nil {
					return err
				}
				setUpgradePartition(newSet, ordinal)
				return nil
			}

			if !evicting {
				return tku.beginEvictLeader(tc, storeID, upgradePod)
			}
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tikv pod: [%s] is evicting leader", ns, tcName, upgradePodName)
		}
	}

	return controller.RequeueErrorf("tidbcluster: [%s/%s] no store status found for tikv pod: [%s]", ns, tcName, upgradePodName)
}

func (tku *tikvUpgrader) readyToUpgrade(upgradePod *corev1.Pod, store v1alpha1.TiKVStore) bool {
	if store.LeaderCount == 0 {
		return true
	}
	if evictLeaderBeginTimeStr, evicting := upgradePod.Annotations[EvictLeaderBeginTime]; evicting {
		evictLeaderBeginTime, err := time.Parse(time.RFC3339, evictLeaderBeginTimeStr)
		if err != nil {
			glog.Errorf("parse annotation:[%s] to time failed.", EvictLeaderBeginTime)
			return false
		}
		if time.Now().After(evictLeaderBeginTime.Add(EvictLeaderTimeout)) {
			return true
		}
	}
	return false
}

func (tku *tikvUpgrader) beginEvictLeader(tc *v1alpha1.TidbCluster, storeID uint64, pod *corev1.Pod) error {
	err := controller.GetPDClient(tku.pdControl, tc).BeginEvictLeader(storeID)
	if err != nil {
		return err
	}
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	pod.Annotations[EvictLeaderBeginTime] = time.Now().Format(time.RFC3339)
	_, err = tku.podControl.UpdatePod(tc, pod)
	return err
}

func (tku *tikvUpgrader) endEvictLeader(tc *v1alpha1.TidbCluster, ordinal int32) error {
	store := tku.getStoreByOrdinal(tc, ordinal)
	storeID, err := strconv.ParseUint(store.ID, 10, 64)
	if err != nil {
		return err
	}
	upgradedPodName := tikvPodName(tc.GetName(), ordinal)
	upgradedPod, err := tku.podLister.Pods(tc.GetNamespace()).Get(upgradedPodName)
	if err != nil {
		return err
	}
	_, evicting := upgradedPod.Annotations[EvictLeaderBeginTime]
	if evicting {
		delete(upgradedPod.Annotations, EvictLeaderBeginTime)
		_, err = tku.podControl.UpdatePod(tc, upgradedPod)
		if err != nil {
			return err
		}
	}
	err = controller.GetPDClient(tku.pdControl, tc).EndEvictLeader(storeID)
	if err != nil {
		return err
	}

	return nil
}

func (tku *tikvUpgrader) getStoreByOrdinal(tc *v1alpha1.TidbCluster, ordinal int32) *v1alpha1.TiKVStore {
	podName := tikvPodName(tc.GetName(), ordinal)
	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == podName {
			return &store
		}
	}
	return nil
}

type fakeTiKVUpgrader struct{}

// NewFakeTiKVUpgrader returns a fake tikv upgrader
func NewFakeTiKVUpgrader() Upgrader {
	return &fakeTiKVUpgrader{}
}

func (tku *fakeTiKVUpgrader) Upgrade(tc *v1alpha1.TidbCluster, _ *apps.StatefulSet, _ *apps.StatefulSet) error {
	tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
	return nil
}
