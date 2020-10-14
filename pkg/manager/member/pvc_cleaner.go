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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
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
	Clean(metav1.Object) (map[string]string, error)
}

type realPVCCleaner struct {
	deps *controller.Dependencies
}

// NewRealPVCCleaner returns a realPVCCleaner
func NewRealPVCCleaner(deps *controller.Dependencies) PVCCleanerInterface {
	return &realPVCCleaner{
		deps: deps,
	}
}

func (rpc *realPVCCleaner) Clean(meta metav1.Object) (map[string]string, error) {
	if skipReason, err := rpc.cleanScheduleLock(meta); err != nil {
		return skipReason, err
	}
	return rpc.reclaimPV(meta)
}

// reclaimPV reclaims PV used by tidb cluster if necessary.
func (rpc *realPVCCleaner) reclaimPV(meta metav1.Object) (map[string]string, error) {
	var clusterType string
	switch meta := meta.(type) {
	case *v1alpha1.TidbCluster:
		if !meta.IsPVReclaimEnabled() {
			return nil, nil
		}
		clusterType = "tidbcluster"
	case *v1alpha1.DMCluster:
		if !meta.IsPVReclaimEnabled() {
			return nil, nil
		}
		clusterType = "dmcluster"
	}
	ns := meta.GetNamespace()
	metaName := meta.GetName()

	skipReason := map[string]string{}

	pvcs, err := rpc.listAllPVCs(meta)
	if err != nil {
		return skipReason, err
	}
	runtimeMeta := meta.(runtime.Object)

	for _, pvc := range pvcs {
		pvcName := pvc.GetName()
		l := label.Label(pvc.Labels)
		if !(l.IsPD() || l.IsTiKV() || l.IsTiFlash() || l.IsDMMaster() || l.IsDMWorker()) {
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

		_, err := rpc.deps.PodLister.Pods(ns).Get(podName)
		if err == nil {
			// PVC is still referenced by this pod, can't reclaim PV
			skipReason[pvcName] = skipReasonPVCCleanerPVCeferencedByPod
			continue
		}
		if !errors.IsNotFound(err) {
			return skipReason, fmt.Errorf("%s %s/%s get pvc %s pod %s from local cache failed, err: %v", clusterType, ns, metaName, pvcName, podName, err)
		}

		// if pod not found in cache, re-check from apiserver directly to make sure the pod really not exist
		_, err = rpc.deps.KubeClientset.CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
		if err == nil {
			// PVC is still referenced by this pod, can't reclaim PV
			skipReason[pvcName] = skipReasonPVCCleanerPVCeferencedByPod
			continue
		}
		if !errors.IsNotFound(err) {
			return skipReason, fmt.Errorf("%s %s/%s get pvc %s pod %s from apiserver failed, err: %v", clusterType, ns, metaName, pvcName, podName, err)
		}

		// Without pod reference this defer delete PVC, start to reclaim PV
		pvName := pvc.Spec.VolumeName
		pv, err := rpc.deps.PVLister.Get(pvName)
		if err != nil {
			if errors.IsNotFound(err) {
				skipReason[pvcName] = skipReasonPVCCleanerNotFoundPV
				continue
			}
			return skipReason, fmt.Errorf("%s %s/%s get pvc %s pv %s failed, err: %v", clusterType, ns, metaName, pvcName, pvName, err)
		}

		if pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimDelete {
			err := rpc.deps.PVControl.PatchPVReclaimPolicy(runtimeMeta, pv, corev1.PersistentVolumeReclaimDelete)
			if err != nil {
				return skipReason, fmt.Errorf("%s %s/%s patch pv %s to %s failed, err: %v", clusterType, ns, metaName, pvName, corev1.PersistentVolumeReclaimDelete, err)
			}
			klog.Infof("%s %s/%s patch pv %s to policy %s success", clusterType, ns, metaName, pvName, corev1.PersistentVolumeReclaimDelete)
		}

		apiPVC, err := rpc.deps.KubeClientset.CoreV1().PersistentVolumeClaims(ns).Get(pvcName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				skipReason[pvcName] = skipReasonPVCCleanerPVCNotFound
				continue
			}
			return skipReason, fmt.Errorf("%s %s/%s get pvc %s failed, err: %v", clusterType, ns, metaName, pvcName, err)
		}

		if apiPVC.UID != pvc.UID || apiPVC.ResourceVersion != pvc.ResourceVersion {
			skipReason[pvcName] = skipReasonPVCCleanerPVCChanged
			continue
		}

		if err := rpc.deps.PVCControl.DeletePVC(runtimeMeta, pvc); err != nil {
			return skipReason, fmt.Errorf("%s %s/%s delete pvc %s failed, err: %v", clusterType, ns, metaName, pvcName, err)
		}
		klog.Infof("%s %s/%s reclaim pv %s success, pvc %s", clusterType, ns, metaName, pvName, pvcName)
	}
	return skipReason, nil
}

