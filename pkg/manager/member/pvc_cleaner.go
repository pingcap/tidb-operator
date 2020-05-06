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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
)

const (
	skipReasonPVCCleanerIsNotTarget              = "pvc cleaner: member type is not pd, tikv or tiflash"
	skipReasonPVCCleanerDeferDeletePVCNotHasLock = "pvc cleaner: defer delete PVC not has schedule lock"
	skipReasonPVCCleanerPVCNotHasLock            = "pvc cleaner: pvc not has schedule lock"
	skipReasonPVCCleanerPodWaitingForScheduling  = "pvc cleaner: waiting for pod scheduling"
	skipReasonPVCCleanerPodNotFound              = "pvc cleaner: the corresponding pod of pvc has not been found"
	skipReasonPVCCleanerPVCNotBound              = "pvc cleaner: the pvc is not bound"
	skipReasonPVCCleanerPVCNotHasPodNameAnn      = "pvc cleaner: pvc has no pod name annotation"
	skipReasonPVCCleanerIsNotDeferDeletePVC      = "pvc cleaner: pvc has not been marked as defer delete pvc"
	skipReasonPVCCleanerPVCeferencedByPod        = "pvc cleaner: pvc is still referenced by a pod"
	skipReasonPVCCleanerNotFoundPV               = "pvc cleaner: not found pv bound to pvc"
	skipReasonPVCCleanerPVCHasBeenDeleted        = "pvc cleaner: pvc has been deleted"
	skipReasonPVCCleanerPVCNotFound              = "pvc cleaner: not found pvc from apiserver"
	skipReasonPVCCleanerPVCChanged               = "pvc cleaner: pvc changed before deletion"
)

// PVCCleaner implements the logic for cleaning the pvc related resource
type PVCCleanerInterface interface {
	Clean(*v1alpha1.TidbCluster) (map[string]string, error)
}

type realPVCCleaner struct {
	kubeCli    kubernetes.Interface
	podLister  corelisters.PodLister
	pvcControl controller.PVCControlInterface
	pvcLister  corelisters.PersistentVolumeClaimLister
	pvLister   corelisters.PersistentVolumeLister
	pvControl  controller.PVControlInterface
}

// NewRealPVCCleaner returns a realPVCCleaner
func NewRealPVCCleaner(
	kubeCli kubernetes.Interface,
	podLister corelisters.PodLister,
	pvcControl controller.PVCControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvLister corelisters.PersistentVolumeLister,
	pvControl controller.PVControlInterface) PVCCleanerInterface {
	return &realPVCCleaner{
		kubeCli,
		podLister,
		pvcControl,
		pvcLister,
		pvLister,
		pvControl,
	}
}

func (rpc *realPVCCleaner) Clean(tc *v1alpha1.TidbCluster) (map[string]string, error) {
	if skipReason, err := rpc.cleanScheduleLock(tc); err != nil {
		return skipReason, err
	}

	if !tc.IsPVReclaimEnabled() {
		// disable PV reclaim, return directly.
		return nil, nil
	}
	return rpc.reclaimPV(tc)
}

