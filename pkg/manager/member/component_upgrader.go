// Copyright 2021 PingCAP, Inc.
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
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

func ComponentUpgrade(context *ComponentContext, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc := context.tc
	dependencies := context.dependencies
	component := context.component

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if component == label.TiDBLabelVal {
		// when scale replica to 0 , all nodes crash and tidb is in upgrade phase, this method will throw error about pod is upgrade.
		// so  directly return nil when scale replica to 0.
		if tc.Spec.TiDB.Replicas == int32(0) {
			return nil
		}
	}

	if componentUpgradeCheckPrequisiteBeforeUpgrade(context) {
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	if err := componentUpgradeCheckPaused(context); err != nil {
		return err
	}

	componentUpgradeSyncStatusPhase(context)

	if !templateEqual(newSet, oldSet) {
		return nil
	}

	if componentUpgradeCheckStatefulsetUpdateRevision(context) {
		return nil
	}

	if oldSet.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType || oldSet.Spec.UpdateStrategy.RollingUpdate == nil {
		// Manually bypass tidb-operator to modify statefulset directly, such as modify tikv statefulset's RollingUpdate strategy to OnDelete strategy,
		// or set RollingUpdate to nil, skip tidb-operator's rolling update logic in order to speed up the upgrade in the test environment occasionally.
		// If we encounter this situation, we will let the native statefulset controller do the upgrade completely, which may be unsafe for upgrading tikv.
		// Therefore, in the production environment, we should try to avoid modifying the tikv statefulset update strategy directly.
		newSet.Spec.UpdateStrategy = oldSet.Spec.UpdateStrategy
		klog.Warningf("tidbcluster: [%s/%s] statefulset %s UpdateStrategy has been modified manually", ns, tcName, oldSet.GetName())
		return nil
	}

	setUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()

	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		if componentUpgradeCheckIfPodCanUpgradeWithNilStore(context, i) {
			setUpgradePartition(newSet, i)
			continue
		}
		podName := getComponentUpgradePodName(context, i)

		pod, err := dependencies.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return fmt.Errorf("Upgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
		}
		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tikv pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if componentUpgradeCheckPodRevision(context, revision) {
			if pod.Status.Phase != corev1.PodRunning {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded pod: [%s] is not running", ns, tcName, podName)
			}

			if err := componentCheckIfPodAvailableAfterPodRestarted(context, i, podName); err != nil {
				return err
			}

			continue
		}

		if component != label.TiFlashLabelVal && dependencies.CLIConfig.PodWebhookEnabled {
			setUpgradePartition(newSet, i)
			return nil
		}
		return componentUpgradeToUpgradePod(context, i, newSet)
	}

	return nil
}

func ComponentTransferPDLeaderTo(context *ComponentContext, targetName string) error {
	// context deserialization
	tc := context.tc
	dependencies := context.dependencies

	return controller.GetPDClient(dependencies.PDControl, tc).TransferPDLeader(targetName)
}

func componentGetStoreByOrdinal(name string, status v1alpha1.TiKVStatus, ordinal int32) *v1alpha1.TiKVStore {
	podName := TikvPodName(name, ordinal)
	for _, store := range status.Stores {
		if store.PodName == podName {
			return &store
		}
	}
	return nil
}

func componentUpgradeTiDBPod(context *ComponentContext, ordinal int32, newSet *apps.StatefulSet) error {
	setUpgradePartition(newSet, ordinal)
	return nil
}

func componentTiKVreadyToUpgrade(upgradePod *corev1.Pod, store v1alpha1.TiKVStore, evictLeaderTimeout time.Duration) bool {
	if store.LeaderCount == 0 {
		return true
	}
	if evictLeaderBeginTimeStr, evicting := upgradePod.Annotations[EvictLeaderBeginTime]; evicting {
		evictLeaderBeginTime, err := time.Parse(time.RFC3339, evictLeaderBeginTimeStr)
		if err != nil {
			klog.Errorf("parse annotation:[%s] to time failed.", EvictLeaderBeginTime)
			return false
		}
		if time.Now().After(evictLeaderBeginTime.Add(evictLeaderTimeout)) {
			return true
		}
	}
	return false
}

func componentTiKVbeginEvictLeader(context *ComponentContext, storeID uint64, pod *corev1.Pod) error {
	tc := context.tc
	dependencies := context.dependencies

	ns := tc.GetNamespace()
	podName := pod.GetName()
	err := controller.GetPDClient(dependencies.PDControl, tc).BeginEvictLeader(storeID)
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
	_, err = dependencies.PodControl.UpdatePod(tc, pod)
	if err != nil {
		klog.Errorf("tikv upgrader: failed to set pod %s/%s annotation %s to %s, %v",
			ns, podName, EvictLeaderBeginTime, now, err)
		return err
	}
	klog.Infof("tikv upgrader: set pod %s/%s annotation %s to %s successfully",
		ns, podName, EvictLeaderBeginTime, now)
	return nil
}

