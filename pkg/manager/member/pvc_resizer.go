// Copyright 2020 PingCAP, Inc.
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
	"context"
	"encoding/json"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	errutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
)

// PVCResizerInterface represents the interface of PVC Resizer.
// It patches the PVCs owned by tidb cluster according to the latest
// storage request specified by the user. See
// https://github.com/pingcap/tidb-operator/issues/3004 for more details.
//
// Implementation:
//
// for every unmatched PVC (desiredCapacity != actualCapacity)
//  if storageClass does not support VolumeExpansion, skip and continue
//  if not patched, patch
//
// We patch all PVCs at the same time. For many cloud storage plugins (e.g.
// AWS-EBS, GCE-PD), they support online file system expansion in latest
// Kubernetes (1.15+).
//
// Limitations:
//
// - Note that the current statfulset implementation does not allow
//   `volumeClaimTemplates` to be changed, so new PVCs created by statefulset
//   controller will use the old storage request.
// - This is best effort, before statefulset volume resize feature (e.g.
//   https://github.com/kubernetes/enhancements/pull/1848) to be implemented.
// - If the feature `ExpandInUsePersistentVolumes` is not enabled or the volume
//   plugin does not support, the pod referencing the volume must be deleted and
//   recreated after the `FileSystemResizePending` condition becomes true.
// - Shrinking volumes is not supported.
//
type PVCResizerInterface interface {
	Sync(*v1alpha1.TidbCluster) error
	SyncDM(*v1alpha1.DMCluster) error
}

var (
	pdRequirement      = util.MustNewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.PDLabelVal})
	tidbRequirement    = util.MustNewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.TiDBLabelVal})
	tikvRequirement    = util.MustNewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.TiKVLabelVal})
	tiflashRequirement = util.MustNewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.TiFlashLabelVal})
	ticdcRequirement   = util.MustNewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.TiCDCLabelVal})
	pumpRequirement    = util.MustNewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.PumpLabelVal})

	dmMasterRequirement = util.MustNewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.DMMasterLabelVal})
	dmWorkerRequirement = util.MustNewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.DMWorkerLabelVal})
)

type pvcResizer struct {
	deps *controller.Dependencies
}

type podVolumeContext struct {
	pod       *corev1.Pod
	volToPVCs map[v1alpha1.StorageVolumeName]*corev1.PersistentVolumeClaim
}

type componentVolumeContext struct {
	comp v1alpha1.MemberType
	// label selector for pvc and pod
	selector labels.Selector
	// desiredVolumeSpec is the volume request in tc spec
	desiredVolumeQuantity map[v1alpha1.StorageVolumeName]resource.Quantity
	// actualPodVolumes is the actual status for all volumes
	actualPodVolumes []*podVolumeContext
	// observedStatus is current volume status assembled from `actualPodVolumes`
	observedStatus map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus

	// sourceVolumeStatus is the volume status in tc status
	// NOTE: modifying it will modify the status in tc
	sourceVolumeStatus map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus
}

func (p *pvcResizer) Sync(tc *v1alpha1.TidbCluster) error {
	id := fmt.Sprintf("%s/%s", tc.Namespace, tc.Name)

	components := []v1alpha1.MemberType{}
	if tc.Spec.PD != nil {
		components = append(components, v1alpha1.PDMemberType)
	}
	if tc.Spec.TiDB != nil {
		components = append(components, v1alpha1.TiDBMemberType)
	}
	if tc.Spec.TiKV != nil {
		components = append(components, v1alpha1.TiKVMemberType)
	}
	if tc.Spec.TiFlash != nil {
		components = append(components, v1alpha1.TiFlashMemberType)
	}
	if tc.Spec.TiCDC != nil {
		components = append(components, v1alpha1.TiCDCMemberType)
	}
	if tc.Spec.Pump != nil {
		components = append(components, v1alpha1.PumpMemberType)
	}

	errs := []error{}
	for _, comp := range components {
		ctx, err := p.buildContextForTC(tc, comp)
		if err != nil {
			errs = append(errs, fmt.Errorf("sync pvc for %q in tc %q failed: failed to prepare: %v", comp, id, err))
			continue
		}

		err = p.resizeVolumes(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("sync pvc for %q in tc %q failed: resize volumes failed: %v", comp, id, err))
		}

		p.updateVolumeStatus(ctx)
	}

	return errutil.NewAggregate(errs)
}

