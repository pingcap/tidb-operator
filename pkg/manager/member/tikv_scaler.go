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

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

type tikvScaler struct {
	generalScaler
}

// NewTiKVScaler returns a tikv Scaler
func NewTiKVScaler(deps *controller.Dependencies) *tikvScaler {
	return &tikvScaler{generalScaler: generalScaler{deps: deps}}
}

func (s *tikvScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return s.ScaleOut(meta, oldSet, newSet)
	} else if scaling < 0 {
		return s.ScaleIn(meta, oldSet, newSet)
	}
	// we only sync auto scaler annotations when we are finishing syncing scaling
	return nil
}

func (s *tikvScaler) ScaleOut(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	obj, ok := meta.(runtime.Object)
	if !ok {
		return fmt.Errorf("cluster[%s/%s] can't conver to runtime.Object", meta.GetNamespace(), meta.GetName())
	}
	klog.Infof("scaling out tikv statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	var pvcName string
	switch meta.(type) {
	case *v1alpha1.TidbCluster:
		pvcName = fmt.Sprintf("tikv-%s-tikv-%d", meta.GetName(), ordinal)
	default:
		return fmt.Errorf("tikv.ScaleOut, failed to convert cluster %s/%s", meta.GetNamespace(), meta.GetName())
	}
	_, err := s.deps.PVCLister.PersistentVolumeClaims(meta.GetNamespace()).Get(pvcName)
	if err == nil {
		_, err = s.deleteDeferDeletingPVC(obj, v1alpha1.TiKVMemberType, ordinal)
		if err != nil {
			return err
		}
		return controller.RequeueErrorf("tikv.ScaleOut, cluster %s/%s ready to scale out, wait for next round", meta.GetNamespace(), meta.GetName())
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("tikv.ScaleOut, cluster %s/%s failed to fetch pvc informaiton, err:%v", meta.GetNamespace(), meta.GetName(), err)
	}
	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

func (s *tikvScaler) ScaleIn(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := meta.GetNamespace()
	tcName := meta.GetName()
	// we can only remove one member at a time when scaling in
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)

	klog.Infof("scaling in tikv statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	// We need remove member from cluster before reducing statefulset replicas
	var podName string

	switch meta.(type) {
	case *v1alpha1.TidbCluster:
		podName = ordinalPodName(v1alpha1.TiKVMemberType, tcName, ordinal)
	default:
		return fmt.Errorf("tikvScaler.ScaleIn: failed to convert cluster %s/%s", meta.GetNamespace(), meta.GetName())
	}
	pod, err := s.deps.PodLister.Pods(ns).Get(podName)
	if err != nil {
		return fmt.Errorf("tikvScaler.ScaleIn: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
	}

	tc, _ := meta.(*v1alpha1.TidbCluster)

	if pass, err := s.preCheckUpStores(tc, podName); !pass {
		return err
	}

	if s.deps.CLIConfig.PodWebhookEnabled {
		setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
		return nil
	}

	// call PD API to delete the store of the TiKV Pod to be scaled in
	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == podName {
			state := store.State
			id, err := strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				return err
			}
			if state != v1alpha1.TiKVStateOffline {
				if err := controller.GetPDClient(s.deps.PDControl, tc).DeleteStore(id); err != nil {
					klog.Errorf("tikvScaler.ScaleIn: failed to delete store %d, %v", id, err)
					return err
				}
				klog.Infof("tikvScaler.ScaleIn: delete store %d for tikv %s/%s successfully", id, ns, podName)
			}
			return controller.RequeueErrorf("TiKV %s/%s store %d is still in cluster, state: %s", ns, podName, id, state)
		}
	}

	// If the store state turns to Tombstone, add defer deleting annotation to the PVCs of the Pod
	for storeID, store := range tc.Status.TiKV.TombstoneStores {
		if store.PodName == podName && pod.Labels[label.StoreIDLabelKey] == storeID {
			id, err := strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				return err
			}

			// TODO: double check if store is really not in Up/Offline/Down state
			klog.Infof("TiKV %s/%s store %d becomes tombstone", ns, podName, id)

			pvcs, err := util.ResolvePVCFromPod(pod, s.deps.PVCLister)
			if err != nil {
				return fmt.Errorf("tikvScaler.ScaleIn: failed to get pvcs for pod %s/%s in tc %s/%s, error: %s", ns, pod.Name, ns, tcName, err)
			}
			for _, pvc := range pvcs {
				if err := addDeferDeletingAnnoToPVC(tc, pvc, s.deps.PVCControl); err != nil {
					return err
				}
			}

			// endEvictLeader for TombStone stores
			if err = endEvictLeaderbyStoreID(s.deps, tc, id); err != nil {
				return err
			}

			setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
			return nil
		}
	}

	// When store not found in TidbCluster status, there are two possible situations:
	// 1. TiKV has already joined cluster but status not synced yet.
	//    In this situation return error to wait for another round for safety.
	// 2. TiKV pod is not ready, such as in pending state.
	//    In this situation we should delete this TiKV pod immediately to avoid blocking the subsequent operations.
	if !podutil.IsPodReady(pod) {
		if tc.TiKVBootStrapped() {
			safeTimeDeadline := pod.CreationTimestamp.Add(5 * s.deps.CLIConfig.ResyncDuration)
			if time.Now().Before(safeTimeDeadline) {
				// Wait for 5 resync periods to ensure that the following situation does not occur:
				//
				// The tikv pod starts for a while, but has not synced its status, and then the pod becomes not ready.
				// Here we wait for 5 resync periods to ensure that the status of this tikv pod has been synced.
				// After this period of time, if there is still no information about this tikv in TidbCluster status,
				// then we can be sure that this tikv has never been added to the tidb cluster.
				// So we can scale in this tikv pod safely.
				resetReplicas(newSet, oldSet)
				return fmt.Errorf("TiKV %s/%s is not ready, wait for 5 resync periods to sync its status", ns, podName)
			}
		}

		pvcs, err := util.ResolvePVCFromPod(pod, s.deps.PVCLister)
		if err != nil {
			return fmt.Errorf("tikvScaler.ScaleIn: failed to get pvcs for pod %s/%s in tc %s/%s, error: %s", ns, pod.Name, ns, tcName, err)
		}
		for _, pvc := range pvcs {
			if err := addDeferDeletingAnnoToPVC(tc, pvc, s.deps.PVCControl); err != nil {
				return err
			}
		}

		setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
		return nil
	}
	return fmt.Errorf("TiKV %s/%s not found in cluster", ns, podName)
}

