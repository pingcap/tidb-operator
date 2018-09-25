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
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
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

	if !tc.Status.TiKV.SyncSuccess {
		return fmt.Errorf("Tidbcluster: [%s/%s]'s tikv status sync failed,can not to be upgraded", ns, tcName)
	}

	tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
	setUpgradePartition(newSet, int(*oldSet.Spec.UpdateStrategy.RollingUpdate.Partition))

	upgradePod, err := tku.findUpgradePod(tc)
	if upgradePod == nil || err != nil {
		return err
	}

	storeIDStr, exist := upgradePod.Labels[label.StoreIDLabelKey]
	if !exist {
		return fmt.Errorf("Tidbcluster: [%s/%s]'s tikv Pod:[%s] does not contain storeID", ns, tcName, upgradePod.GetName())
	}
	upgradeOrdinal, err := getOrdinal(upgradePod)
	if err != nil {
		return err
	}

	if _, exist := tc.Status.TiKV.TombstoneStores[storeIDStr]; exist {
		setUpgradePartition(newSet, upgradeOrdinal)
		return nil
	}

	if store, exist := tc.Status.TiKV.Stores[storeIDStr]; exist {
		storeID, err := strconv.ParseUint(storeIDStr, 10, 64)
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

func (tku *tikvUpgrader) findUpgradePod(tc *v1alpha1.TidbCluster) (*corev1.Pod, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	selector, err := label.New().Cluster(tcName).TiKV().Selector()
	if err != nil {
		return nil, err
	}
	pdPods, err := tku.podLister.Pods(ns).List(selector)
	if err != nil {
		return nil, err
	}
	sort.Sort(descendingOrdinal(pdPods))

	for _, pod := range pdPods {
		podName := pod.GetName()
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return nil, fmt.Errorf("Tidbcluster: [%s/%s]'s pod[%s] have not label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}
		storeID := tku.getStoreID(tc, podName)
		if storeID == "" {
			return nil, fmt.Errorf("Tidbcluster: [%s/%s]'s pod[%s] have not label: %s", ns, tcName, podName, label.StoreIDLabelKey)
		}

		if revisionHash == tc.Status.TiKV.StatefulSet.UpdateRevision {
			if store, exist := tc.Status.TiKV.Stores[storeID]; exist {
				if store.State == metapb.StoreState_name[int32(metapb.StoreState_Up)] {
					continue
				}
				return nil, nil
			}
		} else {
			return pod, nil
		}
	}
	return nil, nil
}

func (tku *tikvUpgrader) needForce(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	selector, err := label.New().Cluster(ns).TiKV().Selector()
	if err != nil {
		return false, err
	}
	pdPods, err := tku.podLister.Pods(ns).List(selector)
	if err != nil {
		return false, err
	}
	for _, pod := range pdPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, fmt.Errorf("Tidbcluster: [%s/%s]'s pod:[%s] have not label: %s", ns, tcName, pod.GetName(), apps.ControllerRevisionHashLabelKey)
		}
		if revisionHash == tc.Status.TiKV.StatefulSet.CurrentRevision {
			return imagePullFailed(pod), nil
		}
	}
	return false, nil
}

func (tku *tikvUpgrader) getStoreID(tc *v1alpha1.TidbCluster, podName string) string {
	for storeID, store := range tc.Status.TiKV.Stores {
		if store.PodName == podName {
			return storeID
		}
	}
	for storeID, store := range tc.Status.TiKV.TombstoneStores {
		if store.PodName == podName {
			return storeID
		}
	}
	return ""
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
