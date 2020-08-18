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
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	coreinformers "k8s.io/client-go/informers/core/v1"
	storageinformers "k8s.io/client-go/informers/storage/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	storagelisters "k8s.io/client-go/listers/storage/v1"
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
}

var (
	pdRequirement      = util.MustNewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.PDLabelVal})
	tikvRequirement    = util.MustNewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.TiKVLabelVal})
	tiflashRequirement = util.MustNewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.TiFlashLabelVal})
	pumpRequirement    = util.MustNewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.PumpLabelVal})
)

type pvcResizer struct {
	kubeCli   kubernetes.Interface
	pvcLister corelisters.PersistentVolumeClaimLister
	scLister  storagelisters.StorageClassLister
}

func (p *pvcResizer) Resize(tc *v1alpha1.TidbCluster) error {
	selector, err := label.New().Instance(tc.GetInstanceName()).Selector()
	if err != nil {
		return err
	}
	// patch PD PVCs
    if storageRequest, ok := tc.Spec.PD.Requests[corev1.ResourceStorage]; ok {
        err = p.patchPVCs(tc.GetNamespace(), selector.Add(*pdRequirement), storageRequest, "") 
        if err != nil {
            return err 
        }   
    }
	// patch TiKV PVCs
    if storageRequest, ok := tc.Spec.TiKV.Requests[corev1.ResourceStorage]; ok {
        err = p.patchPVCs(tc.GetNamespace(), selector.Add(*tikvRequirement), storageRequest, "")
        if err != nil {
            return err
        }
    }
	// patch TiFlash PVCs
	if tc.Spec.TiFlash != nil {
		for i, claim := range tc.Spec.TiFlash.StorageClaims {
			if storageRequest, ok := claim.Resources.Requests[corev1.ResourceStorage]; ok {
				prefix := fmt.Sprintf("data%d", i)
				err = p.patchPVCs(tc.GetNamespace(), selector.Add(*tiflashRequirement), storageRequest, prefix)
				if err != nil {
					return err
				}
			}
		}
	}
	// patch Pump PVCs
	if tc.Spec.Pump != nil {
		if storageRequest, ok := tc.Spec.Pump.Requests[corev1.ResourceStorage]; ok {
			err = p.patchPVCs(tc.GetNamespace(), selector.Add(*pumpRequirement), storageRequest, "")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *pvcResizer) isVolumeExpansionSupported(storageClassName string) (bool, error) {
	sc, err := p.scLister.Get(storageClassName)
	if err != nil {
		return false, err
	}
	if sc.AllowVolumeExpansion == nil {
		return false, nil
	}
	return *sc.AllowVolumeExpansion, nil
}

// patchPVCs patches PVCs filtered by selector and prefix.
func (p *pvcResizer) patchPVCs(ns string, selector labels.Selector, storageRequest resource.Quantity, prefix string) error {
	pvcs, err := p.pvcLister.PersistentVolumeClaims(ns).List(selector)
	if err != nil {
		return err
	}
	mergePatch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"resources": corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageRequest,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	for _, pvc := range pvcs {
		if !strings.HasPrefix(pvc.Name, prefix) {
			continue
		}
		if pvc.Spec.StorageClassName == nil {
			klog.Warningf("PVC %s/%s has no storage class, skipped", pvc.Namespace, pvc.Name)
			continue
		}
		volumeExpansionSupported, err := p.isVolumeExpansionSupported(*pvc.Spec.StorageClassName)
		if err != nil {
			return err
		}
		if !volumeExpansionSupported {
			klog.Warningf("Storage Class %q used by PVC %s/%s does not support volume expansion, skipped", *pvc.Spec.StorageClassName, pvc.Namespace, pvc.Name)
			continue
		}
		if currentRequest, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; !ok || storageRequest.Cmp(currentRequest) > 0 {
			_, err = p.kubeCli.CoreV1().PersistentVolumeClaims(pvc.Namespace).Patch(pvc.Name, types.MergePatchType, mergePatch)
			if err != nil {
				return err
			}
			klog.V(2).Infof("PVC %s/%s storage request is updated from %s to %s", pvc.Namespace, pvc.Name, currentRequest.String(), storageRequest.String())
		} else if storageRequest.Cmp(currentRequest) < 0 {
			klog.Warningf("PVC %s/%s/ storage request cannot be shrunk (%s to %s), skipped", pvc.Namespace, pvc.Name, currentRequest.String(), storageRequest.String())
		} else {
			klog.V(4).Infof("PVC %s/%s storage request is already %s, skipped", pvc.Namespace, pvc.Name, storageRequest.String())
		}
	}
	return nil
}

func NewPVCResizer(kubeCli kubernetes.Interface, pvcInformer coreinformers.PersistentVolumeClaimInformer, storageClassInformer storageinformers.StorageClassInformer) PVCResizerInterface {
	return &pvcResizer{
		kubeCli:   kubeCli,
		pvcLister: pvcInformer.Lister(),
		scLister:  storageClassInformer.Lister(),
	}
}

type fakePVCResizer struct {
}

func (f *fakePVCResizer) Resize(_ *v1alpha1.TidbCluster) error {
	return nil
}

func NewFakePVCResizer() PVCResizerInterface {
	return &fakePVCResizer{}
}
