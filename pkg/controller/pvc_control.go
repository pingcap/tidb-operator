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

	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// TODO add unit tests

// PVCControlInterface manages PVCs used in TidbCluster
type PVCControlInterface interface {
	UpdateMetaInfo(runtime.Object, *corev1.PersistentVolumeClaim, *corev1.Pod) (*corev1.PersistentVolumeClaim, error)
	UpdatePVC(runtime.Object, *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error)
	DeletePVC(runtime.Object, *corev1.PersistentVolumeClaim) error
	GetPVC(name, namespace string) (*corev1.PersistentVolumeClaim, error)
}

type realPVCControl struct {
	kubeCli   kubernetes.Interface
	recorder  record.EventRecorder
	pvcLister corelisters.PersistentVolumeClaimLister
}

// NewRealPVCControl creates a new PVCControlInterface
func NewRealPVCControl(
	kubeCli kubernetes.Interface,
	recorder record.EventRecorder,
	pvcLister corelisters.PersistentVolumeClaimLister) PVCControlInterface {
	return &realPVCControl{
		kubeCli:   kubeCli,
		recorder:  recorder,
		pvcLister: pvcLister,
	}
}

func (c *realPVCControl) GetPVC(name, namespace string) (*corev1.PersistentVolumeClaim, error) {
	return c.pvcLister.PersistentVolumeClaims(namespace).Get(name)
}

func (c *realPVCControl) DeletePVC(controller runtime.Object, pvc *corev1.PersistentVolumeClaim) error {
	controllerMo, ok := controller.(metav1.Object)
	if !ok {
		return fmt.Errorf("%T is not a metav1.Object, cannot call setControllerReference", controller)
	}
	kind := controller.GetObjectKind().GroupVersionKind().Kind
	name := controllerMo.GetName()
	namespace := controllerMo.GetNamespace()

	pvcName := pvc.GetName()
	err := c.kubeCli.CoreV1().PersistentVolumeClaims(namespace).Delete(pvcName, nil)
	if err != nil {
		klog.Errorf("failed to delete PVC: [%s/%s], %s: %s, %v", namespace, pvcName, kind, name, err)
	}
	klog.V(4).Infof("delete PVC: [%s/%s] successfully, %s: %s", namespace, pvcName, kind, name)
	c.recordPVCEvent("delete", kind, name, controller, pvcName, err)
	return err
}

func (c *realPVCControl) UpdatePVC(controller runtime.Object, pvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error) {
	controllerMo, ok := controller.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("%T is not a metav1.Object, cannot call setControllerReference", controller)
	}
	kind := controller.GetObjectKind().GroupVersionKind().Kind
	name := controllerMo.GetName()
	namespace := controllerMo.GetNamespace()

	pvcName := pvc.GetName()

	labels := pvc.GetLabels()
	ann := pvc.GetAnnotations()
	var updatePVC *corev1.PersistentVolumeClaim
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var updateErr error
		updatePVC, updateErr = c.kubeCli.CoreV1().PersistentVolumeClaims(namespace).Update(pvc)
		if updateErr == nil {
			klog.Infof("update PVC: [%s/%s] successfully, %s: %s", namespace, pvcName, kind, name)
			return nil
		}
		klog.Errorf("failed to update PVC: [%s/%s], %s: %s, error: %v", namespace, pvcName, kind, name, updateErr)

		if updated, err := c.pvcLister.PersistentVolumeClaims(namespace).Get(pvcName); err == nil {
			// make a copy so we don't mutate the shared cache
			pvc = updated.DeepCopy()
			pvc.Labels = labels
			pvc.Annotations = ann
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated PVC %s/%s from lister: %v", namespace, pvcName, err))
		}

		return updateErr
	})
	return updatePVC, err
}

func (c *realPVCControl) UpdateMetaInfo(controller runtime.Object, pvc *corev1.PersistentVolumeClaim, pod *corev1.Pod) (*corev1.PersistentVolumeClaim, error) {
	controllerMo, ok := controller.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("%T is not a metav1.Object, cannot call setControllerReference", controller)
	}
	kind := controller.GetObjectKind().GroupVersionKind().Kind
	name := controllerMo.GetName()
	namespace := controllerMo.GetNamespace()

	pvcName := pvc.GetName()
	podName := pod.GetName()

	clusterID := pod.Labels[label.ClusterIDLabelKey]
	storeID := pod.Labels[label.StoreIDLabelKey]
	memberID := pod.Labels[label.MemberIDLabelKey]

	if pvc.Annotations == nil {
		pvc.Annotations = make(map[string]string)
	}

	if pvc.Labels == nil {
		pvc.Labels = make(map[string]string)
	}

	if pvc.Labels[label.ClusterIDLabelKey] == clusterID &&
		pvc.Labels[label.MemberIDLabelKey] == memberID &&
		pvc.Labels[label.StoreIDLabelKey] == storeID &&
		pvc.Labels[label.AnnPodNameKey] == podName &&
		pvc.Annotations[label.AnnPodNameKey] == podName {
		klog.V(4).Infof("pvc %s/%s already has labels and annotations synced, skipping, %s: %s", namespace, pvcName, kind, name)
		return pvc, nil
	}

	setIfNotEmpty(pvc.Labels, label.ClusterIDLabelKey, clusterID)
	setIfNotEmpty(pvc.Labels, label.MemberIDLabelKey, memberID)
	setIfNotEmpty(pvc.Labels, label.StoreIDLabelKey, storeID)
	setIfNotEmpty(pvc.Labels, label.AnnPodNameKey, podName)
	setIfNotEmpty(pvc.Annotations, label.AnnPodNameKey, podName)

	labels := pvc.GetLabels()
	ann := pvc.GetAnnotations()
	var updatePVC *corev1.PersistentVolumeClaim
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var updateErr error
		updatePVC, updateErr = c.kubeCli.CoreV1().PersistentVolumeClaims(namespace).Update(pvc)
		if updateErr == nil {
			klog.V(4).Infof("update PVC: [%s/%s] successfully, %s: %s", namespace, pvcName, kind, name)
			return nil
		}
		klog.Errorf("failed to update PVC: [%s/%s], %s: %s, error: %v", namespace, pvcName, kind, name, updateErr)

		if updated, err := c.pvcLister.PersistentVolumeClaims(namespace).Get(pvcName); err == nil {
			// make a copy so we don't mutate the shared cache
			pvc = updated.DeepCopy()
			pvc.Labels = labels
			pvc.Annotations = ann
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated PVC %s/%s from lister: %v", namespace, pvcName, err))
		}

		return updateErr
	})
	return updatePVC, err
}

