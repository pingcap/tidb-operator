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

package member

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	skipReasonPVCCleanerIsNotPDOrTiKV            = "pvc cleaner: member type is not pd or tikv"
	skipReasonPVCCleanerDeferDeletePVCNotHasLock = "pvc cleaner: defer delete PVC not has schedule lock"
	skipReasonPVCCleanerPVCNotHasLock            = "pvc cleaner: pvc not has schedule lock"
	skipReasonPVCCleanerPodWaitingForScheduling  = "pvc cleaner: waiting for pod scheduling"
	skipReasonPVCCleanerPodNotFound              = "pvc cleaner: the corresponding pod of pvc has not been found"
	skipReasonPVCCleanerWaitingForPVCSync        = "pvc cleaner: waiting for pvc's meta info to be synced"
)

// PVCCleaner implements the logic for cleaning the pvc related resource
type PVCCleanerInterface interface {
	Clean(*v1alpha1.TidbCluster) (map[string]string, error)
}

type realPVCCleaner struct {
	podLister  corelisters.PodLister
	pvcControl controller.PVCControlInterface
	pvcLister  corelisters.PersistentVolumeClaimLister
}

// NewRealPVCCleaner returns a realPVCCleaner
func NewRealPVCCleaner(
	podLister corelisters.PodLister,
	pvcControl controller.PVCControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister) PVCCleanerInterface {
	return &realPVCCleaner{
		podLister,
		pvcControl,
		pvcLister,
	}
}

func (rpc *realPVCCleaner) Clean(tc *v1alpha1.TidbCluster) (map[string]string, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	// for unit test
	skipReason := map[string]string{}

	selector, err := label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).Selector()
	if err != nil {
		return skipReason, fmt.Errorf("cluster %s/%s assemble label selector failed, err: %v", ns, tcName, err)
	}

	pvcs, err := rpc.pvcLister.PersistentVolumeClaims(ns).List(selector)
	if err != nil {
		return skipReason, fmt.Errorf("cluster %s/%s list pvc failed, selector: %s, err: %v", ns, tcName, selector, err)
	}

	for _, pvc := range pvcs {
		pvcName := pvc.GetName()
		l := label.Label(pvc.Labels)
		if !(l.IsPD() || l.IsTiKV()) {
			skipReason[pvcName] = skipReasonPVCCleanerIsNotPDOrTiKV
			continue
		}

		if pvc.Annotations[label.AnnPVCDeferDeleting] != "" {
			if _, exist := pvc.Annotations[label.AnnPVCPodScheduling]; !exist {
				// The defer deleting PVC without pod scheduling annotation, do nothing
				glog.V(4).Infof("cluster %s/%s defer delete pvc %s has not pod scheduling annotation, skip clean", ns, tcName, pvcName)
				skipReason[pvcName] = skipReasonPVCCleanerDeferDeletePVCNotHasLock
				continue
			}
			// The defer deleting PVC has pod scheduling annotation, so we need to delete the pod scheduling annotation
			delete(pvc.Annotations, label.AnnPVCPodScheduling)
			if _, err := rpc.pvcControl.UpdatePVC(tc, pvc); err != nil {
				return skipReason, fmt.Errorf("cluster %s/%s remove pvc %s pod scheduling annotation faild, err: %v", ns, tcName, pvcName, err)
			}
			continue
		}

		podName, exist := pvc.Annotations[label.AnnPodNameKey]
		if !exist {
			// waiting for pvc's meta info to be synced
			skipReason[pvcName] = skipReasonPVCCleanerWaitingForPVCSync
			continue
		}

		pod, err := rpc.podLister.Pods(ns).Get(podName)
		if err != nil {
			if !errors.IsNotFound(err) {
				return skipReason, fmt.Errorf("cluster %s/%s get pvc %s pod %s failed, err: %v", ns, tcName, pvcName, podName, err)
			}
			skipReason[pvcName] = skipReasonPVCCleanerPodNotFound
			continue
		}

		if _, exist := pvc.Annotations[label.AnnPVCPodScheduling]; !exist {
			// The PVC without pod scheduling annotation, do nothing
			glog.V(4).Infof("cluster %s/%s pvc %s has not pod scheduling annotation, skip clean", ns, tcName, pvcName)
			skipReason[pvcName] = skipReasonPVCCleanerPVCNotHasLock
			continue
		}

		if pvc.Status.Phase != corev1.ClaimBound || pod.Spec.NodeName == "" {
			// This pod has not been scheduled yet, no need to clean up the pvc pod schedule annotation
			glog.V(4).Infof("cluster %s/%s pod %s has not been scheduled yet, skip clean pvc %s pod schedule annotation", ns, tcName, podName, pvcName)
			skipReason[pvcName] = skipReasonPVCCleanerPodWaitingForScheduling
			continue
		}

		delete(pvc.Annotations, label.AnnPVCPodScheduling)
		if _, err := rpc.pvcControl.UpdatePVC(tc, pvc); err != nil {
			return skipReason, fmt.Errorf("cluster %s/%s remove pvc %s pod scheduling annotation faild, err: %v", ns, tcName, pvcName, err)
		}
		glog.Infof("cluster %s/%s, clean pvc %s pod scheduling annotation successfully", ns, tcName, pvcName)
	}

	return skipReason, nil
}

var _ PVCCleanerInterface = &realPVCCleaner{}

type FakePVCCleaner struct {
	err error
}

// NewFakePVCCleaner returns a fake PVC cleaner
func NewFakePVCCleaner() *FakePVCCleaner {
	return &FakePVCCleaner{}
}

func (fpc *FakePVCCleaner) SetPVCCleanerError(err error) {
	fpc.err = err
}

func (fpc *FakePVCCleaner) Clean(_ *v1alpha1.TidbCluster) (map[string]string, error) {
	return nil, fpc.err
}

var _ PVCCleanerInterface = &FakePVCCleaner{}
