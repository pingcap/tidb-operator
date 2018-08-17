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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
)

// TODO add unit tests

// PVCControlInterface manages PVCs used in TidbCluster
type PVCControlInterface interface {
	UpdateMetaInfo(*v1alpha1.TidbCluster, *corev1.PersistentVolumeClaim, *corev1.Pod) error
	UpdatePVC(*v1alpha1.TidbCluster, *corev1.PersistentVolumeClaim) error
	DeletePVC(*v1alpha1.TidbCluster, *corev1.PersistentVolumeClaim) error
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

func (rpc *realPVCControl) DeletePVC(tc *v1alpha1.TidbCluster, pvc *corev1.PersistentVolumeClaim) error {
	pvcName := pvc.GetName()
	err := rpc.kubeCli.CoreV1().PersistentVolumeClaims(tc.GetNamespace()).Delete(pvcName, nil)
	rpc.recordPVCEvent("delete", tc, pvcName, err)
	return err
}

func (rpc *realPVCControl) UpdatePVC(tc *v1alpha1.TidbCluster, pvc *corev1.PersistentVolumeClaim) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	pvcName := pvc.GetName()
	pvcDup := pvc.DeepCopy()

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, updateErr := rpc.kubeCli.CoreV1().PersistentVolumeClaims(ns).Update(pvc)
		if updateErr == nil {
			glog.V(4).Infof("update PVC: [%s/%s] successfully, TidbCluster: %s", ns, pvcName, tcName)
			return nil
		}
		glog.Errorf("failed to update PVC: [%s/%s], TidbCluster: %s, error: %v", ns, pvcName, tcName, updateErr)

		if updated, err := rpc.pvcLister.PersistentVolumeClaims(ns).Get(pvcName); err == nil {
			// make a copy so we don't mutate the shared cache
			pvc = updated.DeepCopy()
			pvc.Annotations = pvcDup.Annotations
			pvc.Labels = pvcDup.Labels
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated PVC %s/%s from lister: %v", ns, pvcName, err))
		}

		return updateErr
	})
	rpc.recordPVCEvent("update", tc, pvcName, err)
	return err
}

func (rpc *realPVCControl) UpdateMetaInfo(tc *v1alpha1.TidbCluster, pvc *corev1.PersistentVolumeClaim, pod *corev1.Pod) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	pvcName := pvc.GetName()
	podName := pod.GetName()

	app := pod.Labels[label.AppLabelKey]
	clusterID := pod.Labels[label.ClusterIDLabelKey]
	storeID := pod.Labels[label.StoreIDLabelKey]
	memberID := pod.Labels[label.MemberIDLabelKey]
	owner := pod.Labels[label.OwnerLabelKey]

	if pvc.Annotations == nil {
		pvc.Annotations = make(map[string]string)
	}

	if pvc.Labels == nil {
		pvc.Labels = make(map[string]string)
	}

	if pvc.Labels[label.AppLabelKey] == app &&
		pvc.Labels[label.OwnerLabelKey] == owner &&
		pvc.Labels[label.ClusterLabelKey] == tcName &&
		pvc.Labels[label.ClusterIDLabelKey] == clusterID &&
		pvc.Labels[label.MemberIDLabelKey] == memberID &&
		pvc.Labels[label.StoreIDLabelKey] == storeID &&
		pvc.Annotations[label.AnnPodNameKey] == podName {
		glog.V(4).Infof("pvc %s/%s already has labels and annotations synced, skipping, TidbCluster: %s", ns, pvcName, tcName)
		return nil
	}

	setIfNotEmpty(pvc.Labels, label.AppLabelKey, app)
	setIfNotEmpty(pvc.Labels, label.OwnerLabelKey, owner)
	setIfNotEmpty(pvc.Labels, label.ClusterLabelKey, tcName)
	setIfNotEmpty(pvc.Labels, label.ClusterIDLabelKey, clusterID)
	setIfNotEmpty(pvc.Labels, label.MemberIDLabelKey, memberID)
	setIfNotEmpty(pvc.Labels, label.StoreIDLabelKey, storeID)
	setIfNotEmpty(pvc.Annotations, label.AnnPodNameKey, podName)

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, err := rpc.kubeCli.CoreV1().PersistentVolumeClaims(ns).Update(pvc)
		if err != nil {
			glog.Errorf("failed to update PVC: [%s/%s], TidbCluster: %s, error: %v", ns, pvcName, tcName, err)
			return err
		}
		glog.V(4).Infof("update PVC: [%s/%s] successfully, TidbCluster: %s", ns, pvcName, tcName)
		return nil
	})
	rpc.recordPVCEvent("update", tc, pvcName, err)
	return err
}

