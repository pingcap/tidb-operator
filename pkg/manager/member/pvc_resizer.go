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
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
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
	Resize(*v1alpha1.TidbCluster) error
	ResizeDM(*v1alpha1.DMCluster) error
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

// Resize will resize the PVCs defined in components storage requests in tc.Spec
// Take PD as an example, there are 2 possible places: tc.Spec.PD.Requests & tc.Spec.PD.StorageVolumes
// Note: TiFlash is an exception for now, which uses tc.Spec.TiFlash.StorageClaims
func (p *pvcResizer) Resize(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	selector, err := label.New().Instance(tc.GetInstanceName()).Selector()
	if err != nil {
		return err
	}

	// For each component, we compose a map called pvcPrefix2Quantity, with PVC name prefix for current component as keys,
	// for example "pd-${tcName}-pd" (for tc.Spec.PD.Requests) or "pd-log-${tcName}-pd" (for tc.Spec.PD.storageVolumes elements with name "log").
	// Reference implementation of BuildStorageVolumeAndVolumeMount().
	// Note: for TiFlash, it is currently "data0-${tcName}-tiflash" (for tc.Spec.TiFlash.StorageClaims elements, in list definition order)

	// patch PD PVCs
	if tc.Spec.PD != nil {
		pvcPrefix2Quantity := make(map[string]resource.Quantity)
		pdMemberType := v1alpha1.PDMemberType.String()
		if quantity, ok := tc.Spec.PD.Requests[corev1.ResourceStorage]; ok {
			key := fmt.Sprintf("%s-%s-%s", pdMemberType, tc.Name, pdMemberType)
			pvcPrefix2Quantity[key] = quantity
		}
		for _, sv := range tc.Spec.PD.StorageVolumes {
			key := fmt.Sprintf("%s-%s-%s-%s", pdMemberType, sv.Name, tc.Name, pdMemberType)
			if quantity, err := resource.ParseQuantity(sv.StorageSize); err == nil {
				pvcPrefix2Quantity[key] = quantity
			} else {
				klog.Warningf("StorageVolume %q in %s/%s .Spec.PD is invalid", sv.Name, ns, tc.Name)
			}
		}
		if err := p.patchPVCs(ns, selector.Add(*pdRequirement), pvcPrefix2Quantity); err != nil {
			return err
		}
	}
	// patch TiDB PVCs
	if tc.Spec.TiDB != nil {
		pvcPrefix2Quantity := make(map[string]resource.Quantity)
		tidbMemberType := v1alpha1.TiDBMemberType.String()
		for _, sv := range tc.Spec.TiDB.StorageVolumes {
			key := fmt.Sprintf("%s-%s-%s-%s", tidbMemberType, sv.Name, tc.Name, tidbMemberType)
			if quantity, err := resource.ParseQuantity(sv.StorageSize); err == nil {
				pvcPrefix2Quantity[key] = quantity
			} else {
				klog.Warningf("StorageVolume %q in %s/%s .Spec.TiDB is invalid", sv.Name, ns, tc.Name)
			}
		}
		if err := p.patchPVCs(ns, selector.Add(*tidbRequirement), pvcPrefix2Quantity); err != nil {
			return err
		}
	}
	// patch TiKV PVCs
	if tc.Spec.TiKV != nil {
		pvcPrefix2Quantity := make(map[string]resource.Quantity)
		tikvMemberType := v1alpha1.TiKVMemberType.String()
		if quantity, ok := tc.Spec.TiKV.Requests[corev1.ResourceStorage]; ok {
			key := fmt.Sprintf("%s-%s-%s", tikvMemberType, tc.Name, tikvMemberType)
			pvcPrefix2Quantity[key] = quantity
		}
		for _, sv := range tc.Spec.TiKV.StorageVolumes {
			key := fmt.Sprintf("%s-%s-%s-%s", tikvMemberType, sv.Name, tc.Name, tikvMemberType)
			if quantity, err := resource.ParseQuantity(sv.StorageSize); err == nil {
				pvcPrefix2Quantity[key] = quantity
			} else {
				klog.Warningf("StorageVolume %q in %s/%s .Spec.TiKV is invalid", sv.Name, ns, tc.Name)
			}
		}
		if err := p.patchPVCs(ns, selector.Add(*tikvRequirement), pvcPrefix2Quantity); err != nil {
			return err
		}
	}
	// patch TiFlash PVCs
	if tc.Spec.TiFlash != nil {
		pvcPrefix2Quantity := make(map[string]resource.Quantity)
		tiflashMemberType := v1alpha1.TiFlashMemberType.String()
		for i, claim := range tc.Spec.TiFlash.StorageClaims {
			key := fmt.Sprintf("data%d-%s-%s", i, tc.Name, tiflashMemberType)
			if quantity, ok := claim.Resources.Requests[corev1.ResourceStorage]; ok {
				pvcPrefix2Quantity[key] = quantity
			}
		}
		if err := p.patchPVCs(ns, selector.Add(*tiflashRequirement), pvcPrefix2Quantity); err != nil {
			return err
		}
	}
	// patch TiCDC PVCs
	if tc.Spec.TiCDC != nil {
		pvcPrefix2Quantity := make(map[string]resource.Quantity)
		ticdcMemberType := v1alpha1.TiCDCMemberType.String()
		for _, sv := range tc.Spec.TiCDC.StorageVolumes {
			key := fmt.Sprintf("%s-%s-%s-%s", ticdcMemberType, sv.Name, tc.Name, ticdcMemberType)
			if quantity, err := resource.ParseQuantity(sv.StorageSize); err == nil {
				pvcPrefix2Quantity[key] = quantity
			} else {
				klog.Warningf("StorageVolume %q in %s/%s .Spec.TiCDC is invalid", sv.Name, ns, tc.Name)
			}
		}
		if err := p.patchPVCs(ns, selector.Add(*ticdcRequirement), pvcPrefix2Quantity); err != nil {
			return err
		}
	}
	// patch Pump PVCs
	if tc.Spec.Pump != nil {
		pvcPrefix2Quantity := make(map[string]resource.Quantity)
		pumpMemberType := v1alpha1.PumpMemberType.String()
		if quantity, ok := tc.Spec.Pump.Requests[corev1.ResourceStorage]; ok {
			key := fmt.Sprintf("data-%s-%s", tc.Name, pumpMemberType)
			pvcPrefix2Quantity[key] = quantity
		}
		if err := p.patchPVCs(ns, selector.Add(*pumpRequirement), pvcPrefix2Quantity); err != nil {
			return err
		}
	}
	return nil
}