func (p *pvcResizer) SyncDM(dc *v1alpha1.DMCluster) error {
	id := fmt.Sprintf("%s/%s", dc.Namespace, dc.Name)

	components := []v1alpha1.MemberType{}
	components = append(components, v1alpha1.DMMasterMemberType)
	if dc.Spec.Worker != nil {
		components = append(components, v1alpha1.DMWorkerMemberType)
	}

	errs := []error{}
	for _, comp := range components {
		ctx, err := p.buildContextForDM(dc, comp)
		if err != nil {
			errs = append(errs, fmt.Errorf("sync pvc for %q in dc %q failed: failed to prepare: %v", comp, id, err))
			continue
		}

		err = p.resizeVolumes(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("sync pvc for %q in dc %q failed: resize volumes failed: %v", comp, id, err))
		}

		p.updateVolumeStatus(ctx)
	}

	return errutil.NewAggregate(errs)
}

func (p *pvcResizer) buildContextForTC(tc *v1alpha1.TidbCluster, comp v1alpha1.MemberType) (*componentVolumeContext, error) {
	ns := tc.Namespace
	name := tc.Name

	ctx := &componentVolumeContext{
		comp:                  comp,
		desiredVolumeQuantity: map[v1alpha1.StorageVolumeName]resource.Quantity{},
	}

	selector, err := label.New().Instance(tc.GetInstanceName()).Selector()
	if err != nil {
		return nil, err
	}
	storageVolumes := []v1alpha1.StorageVolume{}
	switch comp {
	case v1alpha1.PDMemberType:
		ctx.selector = selector.Add(*pdRequirement)
		if tc.Status.PD.Volumes == nil {
			tc.Status.PD.Volumes = map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{}
		}
		ctx.sourceVolumeStatus = tc.Status.PD.Volumes
		if quantity, ok := tc.Spec.PD.Requests[corev1.ResourceStorage]; ok {
			ctx.desiredVolumeQuantity[v1alpha1.GetStorageVolumeName("", v1alpha1.PDMemberType)] = quantity
		}
		storageVolumes = tc.Spec.PD.StorageVolumes
	case v1alpha1.TiDBMemberType:
		ctx.selector = selector.Add(*tidbRequirement)
		if tc.Status.TiDB.Volumes == nil {
			tc.Status.TiDB.Volumes = map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{}
		}
		ctx.sourceVolumeStatus = tc.Status.TiDB.Volumes
		storageVolumes = tc.Spec.TiDB.StorageVolumes
	case v1alpha1.TiKVMemberType:
		ctx.selector = selector.Add(*tikvRequirement)
		if tc.Status.TiKV.Volumes == nil {
			tc.Status.TiKV.Volumes = map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{}
		}
		ctx.sourceVolumeStatus = tc.Status.TiKV.Volumes
		if quantity, ok := tc.Spec.TiKV.Requests[corev1.ResourceStorage]; ok {
			ctx.desiredVolumeQuantity[v1alpha1.GetStorageVolumeName("", v1alpha1.TiKVMemberType)] = quantity
		}
		storageVolumes = tc.Spec.TiKV.StorageVolumes
	case v1alpha1.TiFlashMemberType:
		ctx.selector = selector.Add(*tiflashRequirement)
		if tc.Status.TiFlash.Volumes == nil {
			tc.Status.TiFlash.Volumes = map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{}
		}
		ctx.sourceVolumeStatus = tc.Status.TiFlash.Volumes
		for i, claim := range tc.Spec.TiFlash.StorageClaims {
			if quantity, ok := claim.Resources.Requests[corev1.ResourceStorage]; ok {
				ctx.desiredVolumeQuantity[v1alpha1.GetStorageVolumeNameForTiFlash(i)] = quantity
			}
		}
	case v1alpha1.TiCDCMemberType:
		ctx.selector = selector.Add(*ticdcRequirement)
		if tc.Status.TiCDC.Volumes == nil {
			tc.Status.TiCDC.Volumes = map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{}
		}
		ctx.sourceVolumeStatus = tc.Status.TiCDC.Volumes
		storageVolumes = tc.Spec.TiCDC.StorageVolumes
	case v1alpha1.PumpMemberType:
		ctx.selector = selector.Add(*pumpRequirement)
		if tc.Status.Pump.Volumes == nil {
			tc.Status.Pump.Volumes = map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{}
		}
		ctx.sourceVolumeStatus = tc.Status.Pump.Volumes
		if quantity, ok := tc.Spec.Pump.Requests[corev1.ResourceStorage]; ok {
			ctx.desiredVolumeQuantity[v1alpha1.GetStorageVolumeName("", v1alpha1.PumpMemberType)] = quantity
		}
	default:
		return nil, fmt.Errorf("unsupported member type %s", comp)
	}

	for _, sv := range storageVolumes {
		if quantity, err := resource.ParseQuantity(sv.StorageSize); err == nil {
			ctx.desiredVolumeQuantity[v1alpha1.GetStorageVolumeName(sv.Name, comp)] = quantity
		} else {
			klog.Warningf("StorageVolume %q in %s/%s .spec.%s is invalid", sv.Name, ns, name, comp)
		}
	}

	podVolumes, err := p.collectAcutalStatus(ns, ctx.selector)
	if err != nil {
		return nil, err
	}
	ctx.actualPodVolumes = podVolumes

	return ctx, nil
}

