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

	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	corev1 "k8s.io/api/core/v1"
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
}

type realPVControl struct {
	kubeCli  kubernetes.Interface
	pvLister corelisters.PersistentVolumeLister
	recorder record.EventRecorder
}

// NewRealPVControl creates a new PVControlInterface
func NewRealPVControl(kubeCli kubernetes.Interface, pvLister corelisters.PersistentVolumeLister, recorder record.EventRecorder) PVControlInterface {
	return &realPVControl{
		kubeCli,
		pvLister,
		recorder,
	}
}

func (rpc *realPVControl) PatchPVReclaimPolicy(tc *v1.TidbCluster, pv *corev1.PersistentVolume, reclaimPolicy corev1.PersistentVolumeReclaimPolicy) error {
	pvName := pv.GetName()
	patchBytes := []byte(fmt.Sprintf(`{"spec":{"persistentVolumeReclaimPolicy":"%s"}}`, reclaimPolicy))

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, err := rpc.kubeCli.CoreV1().PersistentVolumes().Patch(pvName, types.StrategicMergePatchType, patchBytes)
		return err
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
	PVLister        corelisters.PersistentVolumeLister
	PVIndexer       cache.Indexer
	updatePVTracker requestTracker
}

// NewFakePVControl returns a FakePVControl
func NewFakePVControl(pvInformer coreinformers.PersistentVolumeInformer) *FakePVControl {
	return &FakePVControl{
		pvInformer.Lister(),
		pvInformer.Informer().GetIndexer(),
		requestTracker{0, nil, 0},
	}
}

// SetUpdatePVError sets the error attributes of updatePVTracker
func (ssc *FakePVControl) SetUpdatePVError(err error, after int) {
	ssc.updatePVTracker.err = err
	ssc.updatePVTracker.after = after
}

// PatchPVReclaimPolicy patchs the reclaim policy of PV
func (ssc *FakePVControl) PatchPVReclaimPolicy(tc *v1.TidbCluster, pv *corev1.PersistentVolume, reclaimPolicy corev1.PersistentVolumeReclaimPolicy) error {
	defer ssc.updatePVTracker.inc()
	if ssc.updatePVTracker.errorReady() {
		defer ssc.updatePVTracker.reset()
		return ssc.updatePVTracker.err
	}
	pv.Spec.PersistentVolumeReclaimPolicy = reclaimPolicy

	return ssc.PVIndexer.Update(pv)
}

var _ PVControlInterface = &FakePVControl{}
