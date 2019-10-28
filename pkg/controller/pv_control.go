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

package controller

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
)

// PVControlInterface manages PVs used in TidbCluster
type PVControlInterface interface {
	PatchPVReclaimPolicy(*v1alpha1.TidbCluster, *corev1.PersistentVolume, corev1.PersistentVolumeReclaimPolicy) error
	UpdateMetaInfo(*v1alpha1.TidbCluster, *corev1.PersistentVolume) (*corev1.PersistentVolume, error)
}

type realPVControl struct {
	kubeCli   kubernetes.Interface
	pvcLister corelisters.PersistentVolumeClaimLister
	pvLister  corelisters.PersistentVolumeLister
	recorder  record.EventRecorder
}

// NewRealPVControl creates a new PVControlInterface
func NewRealPVControl(
	kubeCli kubernetes.Interface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvLister corelisters.PersistentVolumeLister,
	recorder record.EventRecorder,
) PVControlInterface {
	return &realPVControl{
		kubeCli:   kubeCli,
		pvcLister: pvcLister,
		pvLister:  pvLister,
		recorder:  recorder,
	}
}

func (rpc *realPVControl) PatchPVReclaimPolicy(tc *v1alpha1.TidbCluster, pv *corev1.PersistentVolume, reclaimPolicy corev1.PersistentVolumeReclaimPolicy) error {
	pvName := pv.GetName()
	patchBytes := []byte(fmt.Sprintf(`{"spec":{"persistentVolumeReclaimPolicy":"%s"}}`, reclaimPolicy))

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, err := rpc.kubeCli.CoreV1().PersistentVolumes().Patch(pvName, types.StrategicMergePatchType, patchBytes)
		return err
	})
	rpc.recordPVEvent("patch", tc, pvName, err)
	return err
}

func (rpc *realPVControl) UpdateMetaInfo(tc *v1alpha1.TidbCluster, pv *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if pv.Annotations == nil {
		pv.Annotations = make(map[string]string)
	}
	if pv.Labels == nil {
		pv.Labels = make(map[string]string)
	}
	pvName := pv.GetName()
	pvcRef := pv.Spec.ClaimRef
	if pvcRef == nil {
		glog.Warningf("PV: [%s] doesn't have a ClaimRef, skipping, TidbCluster: %s/%s", pvName, ns, tcName)
		return pv, nil
	}

	pvcName := pvcRef.Name
	pvc, err := rpc.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil {
		if !apierrs.IsNotFound(err) {
			return pv, err
		}

		glog.Warningf("PV: [%s]'s PVC: [%s/%s] doesn't exist, skipping. TidbCluster: %s", pvName, ns, pvcName, tcName)
		return pv, nil
	}

	component := pvc.Labels[label.ComponentLabelKey]
	podName := pvc.Annotations[label.AnnPodNameKey]
	clusterID := pvc.Labels[label.ClusterIDLabelKey]
	memberID := pvc.Labels[label.MemberIDLabelKey]
	storeID := pvc.Labels[label.StoreIDLabelKey]

	if pv.Labels[label.NamespaceLabelKey] == ns &&
		pv.Labels[label.ComponentLabelKey] == component &&
		pv.Labels[label.NameLabelKey] == pvc.Labels[label.NameLabelKey] &&
		pv.Labels[label.ManagedByLabelKey] == pvc.Labels[label.ManagedByLabelKey] &&
		pv.Labels[label.InstanceLabelKey] == pvc.Labels[label.InstanceLabelKey] &&
		pv.Labels[label.ClusterIDLabelKey] == clusterID &&
		pv.Labels[label.MemberIDLabelKey] == memberID &&
		pv.Labels[label.StoreIDLabelKey] == storeID &&
		pv.Annotations[label.AnnPodNameKey] == podName {
		glog.V(4).Infof("pv %s already has labels and annotations synced, skipping. TidbCluster: %s/%s", pvName, ns, tcName)
		return pv, nil
	}

	pv.Labels[label.NamespaceLabelKey] = ns
	pv.Labels[label.ComponentLabelKey] = component
	pv.Labels[label.NameLabelKey] = pvc.Labels[label.NameLabelKey]
	pv.Labels[label.ManagedByLabelKey] = pvc.Labels[label.ManagedByLabelKey]
	pv.Labels[label.InstanceLabelKey] = pvc.Labels[label.InstanceLabelKey]

	setIfNotEmpty(pv.Labels, label.ClusterIDLabelKey, clusterID)
	setIfNotEmpty(pv.Labels, label.MemberIDLabelKey, memberID)
	setIfNotEmpty(pv.Labels, label.StoreIDLabelKey, storeID)
	setIfNotEmpty(pv.Annotations, label.AnnPodNameKey, podName)

	labels := pv.GetLabels()
	ann := pv.GetAnnotations()
	var updatePV *corev1.PersistentVolume
	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var updateErr error
		updatePV, updateErr = rpc.kubeCli.CoreV1().PersistentVolumes().Update(pv)
		if updateErr == nil {
			glog.Infof("PV: [%s] updated successfully, TidbCluster: %s/%s", pvName, ns, tcName)
			return nil
		}
		glog.Errorf("failed to update PV: [%s], TidbCluster %s/%s, error: %v", pvName, ns, tcName, err)

		if updated, err := rpc.pvLister.Get(pvName); err == nil {
			// make a copy so we don't mutate the shared cache
			pv = updated.DeepCopy()
			pv.Labels = labels
			pv.Annotations = ann
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated PV %s/%s from lister: %v", ns, pvName, err))
		}
		return updateErr
	})

	rpc.recordPVEvent("update", tc, pvName, err)
	return updatePV, err
}