func (p *pvcResizer) buildContextForDM(dc *v1alpha1.DMCluster, comp v1alpha1.MemberType) (*componentVolumeContext, error) {
	ns := dc.Namespace

	ctx := &componentVolumeContext{
		comp:                  comp,
		desiredVolumeQuantity: map[v1alpha1.StorageVolumeName]resource.Quantity{},
	}

	selector, err := label.NewDM().Instance(dc.GetInstanceName()).Selector()
	if err != nil {
		return nil, err
	}
	switch comp {
	case v1alpha1.DMMasterMemberType:
		ctx.selector = selector.Add(*dmMasterRequirement)
		if dc.Status.Master.Volumes == nil {
			dc.Status.Master.Volumes = map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{}
		}
		ctx.sourceVolumeStatus = dc.Status.Master.Volumes
		if quantity, err := resource.ParseQuantity(dc.Spec.Master.StorageSize); err == nil {
			ctx.desiredVolumeQuantity[v1alpha1.GetStorageVolumeName("", v1alpha1.DMMasterMemberType)] = quantity
		}
	case v1alpha1.DMWorkerMemberType:
		ctx.selector = selector.Add(*dmWorkerRequirement)
		if dc.Status.Worker.Volumes == nil {
			dc.Status.Worker.Volumes = map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{}
		}
		ctx.sourceVolumeStatus = dc.Status.Worker.Volumes
		if quantity, err := resource.ParseQuantity(dc.Spec.Worker.StorageSize); err == nil {
			ctx.desiredVolumeQuantity[v1alpha1.GetStorageVolumeName("", v1alpha1.DMWorkerMemberType)] = quantity
		}
	default:
		return nil, fmt.Errorf("unsupported member type %s", comp)
	}

	podVolumes, err := p.collectAcutalStatus(ns, ctx.selector)
	if err != nil {
		return nil, err
	}
	ctx.actualPodVolumes = podVolumes

	ctx.observedStatus = p.assembleObserveVolumeStatus(ctx)

	return ctx, nil
}

// updateVolumeStatus update the volume status in tc status
func (p *pvcResizer) updateVolumeStatus(ctx *componentVolumeContext) {
	if ctx.sourceVolumeStatus == nil {
		return
	}

	observedStatus := ctx.observedStatus
	for volName, status := range observedStatus {
		if _, exist := ctx.sourceVolumeStatus[volName]; !exist {
			ctx.sourceVolumeStatus[volName] = &v1alpha1.StorageVolumeStatus{
				Name: volName,
			}
		}
		ctx.sourceVolumeStatus[volName].ObservedStorageVolumeStatus = *status
	}
	for _, status := range ctx.sourceVolumeStatus {
		if _, exist := observedStatus[status.Name]; !exist {
			delete(ctx.sourceVolumeStatus, status.Name)
		}
	}
}