// cleanScheduleLock cleans AnnPVCPodScheduling label if necessary.
func (rpc *realPVCCleaner) cleanScheduleLock(meta metav1.Object) (map[string]string, error) {
	ns := meta.GetNamespace()
	metaName := meta.GetName()
	skipReason := map[string]string{}

	var clusterType string
	switch meta.(type) {
	case *v1alpha1.TidbCluster:
		clusterType = "tidbcluster"
	case *v1alpha1.DMCluster:
		clusterType = "dmcluster"
	}
	pvcs, err := rpc.listAllPVCs(meta)
	if err != nil {
		return skipReason, err
	}
	runtimeMeta := meta.(runtime.Object)

	for _, pvc := range pvcs {
		pvcName := pvc.GetName()
		l := label.Label(pvc.Labels)
		if !(l.IsPD() || l.IsTiKV() || l.IsTiFlash() || l.IsDMMaster() || l.IsDMWorker()) {
			skipReason[pvcName] = skipReasonPVCCleanerIsNotTarget
			continue
		}

		if pvc.Annotations[label.AnnPVCDeferDeleting] != "" {
			if _, exist := pvc.Annotations[label.AnnPVCPodScheduling]; !exist {
				// The defer deleting PVC without pod scheduling annotation, do nothing
				klog.V(4).Infof("%s %s/%s defer delete pvc %s has not pod scheduling annotation, skip clean", clusterType, ns, metaName, pvcName)
				skipReason[pvcName] = skipReasonPVCCleanerDeferDeletePVCNotHasLock
				continue
			}
			// The defer deleting PVC has pod scheduling annotation, so we need to delete the pod scheduling annotation
			delete(pvc.Annotations, label.AnnPVCPodScheduling)
			if _, err := rpc.deps.PVCControl.UpdatePVC(runtimeMeta, pvc); err != nil {
				return skipReason, fmt.Errorf("%s %s/%s remove pvc %s pod scheduling annotation faild, err: %v", clusterType, ns, metaName, pvcName, err)
			}
			continue
		}

		podName, exist := pvc.Annotations[label.AnnPodNameKey]
		if !exist {
			// PVC has no pod name annotation, this is an unexpected PVC, skip it
			skipReason[pvcName] = skipReasonPVCCleanerPVCNotHasPodNameAnn
			continue
		}

		pod, err := rpc.deps.PodLister.Pods(ns).Get(podName)
		if err != nil {
			if !errors.IsNotFound(err) {
				return skipReason, fmt.Errorf("%s %s/%s get pvc %s pod %s failed, err: %v", clusterType, ns, metaName, pvcName, podName, err)
			}
			skipReason[pvcName] = skipReasonPVCCleanerPodNotFound
			continue
		}

		if _, exist := pvc.Annotations[label.AnnPVCPodScheduling]; !exist {
			// The PVC without pod scheduling annotation, do nothing
			klog.V(4).Infof("%s %s/%s pvc %s has not pod scheduling annotation, skip clean", clusterType, ns, metaName, pvcName)
			skipReason[pvcName] = skipReasonPVCCleanerPVCNotHasLock
			continue
		}

		if pvc.Status.Phase != corev1.ClaimBound || pod.Spec.NodeName == "" {
			// This pod has not been scheduled yet, no need to clean up the pvc pod schedule annotation
			klog.V(4).Infof("%s %s/%s pod %s has not been scheduled yet, skip clean pvc %s pod schedule annotation", clusterType, ns, metaName, podName, pvcName)
			skipReason[pvcName] = skipReasonPVCCleanerPodWaitingForScheduling
			continue
		}

		delete(pvc.Annotations, label.AnnPVCPodScheduling)
		if _, err := rpc.deps.PVCControl.UpdatePVC(runtimeMeta, pvc); err != nil {
			return skipReason, fmt.Errorf("%s %s/%s remove pvc %s pod scheduling annotation faild, err: %v", clusterType, ns, metaName, pvcName, err)
		}
		klog.Infof("%s %s/%s, clean pvc %s pod scheduling annotation successfully", clusterType, ns, metaName, pvcName)
	}

	return skipReason, nil
}

// listAllPVCs lists all PVCs used by the given tidb cluster.
func (rpc *realPVCCleaner) listAllPVCs(meta metav1.Object) ([]*corev1.PersistentVolumeClaim, error) {
	ns := meta.GetNamespace()
	metaName := meta.GetName()

	var (
		selector labels.Selector
		err      error
	)
	switch meta := meta.(type) {
	case *v1alpha1.TidbCluster:
		selector, err = label.New().Instance(meta.GetInstanceName()).Selector()
	case *v1alpha1.DMCluster:
		selector, err = label.NewDM().Instance(meta.GetInstanceName()).Selector()
	default:
		err = fmt.Errorf("realPVCCleaner.listAllPVCs: unknown meta spec %s", meta)
	}
	if err != nil {
		return nil, fmt.Errorf("cluster %s/%s assemble label selector failed, err: %v", ns, metaName, err)
	}

	pvcs, err := rpc.deps.PVCLister.PersistentVolumeClaims(ns).List(selector)
	if err != nil {
		return nil, fmt.Errorf("cluster %s/%s list pvc failed, selector: %s, err: %v", ns, metaName, selector, err)
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

func (fpc *FakePVCCleaner) Clean(_ metav1.Object) (map[string]string, error) {
	return nil, fpc.err
}

var _ PVCCleanerInterface = &FakePVCCleaner{}
