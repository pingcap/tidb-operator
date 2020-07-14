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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

type TiKVScaler interface {
	Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error
	ScaleOut(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error
	ScaleIn(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error
	SyncAutoScalerAnn(meta metav1.Object, actual *apps.StatefulSet) error
}

type tikvScaler struct {
	generalScaler
	podLister corelisters.PodLister
}

// NewTiKVScaler returns a tikv Scaler
func NewTiKVScaler(pdControl pdapi.PDControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvcControl controller.PVCControlInterface,
	podLister corelisters.PodLister) *tikvScaler {
	return &tikvScaler{generalScaler{pdControl, pvcLister, pvcControl}, podLister}
}

func (tsd *tikvScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return tsd.ScaleOut(meta, oldSet, newSet)
	} else if scaling < 0 {
		return tsd.ScaleIn(meta, oldSet, newSet)
	}
	// we only sync auto scaler annotations when we are finishing syncing scaling
	return tsd.SyncAutoScalerAnn(meta, oldSet)
}

func (tsd *tikvScaler) ScaleOut(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	obj, ok := meta.(runtime.Object)
	if !ok {
		return fmt.Errorf("cluster[%s/%s] can't conver to runtime.Object", meta.GetNamespace(), meta.GetName())
	}
	klog.Infof("scaling out tikv statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	_, err := tsd.deleteDeferDeletingPVC(obj, oldSet.GetName(), v1alpha1.TiKVMemberType, ordinal)
	if err != nil {
		return err
	}

	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

func (tsd *tikvScaler) ScaleIn(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := meta.GetNamespace()
	tcName := meta.GetName()
	// we can only remove one member at a time when scaling in
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	setName := oldSet.GetName()

	klog.Infof("scaling in tikv statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	// We need remove member from cluster before reducing statefulset replicas
	podName := ordinalPodName(v1alpha1.TiKVMemberType, tcName, ordinal)
	pod, err := tsd.podLister.Pods(ns).Get(podName)
	if err != nil {
		return fmt.Errorf("tikvScaler.ScaleIn: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
	}

	if controller.PodWebhookEnabled {
		setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
		return nil
	}

	tc, ok := meta.(*v1alpha1.TidbCluster)
	if !ok {
		return fmt.Errorf("cluster tikv [%s/%s] scaling should enabled webhook", meta.GetNamespace(), meta.GetName())
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == podName {
			state := store.State
			id, err := strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				return err
			}
			if state != v1alpha1.TiKVStateOffline {
				if err := controller.GetPDClient(tsd.pdControl, tc).DeleteStore(id); err != nil {
					klog.Errorf("tikv scale in: failed to delete store %d, %v", id, err)
					return err
				}
				klog.Infof("tikv scale in: delete store %d for tikv %s/%s successfully", id, ns, podName)
			}
			return controller.RequeueErrorf("TiKV %s/%s store %d is still in cluster, state: %s", ns, podName, id, state)
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
				return fmt.Errorf("tikvScaler.ScaleIn: failed to get pvc %s for cluster %s/%s, error: %s", pvcName, ns, tcName, err)
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
			return fmt.Errorf("tikvScaler.ScaleIn: failed to get pvc %s for cluster %s/%s, error: %s", pvcName, ns, tcName, err)
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
		setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
		return nil
	}
	return fmt.Errorf("TiKV %s/%s not found in cluster", ns, podName)
}

// SyncAutoScalerAnn would reclaim the auto-scaling out slots if the target pod is no longer existed
func (tsd *tikvScaler) SyncAutoScalerAnn(meta metav1.Object, actual *apps.StatefulSet) error {
	tc, ok := meta.(*v1alpha1.TidbCluster)
	if !ok {
		return nil
	}
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
func NewFakeTiKVScaler() TiKVScaler {
	return &fakeTiKVScaler{}
}

func (fsd *fakeTiKVScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if *newSet.Spec.Replicas > *oldSet.Spec.Replicas {
		return fsd.ScaleOut(meta, oldSet, newSet)
	} else if *newSet.Spec.Replicas < *oldSet.Spec.Replicas {
		return fsd.ScaleIn(meta, oldSet, newSet)
	}
	return nil
}

func (fsd *fakeTiKVScaler) ScaleOut(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas+1, nil)
	return nil
}

func (fsd *fakeTiKVScaler) ScaleIn(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas-1, nil)
	return nil
}

func (fsd *fakeTiKVScaler) SyncAutoScalerAnn(_ metav1.Object, actual *apps.StatefulSet) error {
	return nil
}