// resizeVolumes resize PVCs by comparing `desiredVolumeQuantity` and `actualVolumeQuantity` in context.
func (p *pvcResizer) resizeVolumes(ctx *componentVolumeContext) error {
	desiredVolumeQuantity := ctx.desiredVolumeQuantity
	podVolumes := ctx.actualPodVolumes

	for _, podVolume := range podVolumes {
		for volName, pvc := range podVolume.volToPVCs {
			logPrefix := fmt.Sprintf("Skip to resize PVC %s/%s", pvc.Namespace, pvc.Name)

			quantityInSpec, exist := desiredVolumeQuantity[volName]
			if !exist {
				klog.Warningf("%s: volume %s is not exist in desired volumes", logPrefix, volName)
				continue
			}

			// not support default storage class
			if pvc.Spec.StorageClassName == nil {
				klog.Warningf("%s: PVC has not storage class", logPrefix)
				continue
			}

			// check whether the volume needs to be expanded
			currentRequest, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
			if !ok {
				klog.Warningf("%s: storage request is empty", logPrefix)
				continue
			}
			cmpVal := quantityInSpec.Cmp(currentRequest)
			if cmpVal == 0 {
				klog.V(4).Infof("%s: storage request is already %s", logPrefix, quantityInSpec.String())
				continue
			}
			if cmpVal < 0 {
				klog.Warningf("%s: storage request cannot be shrunk (%s to %s)", logPrefix, currentRequest.String(), quantityInSpec.String())
				continue
			}

			// check whether the storage class support
			if p.deps.StorageClassLister != nil {
				volumeExpansionSupported, err := p.isVolumeExpansionSupported(*pvc.Spec.StorageClassName)
				if err != nil {
					return err
				}
				if !volumeExpansionSupported {
					klog.Warningf("%s: storage class %q does not support volume expansion", logPrefix, *pvc.Spec.StorageClassName)
					continue
				}
			} else {
				klog.V(4).Infof("Storage classes lister is unavailable, skip checking volume expansion support for PVC %s/%s with storage class %s. This may be caused by no relevant permissions",
					pvc.Namespace, pvc.Name, *pvc.Spec.StorageClassName)
			}

			volStatus, exist := ctx.observedStatus[volName]
			if !exist {
				klog.Warningf("%s: wait for status of volume %s to be collected", logPrefix, volName)
				continue
			}
			if volStatus.ResizingCount > 0 {
				klog.Warningf("%s: resizing count is %d and wait for other resize to finish", logPrefix, volStatus.ResizingCount)
				continue
			}

			// patch PVC to expand the storage
			mergePatch, err := json.Marshal(map[string]interface{}{
				"spec": map[string]interface{}{
					"resources": corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: quantityInSpec,
						},
					},
				},
			})
			if err != nil {
				return err
			}
			_, err = p.deps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Patch(context.TODO(), pvc.Name, types.MergePatchType, mergePatch, metav1.PatchOptions{})
			if err != nil {
				return err
			}

			// update the resizing count
			volStatus.ResizingCount++

			klog.V(2).Infof("PVC %s/%s storage request is updated from %s to %s",
				pvc.Namespace, pvc.Name, currentRequest.String(), quantityInSpec.String())
		}
	}

	return nil
}

// collectAcutalStatus list pods and volumes to build context
func (p *pvcResizer) collectAcutalStatus(ns string, selector labels.Selector) ([]*podVolumeContext, error) {
	pods, err := p.deps.PodLister.Pods(ns).List(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to list Pods: %v", err)
	}
	pvcs, err := p.deps.PVCLister.PersistentVolumeClaims(ns).List(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to list PVCs: %v", err)
	}

	result := make([]*podVolumeContext, 0, len(pods))
	findPVC := func(pvcName string) (*corev1.PersistentVolumeClaim, error) {
		for _, pvc := range pvcs {
			if pvc.Name == pvcName {
				return pvc, nil
			}
		}
		return nil, fmt.Errorf("failed to find PVC %s", pvcName)
	}

	for _, pod := range pods {
		volToPVCs := map[v1alpha1.StorageVolumeName]*corev1.PersistentVolumeClaim{}

		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				pvc, err := findPVC(vol.PersistentVolumeClaim.ClaimName)
				if err != nil {
					klog.Warningf("Failed to find PVC %s of Pod %s/%s, maybe some labels are lost",
						vol.PersistentVolumeClaim.ClaimName, pod.Namespace, pod.Name)
					continue
				}
				volToPVCs[v1alpha1.StorageVolumeName(vol.Name)] = pvc
			}
		}

		result = append(result, &podVolumeContext{
			pod:       pod,
			volToPVCs: volToPVCs,
		})
	}

	return result, nil
}

