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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/pkg/util/label"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
)

// PVControlInterface manages PVs used in TidbCluster
type PVControlInterface interface {
	PatchPVReclaimPolicy(*v1.TidbCluster, *corev1.PersistentVolume, corev1.PersistentVolumeReclaimPolicy) error
	UpdateMetaInfo(*v1.TidbCluster, *corev1.PersistentVolume) error
}

type realPVControl struct {
	kubeCli   kubernetes.Interface
	pvcLister corelisters.PersistentVolumeClaimLister
	recorder  record.EventRecorder
}

// NewRealPVControl creates a new PVControlInterface
func NewRealPVControl(
	kubeCli kubernetes.Interface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	recorder record.EventRecorder,
) PVControlInterface {
	return &realPVControl{
		kubeCli:   kubeCli,
		pvcLister: pvcLister,
		recorder:  recorder,
	}
}

func (rpc *realPVControl) PatchPVReclaimPolicy(tc *v1.TidbCluster, pv *corev1.PersistentVolume, reclaimPolicy corev1.PersistentVolumeReclaimPolicy) error {
	pvName := pv.GetName()
	patchBytes := []byte(fmt.Sprintf(`{"spec":{"persistentVolumeReclaimPolicy":"%s"}}`, reclaimPolicy))

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, err := rpc.kubeCli.CoreV1().PersistentVolumes().Patch(pvName, types.StrategicMergePatchType, patchBytes)
		return err
	})
	rpc.recordPVEvent("patch", tc, pvName, err)
	return err
}

func (rpc *realPVControl) UpdateMetaInfo(tc *v1.TidbCluster, pv *corev1.PersistentVolume) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	pvLabels := pv.GetLabels()
	pvName := pv.GetName()
	pvcRef := pv.Spec.ClaimRef
	if pvcRef == nil {
		glog.Warningf("PV: [%s] doesn't have a ClaimRef, skipping, TidbCluster: %s/%s", pvName, ns, tcName)
		return nil
	}

	pvcName := pvcRef.Name
	pvc, err := rpc.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil {
		if !apierrs.IsNotFound(err) {
			return err
		}

		glog.Warningf("PV: [%s]'s PVC: [%s/%s] doesn't exist, skipping. TidbCluster: %s", pvName, ns, pvcName, tcName)
		return nil
	}

	pvcLabels := pvc.GetLabels()
	app := pvc.Labels[label.AppLabelKey]
	podName := pvc.Annotations[label.AnnPodNameKey]
	clusterID := pvcLabels[label.ClusterIDLabelKey]
	memberID := pvcLabels[label.MemberIDLabelKey]
	storeID := pvcLabels[label.StoreIDLabelKey]

	if pv.Annotations == nil {
		pv.Annotations = make(map[string]string)
	}
	if pvLabels[label.NamespaceLabelKey] == ns &&
		pvLabels[label.AppLabelKey] == app &&
		pvLabels[label.ClusterIDLabelKey] == clusterID &&
		pvLabels[label.MemberIDLabelKey] == memberID &&
		pvLabels[label.StoreIDLabelKey] == storeID &&
		pv.Annotations[label.AnnPodNameKey] == podName {
		glog.V(4).Infof("pv %s already has labels and annotations synced, skipping. TidbCluster: %s/%s", pvName, ns, tcName)
		return nil
	}

	pvLabels[label.NamespaceLabelKey] = ns
	pvLabels[label.OwnerLabelKey] = pvcLabels[label.OwnerLabelKey]
	pvLabels[label.ClusterLabelKey] = pvcLabels[label.ClusterLabelKey]

	setIfNotEmpty(pvLabels, label.AppLabelKey, app)
	setIfNotEmpty(pvLabels, label.ClusterIDLabelKey, clusterID)
	setIfNotEmpty(pvLabels, label.MemberIDLabelKey, memberID)
	setIfNotEmpty(pvLabels, label.StoreIDLabelKey, storeID)
	setIfNotEmpty(pv.Annotations, label.AnnPodNameKey, podName)

	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, err = rpc.kubeCli.CoreV1().PersistentVolumes().Update(pv)
		if err != nil {
			glog.Errorf("failed to update PV: [%s], TidbCluster %s/%s, error: %v", pvName, ns, tcName, err)
			return err
		}

		glog.V(4).Infof("PV: [%s] updated successfully, TidbCluster: %s/%s", pvName, ns, tcName)
		return nil
	})

	rpc.recordPVEvent("update", tc, pvName, err)
	return err
}

