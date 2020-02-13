// Copyright 2019 PingCAP, Inc.
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
	"k8s.io/apimachinery/pkg/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

// GeneralPVCControlInterface manages PVCs used in backup and restore's pvc
type GeneralPVCControlInterface interface {
	CreatePVC(object runtime.Object, pvc *corev1.PersistentVolumeClaim) error
}

type realGeneralPVCControl struct {
	kubeCli  kubernetes.Interface
	recorder record.EventRecorder
}

// NewRealGeneralPVCControl creates a new GeneralPVCControlInterface
func NewRealGeneralPVCControl(
	kubeCli kubernetes.Interface,
	recorder record.EventRecorder,
) GeneralPVCControlInterface {
	return &realGeneralPVCControl{
		kubeCli:  kubeCli,
		recorder: recorder,
	}
}

func (gpc *realGeneralPVCControl) CreatePVC(object runtime.Object, pvc *corev1.PersistentVolumeClaim) error {
	ns := pvc.GetNamespace()
	pvcName := pvc.GetName()
	instanceName := pvc.GetLabels()[label.InstanceLabelKey]
	kind := object.GetObjectKind().GroupVersionKind().Kind

	_, err := gpc.kubeCli.CoreV1().PersistentVolumeClaims(ns).Create(pvc)
	if err != nil {
		klog.Errorf("failed to create pvc: [%s/%s], %s: %s, %v", ns, pvcName, kind, instanceName, err)
	} else {
		klog.V(4).Infof("create pvc: [%s/%s] successfully, %s: %s", ns, pvcName, kind, instanceName)
	}
	gpc.recordPVCEvent("create", object, pvc, err)
	return err
}

func (gpc *realGeneralPVCControl) recordPVCEvent(verb string, obj runtime.Object, pvc *corev1.PersistentVolumeClaim, err error) {
	pvcName := pvc.GetName()
	ns := pvc.GetNamespace()
	instanceName := pvc.GetLabels()[label.InstanceLabelKey]
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s PVC %s/%s for %s/%s successful",
			strings.ToLower(verb), ns, pvcName, kind, instanceName)
		gpc.recorder.Event(obj, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s PVC %s/%s for %s/%s failed error: %s",
			strings.ToLower(verb), ns, pvcName, kind, instanceName, err)
		gpc.recorder.Event(obj, corev1.EventTypeWarning, reason, msg)
	}
}

var _ GeneralPVCControlInterface = &realGeneralPVCControl{}

// FakeGeneralPVCControl is a fake GeneralPVCControlInterface
type FakeGeneralPVCControl struct {
	PVCLister        corelisters.PersistentVolumeClaimLister
	PVCIndexer       cache.Indexer
	createPVCTracker RequestTracker
}

// NewFakeGeneralPVCControl returns a FakeGeneralPVCControl
func NewFakeGeneralPVCControl(pvcInformer coreinformers.PersistentVolumeClaimInformer) *FakeGeneralPVCControl {
	return &FakeGeneralPVCControl{
		pvcInformer.Lister(),
		pvcInformer.Informer().GetIndexer(),
		RequestTracker{},
	}
}

// SetCreatePVCError sets the error attributes of createPVCTracker
func (fjc *FakeGeneralPVCControl) SetCreatePVCError(err error, after int) {
	fjc.createPVCTracker.SetError(err).SetAfter(after)
}

// CreatePVC adds the pvc to PVCIndexer
func (fjc *FakeGeneralPVCControl) CreatePVC(_ runtime.Object, pvc *corev1.PersistentVolumeClaim) error {
	defer fjc.createPVCTracker.Inc()
	if fjc.createPVCTracker.ErrorReady() {
		defer fjc.createPVCTracker.Reset()
		return fjc.createPVCTracker.GetError()
	}

	return fjc.PVCIndexer.Add(pvc)
}

var _ GeneralPVCControlInterface = &FakeGeneralPVCControl{}
