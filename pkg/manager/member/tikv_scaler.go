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
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

type tikvScaler struct {
	generalScaler
	podLister corelisters.PodLister
}

// NewTiKVScaler returns a tikv Scaler
func NewTiKVScaler(pdControl pdapi.PDControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvcControl controller.PVCControlInterface,
	podLister corelisters.PodLister) Scaler {
	return &tikvScaler{generalScaler{pdControl, pvcLister, pvcControl}, podLister}
}

func (tsd *tikvScaler) Scale(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return tsd.ScaleOut(tc, oldSet, newSet)
	} else if scaling < 0 {
		return tsd.ScaleIn(tc, oldSet, newSet)
	} else {
		if tc.TiKVScaling() {
			tc.Status.TiKV.Phase = v1alpha1.NormalPhase
		}
	}
	// we only sync auto scaler annotations when we are finishing syncing scaling
	return tsd.SyncAutoScalerAnn(tc, oldSet)
}

func (tsd *tikvScaler) ScaleOut(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	if tc.TiKVUpgrading() {
		klog.Infof("the TidbCluster: [%s/%s]'s tikv is upgrading, can not scale out until the upgrade completed",
			tc.Namespace, tc.Name)
		return nil
	}
	// During TidbCluster upgrade, if TiKV is scaled at the same time, since
	// TiKV cannot be upgraded during PD upgrade, the TiKV scaling will occur
	// before the TiKV upgrade, in this case, the Pump, TiDB, TiFlash, TiCDC, etc.
	// will be upgraded before TiKV upgrade.
	// To avoid this case, we skip the scaling out during PD upgrade.
	if tc.PDUpgrading() {
		klog.Infof("the TidbCluster: [%s/%s]'s pd is upgrading, can not scale out until the upgrade completed",
			tc.Namespace, tc.Name)
		return nil
	}

	klog.Infof("scaling out tikv statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	_, err := tsd.deleteDeferDeletingPVC(tc, oldSet.GetName(), v1alpha1.TiKVMemberType, ordinal)
	if err != nil {
		return err
	}

	tc.Status.TiKV.Phase = v1alpha1.ScaleOutPhase
	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

func (tsd *tikvScaler) ScaleIn(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	// we can only remove one member at a time when scaling in
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	setName := oldSet.GetName()

	// tikv can not scale in when it is upgrading
	if tc.TiKVUpgrading() {
		klog.Infof("the TidbCluster: [%s/%s]'s tikv is upgrading,can not scale in until upgrade have completed",
			ns, tcName)
		return nil
	}
	// During TidbCluster upgrade, if TiKV is scaled at the same time, since
	// TiKV cannot be upgraded during PD upgrade, the TiKV scaling will occur
	// before the TiKV upgrade, in this case, the Pump, TiDB, TiFlash, TiCDC, etc.
	// will be upgraded before TiKV upgrade.
	// To avoid this case, we skip the scaling in during PD upgrade.
	if tc.PDUpgrading() {
		klog.Infof("the TidbCluster: [%s/%s]'s pd is upgrading, can not scale in until the upgrade completed",
			ns, tcName)
		return nil
	}

	klog.Infof("scaling in tikv statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	// We need remove member from cluster before reducing statefulset replicas
	podName := ordinalPodName(v1alpha1.TiKVMemberType, tcName, ordinal)
	pod, err := tsd.podLister.Pods(ns).Get(podName)
	if err != nil {
		return err
	}

	if controller.PodWebhookEnabled {
		tc.Status.TiKV.Phase = v1alpha1.ScaleInPhase
		setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
		return nil
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == podName {
			state := store.State
			id, err := strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				return err
			}
			if state != v1alpha1.TiKVStateOffline {
				tc.Status.TiKV.Phase = v1alpha1.ScaleInPhase
				if err := controller.GetPDClient(tsd.pdControl, tc).DeleteStore(id); err != nil {
					klog.Errorf("tikv scale in: failed to delete store %d, %v", id, err)
					return err
				}
				klog.Infof("tikv scale in: delete store %d for tikv %s/%s successfully", id, ns, podName)
			}
			return controller.RequeueErrorf("TiKV %s/%s store %d  still in cluster, state: %s", ns, podName, id, state)
		}
	}
	for id, store := range tc.Status.TiKV.TombstoneStores {
		if store.PodName == podName && pod.Labels[label.StoreIDLabelKey] == id {
			id, err := strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				return err
			}

			// TODO: double check if store is really not in Up/Offline/Down state
			klog.Infof("TiKV %s/%s store %d becomes tombstone", ns, podName, id)

			pvcName := ordinalPVCName(v1alpha1.TiKVMemberType, setName, ordinal)
			pvc, err := tsd.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
			if err != nil {
				return err
			}
			if pvc.Annotations == nil {
				pvc.Annotations = map[string]string{}
			}
			now := time.Now().Format(time.RFC3339)
			pvc.Annotations[label.AnnPVCDeferDeleting] = now
			_, err = tsd.pvcControl.UpdatePVC(tc, pvc)
			if err != nil {
				klog.Errorf("tikv scale in: failed to set pvc %s/%s annotation: %s to %s",
					ns, pvcName, label.AnnPVCDeferDeleting, now)
				return err
			}
			klog.Infof("tikv scale in: set pvc %s/%s annotation: %s to %s",
				ns, pvcName, label.AnnPVCDeferDeleting, now)
			tc.Status.TiKV.Phase = v1alpha1.ScaleInPhase
			setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
			return nil
		}
	}

	// When store not found in TidbCluster status, there are two situations as follows:
	// 1. This can happen when TiKV joins cluster but we haven't synced its status.
	//    In this situation return error to wait another round for safety.
	//
	// 2. This can happen when TiKV pod has not been successfully registered in the cluster, such as always pending.
	//    In this situation we should delete this TiKV pod immediately to avoid blocking the subsequent operations.
	if !podutil.IsPodReady(pod) {
		pvcName := ordinalPVCName(v1alpha1.TiKVMemberType, setName, ordinal)
		pvc, err := tsd.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
		if err != nil {
			return err
		}
		safeTimeDeadline := pod.CreationTimestamp.Add(5 * controller.ResyncDuration)
		if time.Now().Before(safeTimeDeadline) {
			// Wait for 5 resync periods to ensure that the following situation does not occur:
			//
			// The tikv pod starts for a while, but has not synced its status, and then the pod becomes not ready.
			// Here we wait for 5 resync periods to ensure that the status of this tikv pod has been synced.
			// After this period of time, if there is still no information about this tikv in TidbCluster status,
			// then we can be sure that this tikv has never been added to the tidb cluster.
			// So we can scale in this tikv pod safely.
			resetReplicas(newSet, oldSet)
			return fmt.Errorf("TiKV %s/%s is not ready, wait for some resync periods to synced its status", ns, podName)
		}
		if pvc.Annotations == nil {
			pvc.Annotations = map[string]string{}
		}
		now := time.Now().Format(time.RFC3339)
		pvc.Annotations[label.AnnPVCDeferDeleting] = now
		_, err = tsd.pvcControl.UpdatePVC(tc, pvc)
		if err != nil {
			klog.Errorf("pod %s not ready, tikv scale in: failed to set pvc %s/%s annotation: %s to %s",
				podName, ns, pvcName, label.AnnPVCDeferDeleting, now)
			return err
		}
		klog.Infof("pod %s not ready, tikv scale in: set pvc %s/%s annotation: %s to %s",
			podName, ns, pvcName, label.AnnPVCDeferDeleting, now)
		tc.Status.TiKV.Phase = v1alpha1.ScaleInPhase
		setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
		return nil
	}
	return fmt.Errorf("TiKV %s/%s not found in cluster", ns, podName)
}

// SyncAutoScalerAnn would reclaim the auto-scaling out slots if the target pod is no longer existed
func (tsd *tikvScaler) SyncAutoScalerAnn(tc *v1alpha1.TidbCluster, actual *apps.StatefulSet) error {
	currentScalingSlots := util.GetAutoScalingOutSlots(tc, v1alpha1.TiKVMemberType)
	if currentScalingSlots.Len() < 1 {
		return nil
	}
	currentOrdinals := helper.GetPodOrdinals(tc.Spec.TiKV.Replicas, actual)

	// reclaim the auto-scaling out slots if the target pod is no longer existed
	if !currentOrdinals.HasAll(currentScalingSlots.List()...) {
		reclaimedSlots := currentScalingSlots.Difference(currentOrdinals)
		currentScalingSlots = currentScalingSlots.Delete(reclaimedSlots.List()...)
		if currentScalingSlots.Len() < 1 {
			delete(tc.Annotations, label.AnnTiKVAutoScalingOutOrdinals)
			return nil
		}
		v, err := util.Encode(currentScalingSlots.List())
		if err != nil {
			return err
		}
		tc.Annotations[label.AnnTiKVAutoScalingOutOrdinals] = v
		return nil
	}
	return nil
}

type fakeTiKVScaler struct{}

// NewFakeTiKVScaler returns a fake tikv Scaler
func NewFakeTiKVScaler() Scaler {
	return &fakeTiKVScaler{}
}

func (fsd *fakeTiKVScaler) Scale(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if *newSet.Spec.Replicas > *oldSet.Spec.Replicas {
		return fsd.ScaleOut(tc, oldSet, newSet)
	} else if *newSet.Spec.Replicas < *oldSet.Spec.Replicas {
		return fsd.ScaleIn(tc, oldSet, newSet)
	}
	return nil
}

func (fsd *fakeTiKVScaler) ScaleOut(_ *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas+1, nil)
	return nil
}

func (fsd *fakeTiKVScaler) ScaleIn(_ *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas-1, nil)
	return nil
}

func (fsd *fakeTiKVScaler) SyncAutoScalerAnn(tc *v1alpha1.TidbCluster, actual *apps.StatefulSet) error {
	return nil
}