func (rpc *realPVControl) recordPVEvent(verb string, tc *v1.TidbCluster, pvName string, err error) {
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
	updatePVTracker requestTracker
}

// NewFakePVControl returns a FakePVControl
func NewFakePVControl(pvInformer coreinformers.PersistentVolumeInformer, pvcInformer coreinformers.PersistentVolumeClaimInformer) *FakePVControl {
	return &FakePVControl{
		pvcInformer.Lister(),
		pvInformer.Informer().GetIndexer(),
		requestTracker{0, nil, 0},
	}
}

// SetUpdatePVError sets the error attributes of updatePVTracker
func (fpc *FakePVControl) SetUpdatePVError(err error, after int) {
	fpc.updatePVTracker.err = err
	fpc.updatePVTracker.after = after
}

// PatchPVReclaimPolicy patchs the reclaim policy of PV
func (fpc *FakePVControl) PatchPVReclaimPolicy(tc *v1.TidbCluster, pv *corev1.PersistentVolume, reclaimPolicy corev1.PersistentVolumeReclaimPolicy) error {
	defer fpc.updatePVTracker.inc()
	if fpc.updatePVTracker.errorReady() {
		defer fpc.updatePVTracker.reset()
		return fpc.updatePVTracker.err
	}
	pv.Spec.PersistentVolumeReclaimPolicy = reclaimPolicy

	return fpc.PVIndexer.Update(pv)
}

// UpdateMetaInfo update the meta info of pv
func (fpc *FakePVControl) UpdateMetaInfo(tc *v1.TidbCluster, pv *corev1.PersistentVolume) error {
	defer fpc.updatePVTracker.inc()
	if fpc.updatePVTracker.errorReady() {
		defer fpc.updatePVTracker.reset()
		return fpc.updatePVTracker.err
	}
	ns := tc.GetNamespace()
	pvcName := pv.Spec.ClaimRef.Name
	pvc, err := fpc.PVCLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil {
		return err
	}
	if pv.Labels == nil {
		pv.Labels = make(map[string]string)
	}
	if pv.Annotations == nil {
		pv.Annotations = make(map[string]string)
	}
	pv.Labels[label.NamespaceLabelKey] = ns
	pv.Labels[label.OwnerLabelKey] = pvc.Labels[label.OwnerLabelKey]
	pv.Labels[label.ClusterLabelKey] = pvc.Labels[label.ClusterLabelKey]
	pv.Labels[label.OwnerLabelKey] = pvc.Labels[label.OwnerLabelKey]
	pv.Labels[label.ClusterLabelKey] = pvc.Labels[label.ClusterLabelKey]

	setIfNotEmpty(pv.Labels, label.AppLabelKey, pvc.Labels[label.AppLabelKey])
	setIfNotEmpty(pv.Labels, label.ClusterIDLabelKey, pvc.Labels[label.ClusterIDLabelKey])
	setIfNotEmpty(pv.Labels, label.MemberIDLabelKey, pvc.Labels[label.MemberIDLabelKey])
	setIfNotEmpty(pv.Labels, label.StoreIDLabelKey, pvc.Labels[label.StoreIDLabelKey])
	setIfNotEmpty(pv.Annotations, label.AnnPodNameKey, pvc.Annotations[label.AnnPodNameKey])
	return fpc.PVIndexer.Update(pv)
}

var _ PVControlInterface = &FakePVControl{}
