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
	component := context.component

	switch component {
	case label.PDLabelVal:
		return ComponentGracefulUpgrade(context, oldSet, newSet)
	case label.TiKVLabelVal:
		return componentTiKVUpgrade(context, oldSet, newSet)
	case label.TiFlashLabelVal:
		return componentTiFlashUpgrade(context, oldSet, newSet)
	case label.TiDBLabelVal:
		return componentTiDBUpgrade(context, oldSet, newSet)
	}
	return nil
}

func componentTiKVUpgrade(context *ComponentContext, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc := context.tc
	dependencies := context.dependencies

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	var status *v1alpha1.TiKVStatus
	if tc.Status.PD.Phase == v1alpha1.UpgradePhase || tc.TiKVScaling() {
		klog.Infof("TidbCluster: [%s/%s]'s pd status is %v, tikv status is %v, can not upgrade tikv",
			ns, tcName, tc.Status.PD.Phase, tc.Status.TiKV.Phase)
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}
	status = &tc.Status.TiKV

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
		store := componentGetStoreByOrdinal(tc.GetName(), *status, i)
		if store == nil {
			setUpgradePartition(newSet, i)
			continue
		}
		podName := TikvPodName(tcName, i)
		pod, err := dependencies.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return fmt.Errorf("tikvUpgrader.Upgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
		}
		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tikv pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == status.StatefulSet.UpdateRevision {

			if pod.Status.Phase != corev1.PodRunning {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded tikv pod: [%s] is not running", ns, tcName, podName)
			}
			if store.State != v1alpha1.TiKVStateUp {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded tikv pod: [%s] is not all ready", ns, tcName, podName)
			}

			continue
		}

		if dependencies.CLIConfig.PodWebhookEnabled {
			setUpgradePartition(newSet, i)
			return nil
		}
		return componentUpgradeTiKVPod(tc, i, newSet)
	}

	return nil
}

func componentTiFlashUpgrade(context *ComponentContext, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc := context.tc

	//  Wait for PD, TiKV and TiDB to finish upgrade
	if tc.Status.PD.Phase == v1alpha1.UpgradePhase || tc.Status.TiKV.Phase == v1alpha1.UpgradePhase ||
		tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}
	return nil
}