func (rpc *realPVCCleaner) reclaimPV(tc *v1alpha1.TidbCluster) (map[string]string, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	// for unit test
	skipReason := map[string]string{}

	pvcs, err := rpc.listAllPVCs(tc)
	if err != nil {
		return skipReason, err
	}

	for _, pvc := range pvcs {
		pvcName := pvc.GetName()
		l := label.Label(pvc.Labels)
		if !(l.IsPD() || l.IsTiKV() || l.IsTiFlash()) {
			skipReason[pvcName] = skipReasonPVCCleanerIsNotTarget
			continue
		}

		if pvc.Status.Phase != corev1.ClaimBound {
			// If pvc is not bound yet, it will not be processed
			skipReason[pvcName] = skipReasonPVCCleanerPVCNotBound
			continue
		}

		if pvc.DeletionTimestamp != nil {
			// PVC has been deleted, skip it
			skipReason[pvcName] = skipReasonPVCCleanerPVCHasBeenDeleted
			continue
		}

		if len(pvc.Annotations[label.AnnPVCDeferDeleting]) == 0 {
			// This pvc has not been marked as defer delete PVC, can't reclaim the PV bound to this PVC
			skipReason[pvcName] = skipReasonPVCCleanerIsNotDeferDeletePVC
			continue
		}

		// PVC has been marked as defer delete PVC, try to reclaim the PV bound to this PVC
		podName, exist := pvc.Annotations[label.AnnPodNameKey]
		if !exist {
			// PVC has not pod name annotation, this is an unexpected PVC, skip it
			skipReason[pvcName] = skipReasonPVCCleanerPVCNotHasPodNameAnn
			continue
		}

		_, err := rpc.podLister.Pods(ns).Get(podName)
		if err == nil {
			// PVC is still referenced by this pod, can't reclaim PV
			skipReason[pvcName] = skipReasonPVCCleanerPVCeferencedByPod
			continue
		}
		if !errors.IsNotFound(err) {
			return skipReason, fmt.Errorf("cluster %s/%s get pvc %s pod %s from local cache failed, err: %v", ns, tcName, pvcName, podName, err)
		}

		// if pod not found in cache, re-check from apiserver directly to make sure the pod really not exist
		_, err = rpc.kubeCli.CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
		if err == nil {
			// PVC is still referenced by this pod, can't reclaim PV
			skipReason[pvcName] = skipReasonPVCCleanerPVCeferencedByPod
			continue
		}
		if !errors.IsNotFound(err) {
			return skipReason, fmt.Errorf("cluster %s/%s get pvc %s pod %s from apiserver failed, err: %v", ns, tcName, pvcName, podName, err)
		}

		// Without pod reference this defer delete PVC, start to reclaim PV
		pvName := pvc.Spec.VolumeName
		pv, err := rpc.pvLister.Get(pvName)
		if err != nil {
			if errors.IsNotFound(err) {
				skipReason[pvcName] = skipReasonPVCCleanerNotFoundPV
				continue
			}
			return skipReason, fmt.Errorf("cluster %s/%s get pvc %s pv %s failed, err: %v", ns, tcName, pvcName, pvName, err)
		}

		if pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimDelete {
			err := rpc.pvControl.PatchPVReclaimPolicy(tc, pv, corev1.PersistentVolumeReclaimDelete)
			if err != nil {
				return skipReason, fmt.Errorf("cluster %s/%s patch pv %s to %s failed, err: %v", ns, tcName, pvName, corev1.PersistentVolumeReclaimDelete, err)
			}
			klog.Infof("cluster %s/%s patch pv %s to policy %s success", ns, tcName, pvName, corev1.PersistentVolumeReclaimDelete)
		}

		apiPVC, err := rpc.kubeCli.CoreV1().PersistentVolumeClaims(ns).Get(pvcName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				skipReason[pvcName] = skipReasonPVCCleanerPVCNotFound
				continue
			}
			return skipReason, fmt.Errorf("cluster %s/%s get pvc %s failed, err: %v", ns, tcName, pvcName, err)
		}

		if apiPVC.UID != pvc.UID || apiPVC.ResourceVersion != pvc.ResourceVersion {
			skipReason[pvcName] = skipReasonPVCCleanerPVCChanged
			continue
		}

		if err := rpc.pvcControl.DeletePVC(tc, pvc); err != nil {
			return skipReason, fmt.Errorf("cluster %s/%s delete pvc %s failed, err: %v", ns, tcName, pvcName, err)
		}
		klog.Infof("cluster %s/%s reclaim pv %s success, pvc %s", ns, tcName, pvName, pvcName)
	}
	return skipReason, nil
}

func (rpc *realPVCCleaner) cleanScheduleLock(tc *v1alpha1.TidbCluster) (map[string]string, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	// for unit test
	skipReason := map[string]string{}

	pvcs, err := rpc.listAllPVCs(tc)
	if err != nil {
		return skipReason, err
	}

	for _, pvc := range pvcs {
		pvcName := pvc.GetName()
		l := label.Label(pvc.Labels)
		if !(l.IsPD() || l.IsTiKV() || l.IsTiFlash()) {
			skipReason[pvcName] = skipReasonPVCCleanerIsNotTarget
			continue
		}

		if pvc.Annotations[label.AnnPVCDeferDeleting] != "" {
			if _, exist := pvc.Annotations[label.AnnPVCPodScheduling]; !exist {
				// The defer deleting PVC without pod scheduling annotation, do nothing
				klog.V(4).Infof("cluster %s/%s defer delete pvc %s has not pod scheduling annotation, skip clean", ns, tcName, pvcName)
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
			// PVC has no pod name annotation, this is an unexpected PVC, skip it
			skipReason[pvcName] = skipReasonPVCCleanerPVCNotHasPodNameAnn
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
			klog.V(4).Infof("cluster %s/%s pvc %s has not pod scheduling annotation, skip clean", ns, tcName, pvcName)
			skipReason[pvcName] = skipReasonPVCCleanerPVCNotHasLock
			continue
		}

		if pvc.Status.Phase != corev1.ClaimBound || pod.Spec.NodeName == "" {
			// This pod has not been scheduled yet, no need to clean up the pvc pod schedule annotation
			klog.V(4).Infof("cluster %s/%s pod %s has not been scheduled yet, skip clean pvc %s pod schedule annotation", ns, tcName, podName, pvcName)
			skipReason[pvcName] = skipReasonPVCCleanerPodWaitingForScheduling
			continue
		}

		delete(pvc.Annotations, label.AnnPVCPodScheduling)
		if _, err := rpc.pvcControl.UpdatePVC(tc, pvc); err != nil {
			return skipReason, fmt.Errorf("cluster %s/%s remove pvc %s pod scheduling annotation faild, err: %v", ns, tcName, pvcName, err)
		}
		klog.Infof("cluster %s/%s, clean pvc %s pod scheduling annotation successfully", ns, tcName, pvcName)
	}

	return skipReason, nil
}

func (rpc *realPVCCleaner) listAllPVCs(tc *v1alpha1.TidbCluster) ([]*corev1.PersistentVolumeClaim, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	selector, err := label.New().Instance(tc.GetInstanceName()).Selector()
	if err != nil {
		return nil, fmt.Errorf("cluster %s/%s assemble label selector failed, err: %v", ns, tcName, err)
	}

	pvcs, err := rpc.pvcLister.PersistentVolumeClaims(ns).List(selector)
	if err != nil {
		return nil, fmt.Errorf("cluster %s/%s list pvc failed, selector: %s, err: %v", ns, tcName, selector, err)
	}
	return pvcs, nil
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