func componentTiKVendEvictLeader(context *ComponentContext, ordinal int32) error {
	tc := context.tc
	dependencies := context.dependencies

	// wait 5 second before delete evict scheduler，it is for auto test can catch these info
	if dependencies.CLIConfig.TestMode {
		time.Sleep(5 * time.Second)
	}
	store := componentGetStoreByOrdinal(tc.GetName(), tc.Status.TiKV, ordinal)
	storeID, err := strconv.ParseUint(store.ID, 10, 64)
	if err != nil {
		return err
	}

	if tc.IsHeterogeneous() {
		err = dependencies.PDControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.Spec.Cluster.Name, tc.IsTLSClusterEnabled()).EndEvictLeader(storeID)
	} else {
		err = dependencies.PDControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled()).EndEvictLeader(storeID)
	}

	if err != nil {
		klog.Errorf("tikv upgrader: failed to end evict leader storeID: %d ordinal: %d, %v", storeID, ordinal, err)
		return err
	}
	klog.Infof("tikv upgrader: end evict leader storeID: %d ordinal: %d successfully", storeID, ordinal)
	return nil
}

func componentUpgradeCheckPrequisiteBeforeUpgrade(context *ComponentContext) bool {
	tc := context.tc
	component := context.component
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	var prequisite bool
	switch component {
	case label.PDLabelVal:
		prequisite = tc.PDScaling()
		if prequisite {
			klog.Infof("TidbCluster: [%s/%s]'s pd is scaling, can not upgrade pd",
				ns, tcName)
		}
	case label.TiKVLabelVal:
		prequisite = tc.Status.PD.Phase == v1alpha1.UpgradePhase || tc.TiKVScaling()
		if prequisite {
			klog.Infof("TidbCluster: [%s/%s]'s pd status is %v, tikv status is %v, can not upgrade tikv",
				ns, tcName, tc.Status.PD.Phase, tc.Status.TiKV.Phase)
		}
	case label.TiDBLabelVal:
		prequisite = tc.Status.PD.Phase == v1alpha1.UpgradePhase || tc.Status.TiKV.Phase == v1alpha1.UpgradePhase ||
			tc.Status.Pump.Phase == v1alpha1.UpgradePhase || tc.TiDBScaling()
		if prequisite {
			klog.Infof("TidbCluster: [%s/%s]'s pd status is %s, tikv status is %s, pump status is %s,"+
				"tidb status is %s, can not upgrade tidb", ns, tcName, tc.Status.PD.Phase, tc.Status.TiKV.Phase,
				tc.Status.Pump.Phase, tc.Status.TiDB.Phase)
		}
	case label.TiFlashLabelVal:
		prequisite = tc.Status.PD.Phase == v1alpha1.UpgradePhase || tc.Status.TiKV.Phase == v1alpha1.UpgradePhase ||
			tc.Status.TiDB.Phase == v1alpha1.UpgradePhase
	}

	return prequisite
}

func componentUpgradeCheckPaused(context *ComponentContext) error {
	tc := context.tc
	component := context.component

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	switch component {
	case label.PDLabelVal:
		if !tc.Status.PD.Synced {
			return fmt.Errorf("tidbcluster: [%s/%s]'s pd status sync failed, can not to be upgraded", ns, tcName)
		}
	case label.TiKVLabelVal:
		if !tc.Status.TiKV.Synced {
			return fmt.Errorf("tidbcluster: [%s/%s]'s tikv status sync failed, can not to be upgraded", ns, tcName)
		}
	case label.TiFlashLabelVal:
		if !tc.Status.TiFlash.Synced {
			return fmt.Errorf("cluster: [%s/%s]'s TiFlash status is not synced, can not upgrade", ns, tcName)
		}
	}

	return nil
}

func componentUpgradeSyncStatusPhase(context *ComponentContext) {
	tc := context.tc
	component := context.component

	switch component {
	case label.PDLabelVal:
		tc.Status.PD.Phase = v1alpha1.UpgradePhase
	case label.TiKVLabelVal:
		tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
	case label.TiDBLabelVal:
		tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
	case label.TiFlashLabelVal:
		tc.Status.TiFlash.Phase = v1alpha1.UpgradePhase
	}
}

func componentUpgradeCheckStatefulsetUpdateRevision(context *ComponentContext) bool {
	tc := context.tc
	component := context.component

	var updateVersionCheck bool
	switch component {
	case label.PDLabelVal:
		updateVersionCheck = tc.Status.PD.StatefulSet.UpdateRevision == tc.Status.PD.StatefulSet.CurrentRevision
	case label.TiKVLabelVal:
		updateVersionCheck = tc.Status.TiKV.StatefulSet.UpdateRevision == tc.Status.TiKV.StatefulSet.CurrentRevision
	case label.TiDBLabelVal:
		updateVersionCheck = tc.Status.TiDB.StatefulSet.UpdateRevision == tc.Status.TiDB.StatefulSet.CurrentRevision
	case label.TiFlashLabelVal:
		updateVersionCheck = tc.Status.TiFlash.StatefulSet.UpdateRevision == tc.Status.TiFlash.StatefulSet.CurrentRevision
	}
	return updateVersionCheck
}