func (rpc *realPVCControl) recordPVCEvent(verb string, tc *v1alpha1.TidbCluster, pvcName string, err error) {
	tcName := tc.GetName()
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s PVC %s in TidbCluster %s successful",
			strings.ToLower(verb), pvcName, tcName)
		rpc.recorder.Event(tc, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s PVC %s in TidbCluster %s failed error: %s",
			strings.ToLower(verb), pvcName, tcName, err)
		rpc.recorder.Event(tc, corev1.EventTypeWarning, reason, msg)
	}
}

var _ PVCControlInterface = &realPVCControl{}

// FakePVCControl is a fake PVCControlInterface
type FakePVCControl struct {
	PVCIndexer       cache.Indexer
	updatePVCTracker requestTracker
	deletePVCTracker requestTracker
}

// NewFakePVCControl returns a FakePVCControl
func NewFakePVCControl(pvcInformer coreinformers.PersistentVolumeClaimInformer) *FakePVCControl {
	return &FakePVCControl{
		pvcInformer.Informer().GetIndexer(),
		requestTracker{0, nil, 0},
		requestTracker{0, nil, 0},
	}
}

// SetUpdatePVCError sets the error attributes of updatePVCTracker
func (fpc *FakePVCControl) SetUpdatePVCError(err error, after int) {
	fpc.updatePVCTracker.err = err
	fpc.updatePVCTracker.after = after
}

// SetDeletePVCError sets the error attributes of deletePVCTracker
func (fpc *FakePVCControl) SetDeletePVCError(err error, after int) {
	fpc.deletePVCTracker.err = err
	fpc.deletePVCTracker.after = after
}

// DeletePVC deletes the pvc
func (fpc *FakePVCControl) DeletePVC(tc *v1alpha1.TidbCluster, pvc *corev1.PersistentVolumeClaim) error {
	defer fpc.deletePVCTracker.inc()
	if fpc.deletePVCTracker.errorReady() {
		defer fpc.deletePVCTracker.reset()
		return fpc.deletePVCTracker.err
	}

	return fpc.PVCIndexer.Delete(pvc)
}

// Update updates the annotation, labels and spec of pvc
func (fpc *FakePVCControl) UpdatePVC(tc *v1alpha1.TidbCluster, pvc *corev1.PersistentVolumeClaim) error {
	defer fpc.updatePVCTracker.inc()
	if fpc.updatePVCTracker.errorReady() {
		defer fpc.updatePVCTracker.reset()
		return fpc.updatePVCTracker.err
	}

	return fpc.PVCIndexer.Update(pvc)
}

// UpdateMetaInfo updates the meta info of pvc
func (fpc *FakePVCControl) UpdateMetaInfo(tc *v1alpha1.TidbCluster, pvc *corev1.PersistentVolumeClaim, pod *corev1.Pod) error {
	defer fpc.updatePVCTracker.inc()
	if fpc.updatePVCTracker.errorReady() {
		defer fpc.updatePVCTracker.reset()
		return fpc.updatePVCTracker.err
	}
	if pvc.Labels == nil {
		pvc.Labels = make(map[string]string)
	}
	if pvc.Annotations == nil {
		pvc.Annotations = make(map[string]string)
	}
	setIfNotEmpty(pvc.Labels, label.AppLabelKey, pod.Labels[label.AppLabelKey])
	setIfNotEmpty(pvc.Labels, label.OwnerLabelKey, pod.Labels[label.OwnerLabelKey])
	setIfNotEmpty(pvc.Labels, label.ClusterLabelKey, pod.Labels[label.ClusterLabelKey])
	setIfNotEmpty(pvc.Labels, label.ClusterIDLabelKey, pod.Labels[label.ClusterIDLabelKey])
	setIfNotEmpty(pvc.Labels, label.MemberIDLabelKey, pod.Labels[label.MemberIDLabelKey])
	setIfNotEmpty(pvc.Labels, label.StoreIDLabelKey, pod.Labels[label.StoreIDLabelKey])
	setIfNotEmpty(pvc.Annotations, label.AnnPodNameKey, pod.GetName())
	return fpc.PVCIndexer.Update(pvc)
}

var _ PVCControlInterface = &FakePVCControl{}