func (rpc *realPVControl) recordPVEvent(verb string, tc *v1alpha1.TidbCluster, pvName string, err error) {
	tcName := tc.GetName()
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s PV %s in TidbCluster %s successful",
			strings.ToLower(verb), pvName, tcName)
		rpc.recorder.Event(tc, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s PV %s in TidbCluster %s failed error: %s",
			strings.ToLower(verb), pvName, tcName, err)
		rpc.recorder.Event(tc, corev1.EventTypeWarning, reason, msg)
	}
}

var _ PVControlInterface = &realPVControl{}

// FakePVControl is a fake PVControlInterface
type FakePVControl struct {
	PVCLister       corelisters.PersistentVolumeClaimLister
	PVIndexer       cache.Indexer
	updatePVTracker RequestTracker
}

// NewFakePVControl returns a FakePVControl
func NewFakePVControl(pvInformer coreinformers.PersistentVolumeInformer, pvcInformer coreinformers.PersistentVolumeClaimInformer) *FakePVControl {
	return &FakePVControl{
		pvcInformer.Lister(),
		pvInformer.Informer().GetIndexer(),
		RequestTracker{},
	}
}

// SetUpdatePVError sets the error attributes of updatePVTracker
func (fpc *FakePVControl) SetUpdatePVError(err error, after int) {
	fpc.updatePVTracker.SetError(err).SetAfter(after)
}

// PatchPVReclaimPolicy patchs the reclaim policy of PV
func (fpc *FakePVControl) PatchPVReclaimPolicy(_ *v1alpha1.TidbCluster, pv *corev1.PersistentVolume, reclaimPolicy corev1.PersistentVolumeReclaimPolicy) error {
	defer fpc.updatePVTracker.Inc()
	if fpc.updatePVTracker.ErrorReady() {
		defer fpc.updatePVTracker.Reset()
		return fpc.updatePVTracker.err
	}
	pv.Spec.PersistentVolumeReclaimPolicy = reclaimPolicy

	return fpc.PVIndexer.Update(pv)
}

// UpdateMetaInfo update the meta info of pv
func (fpc *FakePVControl) UpdateMetaInfo(tc *v1alpha1.TidbCluster, pv *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
	defer fpc.updatePVTracker.Inc()
	if fpc.updatePVTracker.ErrorReady() {
		defer fpc.updatePVTracker.Reset()
		return nil, fpc.updatePVTracker.err
	}
	ns := tc.GetNamespace()
	pvcName := pv.Spec.ClaimRef.Name
	pvc, err := fpc.PVCLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil {
		return nil, err
	}
	if pv.Labels == nil {
		pv.Labels = make(map[string]string)
	}
	if pv.Annotations == nil {
		pv.Annotations = make(map[string]string)
	}
	pv.Labels[label.NamespaceLabelKey] = ns
	pv.Labels[label.NameLabelKey] = pvc.Labels[label.NameLabelKey]
	pv.Labels[label.ManagedByLabelKey] = pvc.Labels[label.ManagedByLabelKey]
	pv.Labels[label.InstanceLabelKey] = pvc.Labels[label.InstanceLabelKey]
	pv.Labels[label.ComponentLabelKey] = pvc.Labels[label.ComponentLabelKey]

	setIfNotEmpty(pv.Labels, label.ClusterIDLabelKey, pvc.Labels[label.ClusterIDLabelKey])
	setIfNotEmpty(pv.Labels, label.MemberIDLabelKey, pvc.Labels[label.MemberIDLabelKey])
	setIfNotEmpty(pv.Labels, label.StoreIDLabelKey, pvc.Labels[label.StoreIDLabelKey])
	setIfNotEmpty(pv.Annotations, label.AnnPodNameKey, pvc.Annotations[label.AnnPodNameKey])
	return pv, fpc.PVIndexer.Update(pv)
}

var _ PVControlInterface = &FakePVControl{}