func (c *realPVCControl) recordPVCEvent(verb, kind, name string, object runtime.Object, pvcName string, err error) {
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s PVC %s in %s %s successful",
			strings.ToLower(verb), pvcName, kind, name)
		c.recorder.Event(object, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s PVC %s in %s %s failed error: %s",
			strings.ToLower(verb), pvcName, kind, name, err)
		c.recorder.Event(object, corev1.EventTypeWarning, reason, msg)
	}
}

var _ PVCControlInterface = &realPVCControl{}

// FakePVCControl is a fake PVCControlInterface
type FakePVCControl struct {
	PVCIndexer       cache.Indexer
	updatePVCTracker RequestTracker
	deletePVCTracker RequestTracker
}

// NewFakePVCControl returns a FakePVCControl
func NewFakePVCControl(pvcInformer coreinformers.PersistentVolumeClaimInformer) *FakePVCControl {
	return &FakePVCControl{
		pvcInformer.Informer().GetIndexer(),
		RequestTracker{},
		RequestTracker{},
	}
}

// SetUpdatePVCError sets the error attributes of updatePVCTracker
func (c *FakePVCControl) SetUpdatePVCError(err error, after int) {
	c.updatePVCTracker.SetError(err).SetAfter(after)
}

// SetDeletePVCError sets the error attributes of deletePVCTracker
func (c *FakePVCControl) SetDeletePVCError(err error, after int) {
	c.deletePVCTracker.SetError(err).SetAfter(after)
}

// DeletePVC deletes the pvc
func (c *FakePVCControl) DeletePVC(_ runtime.Object, pvc *corev1.PersistentVolumeClaim) error {
	defer c.deletePVCTracker.Inc()
	if c.deletePVCTracker.ErrorReady() {
		defer c.deletePVCTracker.Reset()
		return c.deletePVCTracker.GetError()
	}

	return c.PVCIndexer.Delete(pvc)
}

// UpdatePVC updates the annotation, labels and spec of pvc
func (c *FakePVCControl) UpdatePVC(_ runtime.Object, pvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error) {
	defer c.updatePVCTracker.Inc()
	if c.updatePVCTracker.ErrorReady() {
		defer c.updatePVCTracker.Reset()
		return nil, c.updatePVCTracker.GetError()
	}

	return pvc, c.PVCIndexer.Update(pvc)
}

// UpdateMetaInfo updates the meta info of pvc
func (c *FakePVCControl) UpdateMetaInfo(_ runtime.Object, pvc *corev1.PersistentVolumeClaim, pod *corev1.Pod) (*corev1.PersistentVolumeClaim, error) {
	defer c.updatePVCTracker.Inc()
	if c.updatePVCTracker.ErrorReady() {
		defer c.updatePVCTracker.Reset()
		return nil, c.updatePVCTracker.GetError()
	}
	if pvc.Labels == nil {
		pvc.Labels = make(map[string]string)
	}
	if pvc.Annotations == nil {
		pvc.Annotations = make(map[string]string)
	}
	setIfNotEmpty(pvc.Labels, label.ClusterIDLabelKey, pod.Labels[label.ClusterIDLabelKey])
	setIfNotEmpty(pvc.Labels, label.MemberIDLabelKey, pod.Labels[label.MemberIDLabelKey])
	setIfNotEmpty(pvc.Labels, label.StoreIDLabelKey, pod.Labels[label.StoreIDLabelKey])
	setIfNotEmpty(pvc.Labels, label.AnnPodNameKey, pod.GetName())
	setIfNotEmpty(pvc.Annotations, label.AnnPodNameKey, pod.GetName())
	return nil, c.PVCIndexer.Update(pvc)
}

func (c *FakePVCControl) GetPVC(name, namespace string) (*corev1.PersistentVolumeClaim, error) {
	defer c.updatePVCTracker.Inc()
	obj, existed, err := c.PVCIndexer.GetByKey(fmt.Sprintf("%s/%s", namespace, name))
	if err != nil {
		return nil, err
	}
	if !existed {
		return nil, fmt.Errorf("pvc[%s/%s] not existed", namespace, name)
	}
	a := obj.(*corev1.PersistentVolumeClaim)
	return a, nil
}

var _ PVCControlInterface = &FakePVCControl{}
