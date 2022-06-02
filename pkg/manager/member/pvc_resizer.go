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
	pod *corev1.Pod
	// key: volume name in pod, value: autacl pvc
	volToPVCs map[string]*corev1.PersistentVolumeClaim
}

type componentVolumeContext struct {
	comp v1alpha1.MemberType
	// label selector for pvc and pod
	selector labels.Selector
	// desiredVolumeSpec is the volume request in tc spec
	// key: volume name in pod, value: volume request in spec
	desiredVolumeQuantity map[string]resource.Quantity
	// function to update volume status in tc status
	updateVolumeStatusFn func(map[string]*v1alpha1.StorageVolumeStatus)

	actualPodVolumes []*podVolumeContext
}

func (p *pvcResizer) Sync(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	name := tc.GetName()

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

	for _, comp := range components {
		ctx, err := p.buildContextForTC(tc, comp)
		if err != nil {
			return fmt.Errorf("sync pvc for tc %s/%s failed: failed to prepare: %v", ns, name, err)
		}

		p.updateVolumeStatus(ctx)

		err = p.resizeVolumes(ctx)
		if err != nil {
			return fmt.Errorf("sync pvc for tc %s/%s failed: resize volumes for %q failed: %v", ns, name, comp, err)
		}
	}

	return nil
}

func (p *pvcResizer) SyncDM(dc *v1alpha1.DMCluster) error {
	ns := dc.GetNamespace()
	name := dc.GetName()

	components := []v1alpha1.MemberType{}
	components = append(components, v1alpha1.DMMasterMemberType)
	if dc.Spec.Worker != nil {
		components = append(components, v1alpha1.DMWorkerMemberType)
	}

	for _, comp := range components {
		ctx, err := p.buildContextForDM(dc, comp)
		if err != nil {
			return fmt.Errorf("sync pvc for dc %s/%s failed: failed to prepare: %v", ns, name, err)
		}

		p.updateVolumeStatus(ctx)

		err = p.resizeVolumes(ctx)
		if err != nil {
			return fmt.Errorf("sync pvc for dc %s/%s failed: resize volumes for %q failed: %v", ns, name, comp, err)
		}
	}

	return nil
}

