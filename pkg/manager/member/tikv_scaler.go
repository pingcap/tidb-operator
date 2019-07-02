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
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
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

func (tsd *tikvScaler) ScaleOut(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if tc.TiKVUpgrading() {
		resetReplicas(newSet, oldSet)
		return nil
	}

	_, err := tsd.deleteDeferDeletingPVC(tc, oldSet.GetName(), v1alpha1.TiKVMemberType, *oldSet.Spec.Replicas)
	if err != nil {
		resetReplicas(newSet, oldSet)
		return err
	}

	increaseReplicas(newSet, oldSet)
	return nil
}

func (tsd *tikvScaler) ScaleIn(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	// we can only remove one member at a time when scale down
	ordinal := *oldSet.Spec.Replicas - 1
	setName := oldSet.GetName()

	// tikv can not scale in when it is upgrading
	if tc.TiKVUpgrading() {
		resetReplicas(newSet, oldSet)
		glog.Infof("the TidbCluster: [%s/%s]'s tikv is upgrading,can not scale in until upgrade have completed",
			ns, tcName)
		return nil
	}

	// We need remove member from cluster before reducing statefulset replicas
	podName := ordinalPodName(v1alpha1.TiKVMemberType, tcName, ordinal)
	pod, err := tsd.podLister.Pods(ns).Get(podName)
	if err != nil {
		resetReplicas(newSet, oldSet)
		return err
	}
	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == podName {
			state := store.State
			id, err := strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				resetReplicas(newSet, oldSet)
				return err
			}
			if state != v1alpha1.TiKVStateOffline {
				if err := controller.GetPDClient(tsd.pdControl, tc).DeleteStore(id); err != nil {
					resetReplicas(newSet, oldSet)
					return err
				}
			}
			resetReplicas(newSet, oldSet)
			return controller.RequeueErrorf("TiKV %s/%s store %d  still in cluster, state: %s", ns, podName, id, state)
		}
	}
	for id, store := range tc.Status.TiKV.TombstoneStores {
		if store.PodName == podName && pod.Labels[label.StoreIDLabelKey] == id {
			id, err := strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				resetReplicas(newSet, oldSet)
				return err
			}

			// TODO: double check if store is really not in Up/Offline/Down state
			glog.Infof("TiKV %s/%s store %d becomes tombstone", ns, podName, id)

			pvcName := ordinalPVCName(v1alpha1.TiKVMemberType, setName, ordinal)
			pvc, err := tsd.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
			if err != nil {
				resetReplicas(newSet, oldSet)
				return err
			}
			if pvc.Annotations == nil {
				pvc.Annotations = map[string]string{}
			}
			pvc.Annotations[label.AnnPVCDeferDeleting] = time.Now().Format(time.RFC3339)
			_, err = tsd.pvcControl.UpdatePVC(tc, pvc)
			if err != nil {
				resetReplicas(newSet, oldSet)
				return err
			}

			decreaseReplicas(newSet, oldSet)
			return nil
		}
	}

	// store not found in TidbCluster status,
	// this can happen when TiKV joins cluster but we haven't synced its status
	// so return error to wait another round for safety
	resetReplicas(newSet, oldSet)
	return fmt.Errorf("TiKV %s/%s not found in cluster", ns, podName)
}

type fakeTiKVScaler struct{}

// NewFakeTiKVScaler returns a fake tikv Scaler
func NewFakeTiKVScaler() Scaler {
	return &fakeTiKVScaler{}
}

func (fsd *fakeTiKVScaler) ScaleOut(_ *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	increaseReplicas(newSet, oldSet)
	return nil
}

func (fsd *fakeTiKVScaler) ScaleIn(_ *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	decreaseReplicas(newSet, oldSet)
	return nil
}