func (s *tikvScaler) preCheckUpStores(tc *v1alpha1.TidbCluster, podName string) (bool, error) {
	if !tc.TiKVBootStrapped() {
		klog.Infof("TiKV of Cluster %s/%s is not bootstrapped yet, skip pre check when scale in TiKV", tc.Namespace, tc.Name)
		return true, nil
	}

	pdClient := controller.GetPDClient(s.deps.PDControl, tc)
	// get the number of stores whose state is up
	upNumber := 0

	storesInfo, err := pdClient.GetStores()
	if err != nil {
		return false, fmt.Errorf("failed to get stores info in TidbCluster %s/%s", tc.GetNamespace(), tc.GetName())
	}
	// filter out TiFlash
	for _, store := range storesInfo.Stores {
		if store.Store != nil {
			if store.Store.StateName == v1alpha1.TiKVStateUp && util.MatchLabelFromStoreLabels(store.Store.Labels, label.TiKVLabelVal) {
				upNumber++
			}
		}
	}

	// get the state of the store which is about to be scaled in
	storeState := ""
	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == podName {
			storeState = store.State
		}
	}

	config, err := pdClient.GetConfig()
	if err != nil {
		return false, err
	}
	maxReplicas := *(config.Replication.MaxReplicas)
	if upNumber < int(maxReplicas) {
		errMsg := fmt.Sprintf("the number of stores in Up state of TidbCluster [%s/%s] is %d, less than MaxReplicas in PD configuration(%d), can't scale in TiKV, podname %s ", tc.GetNamespace(), tc.GetName(), upNumber, maxReplicas, podName)
		klog.Error(errMsg)
		s.deps.Recorder.Event(tc, v1.EventTypeWarning, "FailedScaleIn", errMsg)
		return false, nil
	} else if upNumber == int(maxReplicas) {
		if storeState == v1alpha1.TiKVStateUp {
			errMsg := fmt.Sprintf("can't scale in TiKV of TidbCluster [%s/%s], cause the number of up stores is equal to MaxReplicas in PD configuration(%d), and the store in Pod %s which is going to be deleted is up too", tc.GetNamespace(), tc.GetName(), maxReplicas, podName)
			klog.Error(errMsg)
			s.deps.Recorder.Event(tc, v1.EventTypeWarning, "FailedScaleIn", errMsg)
			return false, nil
		}
	}

	return true, nil
}

type fakeTiKVScaler struct{}

// NewFakeTiKVScaler returns a fake tikv Scaler
func NewFakeTiKVScaler() Scaler {
	return &fakeTiKVScaler{}
}

func (s *fakeTiKVScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if *newSet.Spec.Replicas > *oldSet.Spec.Replicas {
		return s.ScaleOut(meta, oldSet, newSet)
	} else if *newSet.Spec.Replicas < *oldSet.Spec.Replicas {
		return s.ScaleIn(meta, oldSet, newSet)
	}
	return nil
}

func (s *fakeTiKVScaler) ScaleOut(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas+1, nil)
	return nil
}

func (s *fakeTiKVScaler) ScaleIn(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas-1, nil)
	return nil
}