func componentTiDBUpgrade(context *ComponentContext, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc := context.tc
	dependencies := context.dependencies

	// when scale replica to 0 , all nodes crash and tidb is in upgrade phase, this method will throw error about pod is upgrade.
	// so  directly return nil when scale replica to 0.
	if tc.Spec.TiDB.Replicas == int32(0) {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Status.PD.Phase == v1alpha1.UpgradePhase || tc.Status.TiKV.Phase == v1alpha1.UpgradePhase ||
		tc.Status.Pump.Phase == v1alpha1.UpgradePhase || tc.TiDBScaling() {
		klog.Infof("TidbCluster: [%s/%s]'s pd status is %s, tikv status is %s, pump status is %s,"+
			"tidb status is %s, can not upgrade tidb", ns, tcName, tc.Status.PD.Phase, tc.Status.TiKV.Phase,
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

	if oldSet.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType || oldSet.Spec.UpdateStrategy.RollingUpdate == nil {
		// Manually bypass tidb-operator to modify statefulset directly, such as modify tidb statefulset's RollingUpdate strategy to OnDelete strategy,
		// or set RollingUpdate to nil, skip tidb-operator's rolling update logic in order to speed up the upgrade in the test environment occasionally.
		// If we encounter this situation, we will let the native statefulset controller do the upgrade completely, which may be unsafe for upgrading tidb.
		// Therefore, in the production environment, we should try to avoid modifying the tidb statefulset update strategy directly.
		newSet.Spec.UpdateStrategy = oldSet.Spec.UpdateStrategy
		klog.Warningf("tidbcluster: [%s/%s] tidb statefulset %s UpdateStrategy has been modified manually", ns, tcName, oldSet.GetName())
		return nil
	}

	setUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()
	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		podName := tidbPodName(tcName, i)
		pod, err := dependencies.PodLister.Pods(ns).Get(podName)
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
		return componentUpgradeTiDBPod(tc, i, newSet)
	}

	return nil
}

func ComponentGracefulUpgrade(context *ComponentContext, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	// context deserialization
	tc := context.tc
	dependencies := context.dependencies

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	if !tc.Status.PD.Synced {
		return fmt.Errorf("tidbcluster: [%s/%s]'s pd status sync failed, can not to be upgraded", ns, tcName)
	}
	if tc.PDScaling() {
		klog.Infof("TidbCluster: [%s/%s]'s pd is scaling, can not upgrade pd",
			ns, tcName)
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	if !templateEqual(newSet, oldSet) {
		return nil
	}

	if tc.Status.PD.StatefulSet.UpdateRevision == tc.Status.PD.StatefulSet.CurrentRevision {
		return nil
	}

	if oldSet.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType || oldSet.Spec.UpdateStrategy.RollingUpdate == nil {
		// Manually bypass tidb-operator to modify statefulset directly, such as modify pd statefulset's RollingUpdate straregy to OnDelete strategy,
		// or set RollingUpdate to nil, skip tidb-operator's rolling update logic in order to speed up the upgrade in the test environment occasionally.
		// If we encounter this situation, we will let the native statefulset controller do the upgrade completely, which may be unsafe for upgrading pd.
		// Therefore, in the production environment, we should try to avoid modifying the pd statefulset update strategy directly.
		newSet.Spec.UpdateStrategy = oldSet.Spec.UpdateStrategy
		klog.Warningf("tidbcluster: [%s/%s] pd statefulset %s UpdateStrategy has been modified manually", ns, tcName, oldSet.GetName())
		return nil
	}

	setUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()
	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		podName := PdPodName(tcName, i)
		pod, err := dependencies.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return fmt.Errorf("gracefulUpgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
		}

		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == tc.Status.PD.StatefulSet.UpdateRevision {
			if member, exist := tc.Status.PD.Members[PdName(tc.Name, i, tc.Namespace, tc.Spec.ClusterDomain)]; !exist || !member.Health {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd upgraded pod: [%s] is not ready", ns, tcName, podName)
			}
			continue
		}

		if dependencies.CLIConfig.PodWebhookEnabled {
			setUpgradePartition(newSet, i)
			return nil
		}

		return ComponentUpgradePDPod(context, i, newSet)
	}

	return nil
}

func ComponentUpgradePDPod(context *ComponentContext, ordinal int32, newSet *apps.StatefulSet) error {
	// context deserialization
	tc := context.tc

	ns := tc.GetNamespace()
	tcName := tc.GetName()
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
	setUpgradePartition(newSet, ordinal)
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

func componentUpgradeTiKVPod(tc *v1alpha1.TidbCluster, ordinal int32, newSet *apps.StatefulSet) error {
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

			if u.readyToUpgrade(upgradePod, store, tc.TiKVEvictLeaderTimeout()) {
				err := u.endEvictLeader(tc, ordinal)
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

func componentUpgradeTiDBPod(tc *v1alpha1.TidbCluster, ordinal int32, newSet *apps.StatefulSet) error {
	setUpgradePartition(newSet, ordinal)
	return nil
}

func readyToUpgrade(upgradePod *corev1.Pod, store v1alpha1.TiKVStore, evictLeaderTimeout time.Duration) bool {
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

func beginEvictLeader(tc *v1alpha1.TidbCluster, storeID uint64, pod *corev1.Pod) error {
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

func endEvictLeader(tc *v1alpha1.TidbCluster, ordinal int32) error {
	// wait 5 second before delete evict scheduler，it is for auto test can catch these info
	if u.deps.CLIConfig.TestMode {
		time.Sleep(5 * time.Second)
	}
	store := u.getStoreByOrdinal(tc.GetName(), tc.Status.TiKV, ordinal)
	storeID, err := strconv.ParseUint(store.ID, 10, 64)
	if err != nil {
		return err
	}

	if tc.IsHeterogeneous() {
		err = u.deps.PDControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.Spec.Cluster.Name, tc.IsTLSClusterEnabled()).EndEvictLeader(storeID)
	} else {
		err = u.deps.PDControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled()).EndEvictLeader(storeID)
	}

	if err != nil {
		klog.Errorf("tikv upgrader: failed to end evict leader storeID: %d ordinal: %d, %v", storeID, ordinal, err)
		return err
	}
	klog.Infof("tikv upgrader: end evict leader storeID: %d ordinal: %d successfully", storeID, ordinal)
	return nil
}