// assembleObserveVolumeStatus assemble volume status from `actualPodVolumes`
func (p *pvcResizer) assembleObserveVolumeStatus(ctx *componentVolumeContext) map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus {
	parse := func(volName v1alpha1.StorageVolumeName, pvc *corev1.PersistentVolumeClaim) (resource.Quantity, resource.Quantity, bool) {
		var desired, actual resource.Quantity
		var exist bool

		if pvc.Status.Phase != corev1.ClaimBound {
			klog.Warningf("PVC %s/%s is not bound", pvc.Namespace, pvc.Name)
			return desired, actual, false
		}
		desired, exist = ctx.desiredVolumeQuantity[volName]
		if !exist {
			klog.Warningf("PVC %s/%s does not exist in desired volumes", pvc.Namespace, pvc.Name)
			return desired, actual, false
		}
		actual, exist = pvc.Status.Capacity[corev1.ResourceStorage]
		if !exist {
			klog.Warningf("PVC %s/%s dose not have cacacity in status", pvc.Namespace, pvc.Name)
			return desired, actual, false
		}

		return desired, actual, true
	}

	// build observed status from `actualPodVolumes`
	observedStatus := map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus{}
	for _, podVolume := range ctx.actualPodVolumes {
		for volName, pvc := range podVolume.volToPVCs {
			desiredQuantity, actualQuantity, pred := parse(volName, pvc)
			if !pred {
				continue
			}

			resizing := false
			for _, cond := range pvc.Status.Conditions {
				switch cond.Type {
				case corev1.PersistentVolumeClaimResizing, corev1.PersistentVolumeClaimFileSystemResizePending:
					resizing = true
				}
			}

			status, exist := observedStatus[volName]
			if !exist {
				observedStatus[volName] = &v1alpha1.ObservedStorageVolumeStatus{
					BoundCount:    0,
					ResizingCount: 0,
					CurrentCount:  0,
					ResizedCount:  0,
					// CurrentCapacity is default to same as desired capacity, and maybe changed later if any
					// volume is reszing.
					CurrentCapacity: desiredQuantity,
					// ResizedCapacity is always same as desired capacity
					ResizedCapacity: desiredQuantity,
				}
				status = observedStatus[volName]
			}

			status.BoundCount++
			if actualQuantity.Cmp(desiredQuantity) == 0 {
				status.ResizedCount++
			} else {
				status.CurrentCount++
				status.CurrentCapacity = actualQuantity
			}
			if resizing {
				status.ResizingCount++
			}
		}
	}
	for _, status := range observedStatus {
		// all volumes are resized, reset the current count
		if status.CurrentCapacity.Cmp(status.ResizedCapacity) == 0 {
			status.CurrentCount = status.ResizedCount
		}
	}

	return observedStatus
}

func (p *pvcResizer) isVolumeExpansionSupported(storageClassName string) (bool, error) {
	sc, err := p.deps.StorageClassLister.Get(storageClassName)
	if err != nil {
		return false, err
	}
	if sc.AllowVolumeExpansion == nil {
		return false, nil
	}
	return *sc.AllowVolumeExpansion, nil
}

func NewPVCResizer(deps *controller.Dependencies) PVCResizerInterface {
	return &pvcResizer{
		deps: deps,
	}
}

type fakePVCResizer struct {
}

func (f *fakePVCResizer) Sync(_ *v1alpha1.TidbCluster) error {
	return nil
}

func (f *fakePVCResizer) SyncDM(_ *v1alpha1.DMCluster) error {
	return nil
}

func NewFakePVCResizer() PVCResizerInterface {
	return &fakePVCResizer{}
}
