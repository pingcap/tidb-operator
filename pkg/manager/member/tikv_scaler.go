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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type tikvScaler struct {
	pdControl  controller.PDControlInterface
	pvcLister  corelisters.PersistentVolumeClaimLister
	pvcControl controller.PVCControlInterface
}

// NewTiKVScaler returns a tikv Scaler
func NewTiKVScaler(pdControl controller.PDControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvcControl controller.PVCControlInterface) Scaler {
	return &tikvScaler{pdControl, pvcLister, pvcControl}
}

func (tsd *tikvScaler) ScaleOut(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	ordinal := *oldSet.Spec.Replicas + 1
	setName := oldSet.GetName()

	if tc.TiKVUpgrading() {
		resetReplicas(newSet, oldSet)
		return nil
	}

	pvcName := fmt.Sprintf("%s-%s-%d", v1alpha1.TiKVMemberType, setName, ordinal)
	pvc, err := tsd.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
	if errors.IsNotFound(err) {
		increaseReplicas(newSet, oldSet)
		return nil
	}
	if err != nil {
		resetReplicas(newSet, oldSet)
		return err
	}
	if pvc.Annotations == nil {
		increaseReplicas(newSet, oldSet)
		return nil
	}
	if _, ok := pvc.Annotations[label.AnnPVCDeferDeleting]; !ok {
		increaseReplicas(newSet, oldSet)
		return nil
	}
	err = tsd.pvcControl.DeletePVC(tc, pvc)
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
	podName := fmt.Sprintf("%s-%s-%d", tc.Name, v1alpha1.TiKVMemberType, ordinal)
	for _, store := range tc.Status.TiKV.Stores.CurrentStores {
		if store.PodName == podName {
			state := store.State
			id, err := strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				resetReplicas(newSet, oldSet)
				return err
			}
			if state != util.StoreOfflineState {
				if err := tsd.pdControl.GetPDClient(tc).DeleteStore(id); err != nil {
					resetReplicas(newSet, oldSet)
					return err
				}
			}
			resetReplicas(newSet, oldSet)
			return fmt.Errorf("TiKV %s/%s store %d  still in cluster, state: %s", ns, podName, id, state)
		}
	}
	for _, store := range tc.Status.TiKV.Stores.TombStoneStores {
		if store.PodName == podName {
			id, err := strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				resetReplicas(newSet, oldSet)
				return err
			}

			// TODO: double check if store is really not in Up/Offline/Down state

			pvcName := fmt.Sprintf("%s-%s-%d", v1alpha1.TiKVMemberType, setName, ordinal)
			pvc, err := tsd.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
			if err != nil {
				resetReplicas(newSet, oldSet)
				return err
			}
			if pvc.Annotations == nil {
				pvc.Annotations = map[string]string{}
			}
			pvc.Annotations[label.AnnPVCDeferDeleting] = time.Now().Format(time.RFC3339)
			err = tsd.pvcControl.UpdatePVC(tc, pvc)
			if err != nil {
				resetReplicas(newSet, oldSet)
				return err
			}

			decreaseReplicas(newSet, oldSet)
			glog.Infof("TiKV %s/%s store %d becomes tombstone", ns, podName, id)
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

func (fsd *fakeTiKVScaler) ScaleOut(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	return nil
}

func (fsd *fakeTiKVScaler) ScaleIn(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	return nil
}
