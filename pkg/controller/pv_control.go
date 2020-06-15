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
	"k8s.io/apimachinery/pkg/runtime"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// PVControlInterface manages PVs used in TidbCluster
type PVControlInterface interface {
	PatchPVReclaimPolicy(runtime.Object, *corev1.PersistentVolume, corev1.PersistentVolumeReclaimPolicy) error
	UpdateMetaInfo(runtime.Object, *corev1.PersistentVolume) (*corev1.PersistentVolume, error)
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

func (rpc *realPVControl) PatchPVReclaimPolicy(obj runtime.Object, pv *corev1.PersistentVolume, reclaimPolicy corev1.PersistentVolumeReclaimPolicy) error {
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return fmt.Errorf("%+v is not a runtime.Object, cannot get controller from it", obj)
	}

	name := metaObj.GetName()
	pvName := pv.GetName()

	latestPV, err := rpc.kubeCli.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if string(latestPV.Spec.PersistentVolumeReclaimPolicy) == string(reclaimPolicy) {
		return nil
	}
	patchBytes := []byte(fmt.Sprintf(`{"spec":{"persistentVolumeReclaimPolicy":"%s"}}`, reclaimPolicy))
	
	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, err := rpc.kubeCli.CoreV1().PersistentVolumes().Patch(pvName, types.StrategicMergePatchType, patchBytes)
		return err
	})
	rpc.recordPVEvent("patch", obj, name, pvName, err)
	return err
}

func (rpc *realPVControl) UpdateMetaInfo(obj runtime.Object, pv *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("%+v is not a runtime.Object, cannot get controller from it", obj)
	}

	ns := metaObj.GetNamespace()
	name := metaObj.GetName()
	kind := obj.GetObjectKind().GroupVersionKind().Kind

	if pv.Annotations == nil {
		pv.Annotations = make(map[string]string)
	}
	if pv.Labels == nil {
		pv.Labels = make(map[string]string)
	}
	pvName := pv.GetName()
	pvcRef := pv.Spec.ClaimRef
	if pvcRef == nil {
		klog.Warningf("PV: [%s] doesn't have a ClaimRef, skipping, %s: %s/%s", kind, pvName, ns, name)
		return pv, nil
	}

	pvcName := pvcRef.Name
	pvc, err := rpc.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil {
		if !apierrs.IsNotFound(err) {
			return pv, err
		}

		klog.Warningf("PV: [%s]'s PVC: [%s/%s] doesn't exist, skipping. %s: %s", pvName, ns, pvcName, kind, name)
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
		klog.V(4).Infof("pv %s already has labels and annotations synced, skipping. %s: %s/%s", pvName, kind, ns, name)
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
			klog.Infof("PV: [%s] updated successfully, %s: %s/%s", pvName, kind, ns, name)
			return nil
		}
		klog.Errorf("failed to update PV: [%s], %s %s/%s, error: %v", pvName, kind, ns, name, err)

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

	return updatePV, err
}

func (rpc *realPVControl) recordPVEvent(verb string, obj runtime.Object, objName, pvName string, err error) {
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s PV %s in TidbCluster %s successful",
			strings.ToLower(verb), pvName, objName)
		rpc.recorder.Event(obj, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s PV %s in TidbCluster %s failed error: %s",
			strings.ToLower(verb), pvName, objName, err)
		rpc.recorder.Event(obj, corev1.EventTypeWarning, reason, msg)
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
func (fpc *FakePVControl) PatchPVReclaimPolicy(_ runtime.Object, pv *corev1.PersistentVolume, reclaimPolicy corev1.PersistentVolumeReclaimPolicy) error {
	defer fpc.updatePVTracker.Inc()
	if fpc.updatePVTracker.ErrorReady() {
		defer fpc.updatePVTracker.Reset()
		return fpc.updatePVTracker.GetError()
	}
	pv.Spec.PersistentVolumeReclaimPolicy = reclaimPolicy

	return fpc.PVIndexer.Update(pv)
}

// UpdateMetaInfo update the meta info of pv
func (fpc *FakePVControl) UpdateMetaInfo(obj runtime.Object, pv *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
	defer fpc.updatePVTracker.Inc()

	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("%+v is not a runtime.Object, cannot get controller from it", obj)
	}
	ns := metaObj.GetNamespace()
	if fpc.updatePVTracker.ErrorReady() {
		defer fpc.updatePVTracker.Reset()
		return nil, fpc.updatePVTracker.GetError()
	}
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