func getComponentUpgradePodName(context *ComponentContext, ordinal int32) string {
	component := context.component
	tc := context.tc

	tcName := tc.Name
	var podName string

	switch component {
	case label.PDLabelVal:
		podName = PdPodName(tcName, ordinal)
	case label.TiKVLabelVal:
		podName = TikvPodName(tcName, ordinal)
	case label.TiFlashLabelVal:
		podName = TiFlashPodName(tcName, ordinal)
	}

	return podName
}

func componentUpgradeCheckPodRevision(context *ComponentContext, revision string) bool {
	tc := context.tc
	component := context.component

	var revisionCheck bool
	switch component {
	case label.PDLabelVal:
		revisionCheck = revision == tc.Status.PD.StatefulSet.UpdateRevision
	case label.TiKVLabelVal:
		revisionCheck = tc.Status.TiKV.StatefulSet.UpdateRevision == revision
	}
	return revisionCheck

}

func componentUpgradeCheckIfPodCanUpgradeWithNilStore(context *ComponentContext, ordinal int32) bool {
	tc := context.tc
	component := context.component

	var directUpgrade bool
	directUpgrade = false

	switch component {
	case label.PDLabelVal:
		directUpgrade = false
	case label.TiKVLabelVal:
		store := componentGetStoreByOrdinal(tc.GetName(), tc.Status.TiKV, ordinal)
		if store == nil {
			directUpgrade = true
		}
	case label.TiFlashLabelVal:
		store := getTiFlashStoreByOrdinal(tc.GetName(), tc.Status.TiFlash, ordinal)
		if store == nil {
			directUpgrade = true
		}
	}
	return directUpgrade
}

func componentCheckIfPodAvailableAfterPodRestarted(context *ComponentContext, ordinal int32, podName string) error {
	tc := context.tc
	component := context.component

	ns := tc.Namespace
	tcName := tc.Name

	switch component {
	case label.PDLabelVal:
		if member, exist := tc.Status.PD.Members[PdName(tc.Name, ordinal, tc.Namespace, tc.Spec.ClusterDomain)]; !exist || !member.Health {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd upgraded pod: [%s] is not ready", ns, tcName, podName)
		}
	case label.TiKVLabelVal:
		store := componentGetStoreByOrdinal(tc.GetName(), tc.Status.TiKV, ordinal)
		if store.State != v1alpha1.TiKVStateUp {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded tikv pod: [%s] is not all ready", ns, tcName, podName)
		}
	case label.TiDBLabelVal:
		if member, exist := tc.Status.TiDB.Members[podName]; !exist || !member.Health {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb upgraded pod: [%s] is not ready", ns, tcName, podName)
		}
	case label.TiFlashLabelVal:
		store := getTiFlashStoreByOrdinal(tc.GetName(), tc.Status.TiFlash, ordinal)
		if store.State != v1alpha1.TiKVStateUp {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded TiFlash pod: [%s], store state is not UP", ns, tcName, podName)
		}
	}

	return nil
}

func componentUpgradeToUpgradePod(context *ComponentContext, ordinal int32, newSet *apps.StatefulSet) error {
	component := context.component
	tc := context.tc
	dependencies := context.dependencies

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	switch component {
	case label.PDLabelVal:
		upgradePdName := PdName(tcName, ordinal, tc.Namespace, tc.Spec.ClusterDomain)
		upgradePodName := PdPodName(tcName, ordinal)
		if tc.Status.PD.Leader.Name == upgradePdName || tc.Status.PD.Leader.Name == upgradePodName {
			var targetName string
			if tc.PDStsActualReplicas() > 1 {
				targetOrdinal := helper.GetMaxPodOrdinal(*newSet.Spec.Replicas, newSet)
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
				err := ComponentTransferPDLeaderTo(context, targetName)
				if err != nil {
					klog.Errorf("pd upgrader: failed to transfer pd leader to: %s, %v", targetName, err)
					return err
				}
				klog.Infof("pd upgrader: transfer pd leader to: %s successfully", targetName)
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd member: [%s] is transferring leader to pd member: [%s]", ns, tcName, upgradePdName, targetName)
			}
		}
	case label.TiKVLabelVal:
		upgradePodName := TikvPodName(tcName, ordinal)
		upgradePod, err := dependencies.PodLister.Pods(ns).Get(upgradePodName)
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
					return componentTiKVbeginEvictLeader(context, storeID, upgradePod)
				}

				if componentTiKVreadyToUpgrade(upgradePod, store, tc.TiKVEvictLeaderTimeout()) {
					err := componentTiKVendEvictLeader(context, ordinal)
					if err != nil {
						return err
					}
					setUpgradePartition(newSet, ordinal)
					return nil
				}

				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tikv pod: [%s] is evicting leader", ns, tcName, upgradePodName)
			}
		}

		return controller.RequeueErrorf("tidbcluster: [%s/%s] no store status found for tikv pod: [%s]", ns, tcName, upgradePodName)
	}

	setUpgradePartition(newSet, ordinal)
	return nil
}