// ResizeDM do things similar to Resize for TidbCluster
func (p *pvcResizer) ResizeDM(dc *v1alpha1.DMCluster) error {
	ns := dc.GetNamespace()
	selector, err := label.NewDM().Instance(dc.GetInstanceName()).Selector()
	if err != nil {
		return err
	}

	// patch dm-master PVCs
	{
		pvcPrefix2Quantity := make(map[string]resource.Quantity)
		if quantity, err := resource.ParseQuantity(dc.Spec.Master.StorageSize); err == nil {
			dmMasterMemberType := v1alpha1.DMMasterMemberType.String()
			key := fmt.Sprintf("%s-%s-%s", dmMasterMemberType, dc.Name, dmMasterMemberType)
			pvcPrefix2Quantity[key] = quantity
		}
		if err := p.patchPVCs(ns, selector.Add(*dmMasterRequirement), pvcPrefix2Quantity); err != nil {
			return err
		}
	}

	// patch dm-worker PVCs
	if dc.Spec.Worker != nil {
		pvcPrefix2Quantity := make(map[string]resource.Quantity)
		if quantity, err := resource.ParseQuantity(dc.Spec.Worker.StorageSize); err == nil {
			dmWorkerMemberType := v1alpha1.DMWorkerMemberType.String()
			key := fmt.Sprintf("%s-%s-%s", dmWorkerMemberType, dc.Name, dmWorkerMemberType)
			pvcPrefix2Quantity[key] = quantity
		}
		if err := p.patchPVCs(ns, selector.Add(*dmWorkerRequirement), pvcPrefix2Quantity); err != nil {
			return err
		}
	}
	return nil
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

// patchPVCs patches PVCs filtered by selector and prefix.
func (p *pvcResizer) patchPVCs(ns string, selector labels.Selector, pvcQuantityInSpec map[string]resource.Quantity) error {
	if len(pvcQuantityInSpec) == 0 {
		return nil
	}
	pvcs, err := p.deps.PVCLister.PersistentVolumeClaims(ns).List(selector)
	if err != nil {
		return err
	}

	// the PVC name for StatefulSet will be ${pvcNameInTemplate}-${stsName}-${ordinal}, here we want to drop the ordinal
	rePvcPrefix := regexp.MustCompile(`^(.+)-\d+$`)
	for _, pvc := range pvcs {
		match := rePvcPrefix.FindStringSubmatch(pvc.Name)
		pvcPrefix := match[1]
		quantityInSpec, ok := pvcQuantityInSpec[pvcPrefix]
		if !ok {
			// TODO: PVC not specified in tc.spec, should we deal with it and raise a warning
			continue
		}

		if pvc.Spec.StorageClassName == nil {
			klog.Warningf("PVC %s/%s has no storage class, skipped", pvc.Namespace, pvc.Name)
			continue
		}

		currentRequest, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
		if !ok {
			klog.Warningf("PVC %s/%s storage request is empty, skipped", pvc.Namespace, pvc.Name)
			continue
		}

		if quantityInSpec.Cmp(currentRequest) > 0 {
			if p.deps.StorageClassLister != nil {
				volumeExpansionSupported, err := p.isVolumeExpansionSupported(*pvc.Spec.StorageClassName)
				if err != nil {
					return err
				}
				if !volumeExpansionSupported {
					klog.Warningf("Storage Class %q used by PVC %s/%s does not support volume expansion, skipped", *pvc.Spec.StorageClassName, pvc.Namespace, pvc.Name)
					continue
				}
			} else {
				klog.V(4).Infof("Storage classes lister is unavailable, skip checking volume expansion support for PVC %s/%s with storage class %s. This may be caused by no relevant permissions",
					pvc.Namespace, pvc.Name, *pvc.Spec.StorageClassName)
			}
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
			_, err = p.deps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Patch(pvc.Name, types.MergePatchType, mergePatch)
			if err != nil {
				return err
			}
			klog.V(2).Infof("PVC %s/%s storage request is updated from %s to %s", pvc.Namespace, pvc.Name, currentRequest.String(), quantityInSpec.String())
		} else if quantityInSpec.Cmp(currentRequest) < 0 {
			klog.Warningf("PVC %s/%s/ storage request cannot be shrunk (%s to %s), skipped", pvc.Namespace, pvc.Name, currentRequest.String(), quantityInSpec.String())
		} else {
			klog.V(4).Infof("PVC %s/%s storage request is already %s, skipped", pvc.Namespace, pvc.Name, quantityInSpec.String())
		}
	}
	return nil
}

func NewPVCResizer(deps *controller.Dependencies) PVCResizerInterface {
	return &pvcResizer{
		deps: deps,
	}
}

type fakePVCResizer struct {
}

func (f *fakePVCResizer) Resize(_ *v1alpha1.TidbCluster) error {
	return nil
}

func (f *fakePVCResizer) ResizeDM(_ *v1alpha1.DMCluster) error {
	return nil
}

func NewFakePVCResizer() PVCResizerInterface {
	return &fakePVCResizer{}
}
