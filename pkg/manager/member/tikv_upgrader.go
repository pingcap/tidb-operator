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
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	EvictLeaderBeginTime = "evictLeaderBeginTime"
	EvictLeaderTimeOut   = 3 * time.Minute
)

type tikvUpgrader struct {
	pdControl  controller.PDControlInterface
	podControl controller.PodControlInterface
	podLister  corelisters.PodLister
}

// NewTiKVUpgrader returns a tikv Upgrader
func NewTiKVUpgrader(pdControl controller.PDControlInterface,
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
		return fmt.Errorf("Tidbcluster: [%s/%s]'s tikv status sync failed,can not to be upgraded", ns, tcName)
	}

	tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
	setUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)

	if tc.Status.TiKV.StatefulSet.CurrentReplicas == 0 {
		return nil
	}

	upgradeOrdinal, err := tku.getUpgradeOrdinal(tc)
	if err != nil {
		return err
	}

	upgradePodName := tikvPodName(tc, upgradeOrdinal)
	upgradePod, err := tku.podLister.Pods(ns).Get(upgradePodName)
	if err != nil {
		return err
	}

	for _, store := range tc.Status.TiKV.TombstoneStores {
		if store.PodName == upgradePodName {
			setUpgradePartition(newSet, upgradeOrdinal)
			return nil
		}
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == upgradePodName {
			storeID, err := strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				return err
			}
			_, evicting := upgradePod.Annotations[EvictLeaderBeginTime]

			if tku.readyToUpgrade(upgradePod, store) {
				err = tku.pdControl.GetPDClient(tc).EndEvictLeader(storeID)
				if err != nil {
					return err
				}
				if evicting {
					delete(upgradePod.Annotations, EvictLeaderBeginTime)
					_, err = tku.podControl.UpdatePod(tc, upgradePod)
					if err != nil {
						return err
					}
				}
				setUpgradePartition(newSet, upgradeOrdinal)
				return nil
			}

			if !evicting {
				err = tku.pdControl.GetPDClient(tc).BeginEvictLeader(storeID)
				if err != nil {
					return err
				}
				// updete upgradePod
				if upgradePod.Annotations == nil {
					upgradePod.Annotations = map[string]string{}
				}
				upgradePod.Annotations[EvictLeaderBeginTime] = time.Now().Format(time.RFC3339)
				_, err = tku.podControl.UpdatePod(tc, upgradePod)
				if err != nil {
					return err
				}
				return nil
			}
		}
	}

	return nil
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
		if time.Now().After(evictLeaderBeginTime.Add(EvictLeaderTimeOut)) {
			return true
		}
	}
	return false
}

func (tku *tikvUpgrader) getUpgradeOrdinal(tc *v1alpha1.TidbCluster) (int32, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Status.TiKV.StatefulSet.UpdatedReplicas+tc.Status.TiKV.StatefulSet.CurrentReplicas != tc.Status.TiKV.StatefulSet.Replicas {
		return -1, controller.RequeueErrorf("tidbcluster: [%s/%s]'s tikv pods are not all created", ns, tcName)
	}
	for i := tc.Status.TiKV.StatefulSet.Replicas; i > tc.Status.TiKV.StatefulSet.CurrentReplicas; i-- {
		store := tku.getStoreByOrdinal(tc, i-1)
		if store == nil || store.State != metapb.StoreState_name[int32(metapb.StoreState_Up)] {
			return -1, controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded tikv pods are not all ready", ns, tcName)
		}
	}
	return tc.Status.TiKV.StatefulSet.CurrentReplicas - 1, nil
}

func (tku *tikvUpgrader) getStoreByOrdinal(tc *v1alpha1.TidbCluster, ordinal int32) *v1alpha1.TiKVStore {
	podName := tikvPodName(tc, ordinal)
	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == podName {
			return &store
		}
	}
	for _, store := range tc.Status.TiKV.TombstoneStores {
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