func (p *pvcResizer) buildContextForTC(tc *v1alpha1.TidbCluster, comp v1alpha1.MemberType) (*componentVolumeContext, error) {
	ns := tc.Namespace
	name := tc.Name

	ctx := &componentVolumeContext{
		comp:                  comp,
		desiredVolumeQuantity: map[string]resource.Quantity{},
	}

	selector, err := label.New().Instance(tc.GetInstanceName()).Selector()
	if err != nil {
		return nil, err
	}
	storageVolumes := []v1alpha1.StorageVolume{}
	switch comp {
	case v1alpha1.PDMemberType:
		ctx.selector = selector.Add(*pdRequirement)
		ctx.updateVolumeStatusFn = func(m map[string]*v1alpha1.StorageVolumeStatus) { tc.Status.PD.Volumes = m }
		if quantity, ok := tc.Spec.PD.Requests[corev1.ResourceStorage]; ok {
			ctx.desiredVolumeQuantity[v1alpha1.GetPVCTemplateName("", v1alpha1.PDMemberType)] = quantity
		}
		storageVolumes = tc.Spec.PD.StorageVolumes
	case v1alpha1.TiDBMemberType:
		ctx.selector = selector.Add(*tidbRequirement)
		ctx.updateVolumeStatusFn = func(m map[string]*v1alpha1.StorageVolumeStatus) { tc.Status.TiDB.Volumes = m }
		storageVolumes = tc.Spec.TiDB.StorageVolumes
	case v1alpha1.TiKVMemberType:
		ctx.selector = selector.Add(*tikvRequirement)
		ctx.updateVolumeStatusFn = func(m map[string]*v1alpha1.StorageVolumeStatus) { tc.Status.TiKV.Volumes = m }
		if quantity, ok := tc.Spec.TiKV.Requests[corev1.ResourceStorage]; ok {
			ctx.desiredVolumeQuantity[v1alpha1.GetPVCTemplateName("", v1alpha1.TiKVMemberType)] = quantity
		}
		storageVolumes = tc.Spec.TiKV.StorageVolumes
	case v1alpha1.TiFlashMemberType:
		ctx.selector = selector.Add(*tiflashRequirement)
		ctx.updateVolumeStatusFn = func(m map[string]*v1alpha1.StorageVolumeStatus) { tc.Status.TiFlash.Volumes = m }
		for i, claim := range tc.Spec.TiFlash.StorageClaims {
			if quantity, ok := claim.Resources.Requests[corev1.ResourceStorage]; ok {
				ctx.desiredVolumeQuantity[v1alpha1.GetPVCTemplateNameForTiFlash(i)] = quantity
			}
		}
	case v1alpha1.TiCDCMemberType:
		ctx.selector = selector.Add(*ticdcRequirement)
		ctx.updateVolumeStatusFn = func(m map[string]*v1alpha1.StorageVolumeStatus) { tc.Status.TiCDC.Volumes = m }
		storageVolumes = tc.Spec.TiCDC.StorageVolumes
	case v1alpha1.PumpMemberType:
		ctx.selector = selector.Add(*pumpRequirement)
		ctx.updateVolumeStatusFn = func(m map[string]*v1alpha1.StorageVolumeStatus) { tc.Status.Pump.Volumes = m }
		if quantity, ok := tc.Spec.Pump.Requests[corev1.ResourceStorage]; ok {
			ctx.desiredVolumeQuantity[v1alpha1.GetPVCTemplateName("", v1alpha1.PumpMemberType)] = quantity
		}
	default:
		return nil, fmt.Errorf("unsupported member type %s", comp)
	}

	for _, sv := range storageVolumes {
		if quantity, err := resource.ParseQuantity(sv.StorageSize); err == nil {
			ctx.desiredVolumeQuantity[v1alpha1.GetPVCTemplateName(sv.Name, comp)] = quantity
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
		updateVolumeStatusFn:  nil,
		desiredVolumeQuantity: map[string]resource.Quantity{},
	}

	selector, err := label.NewDM().Instance(dc.GetInstanceName()).Selector()
	if err != nil {
		return nil, err
	}
	switch comp {
	case v1alpha1.DMMasterMemberType:
		ctx.selector = selector.Add(*dmMasterRequirement)
		if quantity, err := resource.ParseQuantity(dc.Spec.Master.StorageSize); err == nil {
			ctx.desiredVolumeQuantity[v1alpha1.GetPVCTemplateName("", v1alpha1.DMMasterMemberType)] = quantity
		}
	case v1alpha1.DMWorkerMemberType:
		ctx.selector = selector.Add(*dmWorkerRequirement)
		if quantity, err := resource.ParseQuantity(dc.Spec.Worker.StorageSize); err == nil {
			ctx.desiredVolumeQuantity[v1alpha1.GetPVCTemplateName("", v1alpha1.DMWorkerMemberType)] = quantity
		}
	default:
		return nil, fmt.Errorf("unsupported member type %s", comp)
	}

	podVolumes, err := p.collectAcutalStatus(ns, ctx.selector)
	if err != nil {
		return nil, err
	}
	ctx.actualPodVolumes = podVolumes

	return ctx, nil
}

// updateVolumeStatus build volume status from `actualPodVolumes` and call the `updateVolumeStatusFn` function.
func (p *pvcResizer) updateVolumeStatus(ctx *componentVolumeContext) {
	if ctx.updateVolumeStatusFn == nil {
		return
	}

	getCapacity := func(volName string, pvc *corev1.PersistentVolumeClaim) (resource.Quantity, resource.Quantity, bool) {
		var desired, actual resource.Quantity
		var exist bool

		if pvc.Status.Phase != corev1.ClaimBound {
			return desired, actual, false
		}
		desired, exist = ctx.desiredVolumeQuantity[volName]
		if !exist {
			return desired, actual, false
		}
		actual, exist = pvc.Status.Capacity[corev1.ResourceStorage]
		if !exist {
			return desired, actual, false
		}

		return desired, actual, true
	}

	allStatus := map[string]*v1alpha1.StorageVolumeStatus{}
	for _, podVolume := range ctx.actualPodVolumes {
		for volName, pvc := range podVolume.volToPVCs {
			desiredQuantity, actualQuantity, pred := getCapacity(volName, pvc)
			if !pred {
				continue
			}

			status, exist := allStatus[volName]
			if !exist {
				status = &v1alpha1.StorageVolumeStatus{
					Name:            volName,
					BoundCount:      0,
					CurrentCount:    0,
					ResizedCount:    0,
					CurrentCapacity: desiredQuantity,
					ResizedCapacity: desiredQuantity,
				}
				allStatus[volName] = status
			}

			status.BoundCount++
			if actualQuantity.Cmp(desiredQuantity) == 0 {
				status.ResizedCount++
			} else {
				status.CurrentCount++
				status.CurrentCapacity = actualQuantity
			}
		}
	}

	for _, status := range allStatus {
		if status.CurrentCapacity.Cmp(status.ResizedCapacity) == 0 {
			status.CurrentCount = status.ResizedCount
		}
	}

	ctx.updateVolumeStatusFn(allStatus)
}

// resizeVolumes resize PVCs by comparing `desiredVolumeQuantity` and `actualVolumeQuantity` in context.
func (p *pvcResizer) resizeVolumes(ctx *componentVolumeContext) error {
	desiredVolumeQuantity := ctx.desiredVolumeQuantity
	podVolumes := ctx.actualPodVolumes

	for _, podVolume := range podVolumes {
		for volName, pvc := range podVolume.volToPVCs {
			quantityInSpec, exist := desiredVolumeQuantity[volName]
			if !exist {
				continue
			}

			// not support default storage class
			if pvc.Spec.StorageClassName == nil {
				klog.Warningf("PVC %s/%s has no storage class, skipped", pvc.Namespace, pvc.Name)
				continue
			}

			// check whether the volume needs to be expanded
			currentRequest, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
			if !ok {
				klog.Warningf("PVC %s/%s storage request is empty, skipped", pvc.Namespace, pvc.Name)
				continue
			}
			cmpVal := quantityInSpec.Cmp(currentRequest)
			if cmpVal == 0 {
				klog.V(4).Infof("PVC %s/%s storage request is already %s, skipped", pvc.Namespace, pvc.Name, quantityInSpec.String())
				continue
			}
			if cmpVal < 0 {
				klog.Warningf("PVC %s/%s/ storage request cannot be shrunk (%s to %s), skipped",
					pvc.Namespace, pvc.Name, currentRequest.String(), quantityInSpec.String())
				continue
			}

			// check whether the storage class support
			if p.deps.StorageClassLister != nil {
				volumeExpansionSupported, err := p.isVolumeExpansionSupported(*pvc.Spec.StorageClassName)
				if err != nil {
					return err
				}
				if !volumeExpansionSupported {
					klog.Warningf("Storage Class %q used by PVC %s/%s does not support volume expansion, skipped",
						*pvc.Spec.StorageClassName, pvc.Namespace, pvc.Name)
					continue
				}
			} else {
				klog.V(4).Infof("Storage classes lister is unavailable, skip checking volume expansion support for PVC %s/%s with storage class %s. This may be caused by no relevant permissions",
					pvc.Namespace, pvc.Name, *pvc.Spec.StorageClassName)
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
			klog.V(2).Infof("PVC %s/%s storage request is updated from %s to %s",
				pvc.Namespace, pvc.Name, currentRequest.String(), quantityInSpec.String())
		}
	}

	return nil
}

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
		volToPVCs := map[string]*corev1.PersistentVolumeClaim{}

		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				pvc, err := findPVC(vol.PersistentVolumeClaim.ClaimName)
				if err != nil {
					// FIXME: log
					continue
				}
				volToPVCs[vol.Name] = pvc
			}
		}

		result = append(result, &podVolumeContext{
			pod:       pod,
			volToPVCs: volToPVCs,
		})
	}

	return result, nil
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
