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
	"sort"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	errutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	storagelister "k8s.io/client-go/listers/storage/v1"
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
// We patch all PVCs of one Pod  at the same time. For many cloud storage plugins (e.g.
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

type volumePhase string

const (
	// needResize means the storage request of PVC is different from the storage request in TC/DC.
	needResize volumePhase = "NeedResize"
	// resizing means the storage request of PVC is equal to the storage request in TC/DC and PVC is resizing.
	resizing volumePhase = "Resizing"
	// resized means the storage request of PVC is equal to the storage request in TC/DC and PVC has been resized.
	resized volumePhase = "Resized"
)

type volume struct {
	name v1alpha1.StorageVolumeName
	pvc  *corev1.PersistentVolumeClaim
}

type podVolumeContext struct {
	pod     *corev1.Pod
	volumes []*volume
}

type componentVolumeContext struct {
	cluster metav1.Object
	status  v1alpha1.ComponentStatus

	// label selector for pvc and pod
	selector labels.Selector
	// desiredVolumeSpec is the volume request in tc spec
	desiredVolumeQuantity map[v1alpha1.StorageVolumeName]resource.Quantity
	// actualPodVolumes is the actual status for all volumes
	actualPodVolumes []*podVolumeContext
}

func (c *componentVolumeContext) ComponentID() string {
	return fmt.Sprintf("%s/%s:%s", c.cluster.GetNamespace(), c.cluster.GetName(), c.status.GetMemberType())
}

type pvcResizer struct {
	deps *controller.Dependencies
}

func NewPVCResizer(deps *controller.Dependencies) PVCResizerInterface {
	return &pvcResizer{
		deps: deps,
	}
}

func (p *pvcResizer) Sync(tc *v1alpha1.TidbCluster) error {
	components := v1alpha1.ComponentStatusFromTC(tc)
	errs := []error{}

	for _, comp := range components {
		ctx, err := p.buildContextForTC(tc, comp)
		if err != nil {
			errs = append(errs, fmt.Errorf("build ctx used by resize for %s failed: %w", ctx.ComponentID(), err))
			continue
		}

		p.updateVolumeStatus(ctx)

		err = p.resizeVolumes(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("resize volumes for %s failed: %w", ctx.ComponentID(), err))
			continue
		}
	}

	return errutil.NewAggregate(errs)
}

func (p *pvcResizer) SyncDM(dc *v1alpha1.DMCluster) error {
	components := v1alpha1.ComponentStatusFromDC(dc)
	errs := []error{}

	for _, comp := range components {
		ctx, err := p.buildContextForDM(dc, comp)
		if err != nil {
			errs = append(errs, fmt.Errorf("build ctx used by resize for %s failed: %w", ctx.ComponentID(), err))
			continue
		}

		p.updateVolumeStatus(ctx)

		err = p.resizeVolumes(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("resize volumes for %s failed: %w", ctx.ComponentID(), err))
			continue
		}
	}

	return errutil.NewAggregate(errs)
}

func (p *pvcResizer) buildContextForTC(tc *v1alpha1.TidbCluster, status v1alpha1.ComponentStatus) (*componentVolumeContext, error) {
	comp := status.GetMemberType()

	ctx := &componentVolumeContext{
		cluster:               tc,
		status:                status,
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
		if quantity, ok := tc.Spec.PD.Requests[corev1.ResourceStorage]; ok {
			ctx.desiredVolumeQuantity[v1alpha1.GetStorageVolumeName("", v1alpha1.PDMemberType)] = quantity
		}
		storageVolumes = tc.Spec.PD.StorageVolumes
	case v1alpha1.TiDBMemberType:
		ctx.selector = selector.Add(*tidbRequirement)
		if tc.Status.TiDB.Volumes == nil {
			tc.Status.TiDB.Volumes = map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{}
		}
		storageVolumes = tc.Spec.TiDB.StorageVolumes
	case v1alpha1.TiKVMemberType:
		ctx.selector = selector.Add(*tikvRequirement)
		if tc.Status.TiKV.Volumes == nil {
			tc.Status.TiKV.Volumes = map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{}
		}
		if quantity, ok := tc.Spec.TiKV.Requests[corev1.ResourceStorage]; ok {
			ctx.desiredVolumeQuantity[v1alpha1.GetStorageVolumeName("", v1alpha1.TiKVMemberType)] = quantity
		}
		storageVolumes = tc.Spec.TiKV.StorageVolumes
	case v1alpha1.TiFlashMemberType:
		ctx.selector = selector.Add(*tiflashRequirement)
		if tc.Status.TiFlash.Volumes == nil {
			tc.Status.TiFlash.Volumes = map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{}
		}
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
		storageVolumes = tc.Spec.TiCDC.StorageVolumes
	case v1alpha1.PumpMemberType:
		ctx.selector = selector.Add(*pumpRequirement)
		if tc.Status.Pump.Volumes == nil {
			tc.Status.Pump.Volumes = map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{}
		}
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
			klog.Warningf("StorageVolume %q in %s .spec.%s is invalid", sv.Name, ctx.ComponentID(), comp)
		}
	}

	podVolumes, err := p.collectAcutalStatus(ctx.cluster.GetNamespace(), ctx.selector)
	if err != nil {
		return nil, err
	}
	ctx.actualPodVolumes = podVolumes

	return ctx, nil
}

func (p *pvcResizer) buildContextForDM(dc *v1alpha1.DMCluster, status v1alpha1.ComponentStatus) (*componentVolumeContext, error) {
	comp := status.GetMemberType()

	ctx := &componentVolumeContext{
		cluster:               dc,
		status:                status,
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
		if quantity, err := resource.ParseQuantity(dc.Spec.Master.StorageSize); err == nil {
			ctx.desiredVolumeQuantity[v1alpha1.GetStorageVolumeName("", v1alpha1.DMMasterMemberType)] = quantity
		}
	case v1alpha1.DMWorkerMemberType:
		ctx.selector = selector.Add(*dmWorkerRequirement)
		if dc.Status.Worker.Volumes == nil {
			dc.Status.Worker.Volumes = map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{}
		}
		if quantity, err := resource.ParseQuantity(dc.Spec.Worker.StorageSize); err == nil {
			ctx.desiredVolumeQuantity[v1alpha1.GetStorageVolumeName("", v1alpha1.DMWorkerMemberType)] = quantity
		}
	default:
		return nil, fmt.Errorf("unsupported member type %s", comp)
	}

	podVolumes, err := p.collectAcutalStatus(ctx.cluster.GetNamespace(), ctx.selector)
	if err != nil {
		return nil, err
	}
	ctx.actualPodVolumes = podVolumes

	return ctx, nil
}

// updateVolumeStatus build volume status from `actualPodVolumes` and update `sourceVolumeStatus`.
func (p *pvcResizer) updateVolumeStatus(ctx *componentVolumeContext) {
	sourceVolumeStatus := ctx.status.GetVolumes()
	if sourceVolumeStatus == nil {
		return
	}

	getCapacity := func(volName v1alpha1.StorageVolumeName, pvc *corev1.PersistentVolumeClaim) (resource.Quantity, resource.Quantity, bool) {
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
	for _, podVolumes := range ctx.actualPodVolumes {
		for _, volume := range podVolumes.volumes {
			volName := volume.name
			pvc := volume.pvc

			desiredQuantity, actualQuantity, pred := getCapacity(volName, pvc)
			if !pred {
				continue
			}

			status, exist := observedStatus[volName]
			if !exist {
				observedStatus[volName] = &v1alpha1.ObservedStorageVolumeStatus{
					BoundCount:   0,
					CurrentCount: 0,
					ResizedCount: 0,
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
		}
	}
	for _, status := range observedStatus {
		// all volumes are resized, reset the current count
		if status.CurrentCapacity.Cmp(status.ResizedCapacity) == 0 {
			status.CurrentCount = status.ResizedCount
		}
	}

	// sync volume status for `sourceVolumeStatus`
	for volName, status := range observedStatus {
		if _, exist := sourceVolumeStatus[volName]; !exist {
			sourceVolumeStatus[volName] = &v1alpha1.StorageVolumeStatus{
				Name: volName,
			}
		}
		sourceVolumeStatus[volName].ObservedStorageVolumeStatus = *status
	}
	for _, status := range sourceVolumeStatus {
		if _, exist := observedStatus[status.Name]; !exist {
			delete(sourceVolumeStatus, status.Name)
		}
	}
}

// resizeVolumes resize PVCs by comparing `desiredVolumeQuantity` and `actualVolumeQuantity` in context.
func (p *pvcResizer) resizeVolumes(ctx *componentVolumeContext) error {
	var (
		resizingPod       *corev1.Pod
		classifiedVolumes map[volumePhase][]*volume
	)

	// choose one pod whose volume need to be resized
	for _, podVolumes := range ctx.actualPodVolumes {
		curClassifiedVolumes, err := p.classifyVolumes(ctx, podVolumes.volumes)
		if err != nil {
			return fmt.Errorf("classify volumes for %s failed: %w", ctx.ComponentID(), err)
		}

		if len(curClassifiedVolumes[resizing]) != 0 || len(curClassifiedVolumes[needResize]) != 0 {
			resizingPod = podVolumes.pod
			classifiedVolumes = curClassifiedVolumes
			break
		}
	}

	allResized := resizingPod == nil
	condResizing := meta.IsStatusConditionTrue(ctx.status.GetConditions(), v1alpha1.ComponentVolumeResizing)

	if allResized {
		if condResizing {
			return p.endResize(ctx)
		}
		klog.V(4).Infof("all volumes are resized for %s", ctx.ComponentID())
		return nil
	}

	if !condResizing {
		return p.beginResize(ctx)
	}

	// some volumes are resizing
	for _, volume := range classifiedVolumes[resizing] {
		klog.Infof("PVC %s/%s for %s is resizing", volume.pvc.Namespace, volume.pvc.Name, ctx.ComponentID())
	}

	// some volumes need to be resized
	if len(classifiedVolumes[needResize]) != 0 {
		klog.V(4).Infof("start to resize volumes of Pod %s/%s for %s", resizingPod.Namespace, resizingPod.Name, ctx.ComponentID())
		return p.resizeVolumesForPod(ctx, resizingPod, classifiedVolumes[needResize])
	}

	return nil
}

func (p *pvcResizer) classifyVolumes(ctx *componentVolumeContext, volumes []*volume) (map[volumePhase][]*volume, error) {
	desiredVolumeQuantity := ctx.desiredVolumeQuantity
	cid := ctx.ComponentID()

	needResizeVolumes := []*volume{}
	resizingVolumes := []*volume{}
	resizedVolumes := []*volume{}

	for _, volume := range volumes {
		volName := volume.name
		pvc := volume.pvc
		pvcID := fmt.Sprintf("%s/%s", pvc.Namespace, pvc.Name)

		// check whether the PVC is resized
		quantityInSpec, exist := desiredVolumeQuantity[volName]
		if !exist {
			klog.Errorf("Check PVC %q of %q resized failed: not exist in desired volumes", pvcID, cid)
			continue
		}
		currentRequest, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
		if !ok {
			klog.Errorf("Check PVC %q of %q resized failed: storage request is empty", pvcID, cid)
			continue
		}
		currentCapacity, ok := pvc.Status.Capacity[corev1.ResourceStorage]
		if !ok {
			klog.Errorf("Check PVC %q of %q resized failed: storage capacity is empty", pvcID, cid)
			continue
		}

		cmpVal := quantityInSpec.Cmp(currentRequest)
		resizing := currentRequest.Cmp(currentCapacity) != 0
		if cmpVal == 0 {
			if resizing {
				resizingVolumes = append(resizingVolumes, volume)
			} else {
				resizedVolumes = append(resizedVolumes, volume)
			}
			continue
		}

		// check whether the PVC can be resized

		// not support shrink
		if cmpVal < 0 {
			klog.Warningf("Skip to resize PVC %q of %q: storage request cannot be shrunk (%s to %s)",
				pvcID, cid, currentRequest.String(), quantityInSpec.String())
			continue
		}
		// not support default storage class
		if pvc.Spec.StorageClassName == nil {
			klog.Warningf("Skip to resize PVC %q of %q: PVC have no storage class", pvcID, cid)
			continue
		}
		// check whether the storage class support
		if p.deps.StorageClassLister != nil {
			volumeExpansionSupported, err := isVolumeExpansionSupported(p.deps.StorageClassLister, *pvc.Spec.StorageClassName)
			if err != nil {
				return nil, err
			}
			if !volumeExpansionSupported {
				klog.Warningf("Skip to resize PVC %q of %q: storage class %q does not support volume expansion",
					*pvc.Spec.StorageClassName, pvcID, cid)
				continue
			}
		} else {
			klog.V(4).Infof("Storage classes lister is unavailable, skip checking volume expansion support for PVC %q of %q with storage class %s. This may be caused by no relevant permissions",
				pvcID, cid, *pvc.Spec.StorageClassName)
		}

		needResizeVolumes = append(needResizeVolumes, volume)
	}

	return map[volumePhase][]*volume{
		needResize: needResizeVolumes,
		resizing:   resizingVolumes,
		resized:    resizedVolumes,
	}, nil
}

func (p *pvcResizer) resizeVolumesForPod(ctx *componentVolumeContext, pod *corev1.Pod, volumes []*volume) error {
	if err := p.beforeResizeForPod(ctx, pod, volumes); err != nil {
		return err
	}

	desiredVolumeQuantity := ctx.desiredVolumeQuantity
	errs := []error{}

	for _, volume := range volumes {
		volName := volume.name
		pvc := volume.pvc
		pvcID := fmt.Sprintf("%s/%s", pvc.Namespace, pvc.Name)

		currentRequest, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
		if !ok {
			errs = append(errs, fmt.Errorf("resize PVC %s failed: storage request is empty", pvcID))
			continue
		}
		quantityInSpec, exist := desiredVolumeQuantity[volName]
		if !exist {
			errs = append(errs, fmt.Errorf("resize PVC %s failed: not exist in desired volumes", pvcID))
			continue
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
			errs = append(errs, fmt.Errorf("resize PVC %s failed: %s", pvcID, err))
			continue
		}
		_, err = p.deps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Patch(context.TODO(), pvc.Name, types.MergePatchType, mergePatch, metav1.PatchOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("resize PVC %s failed: %s", pvcID, err))
			continue
		}

		klog.Infof("resize PVC %s of %s: storage request is updated from %s to %s",
			pvcID, ctx.ComponentID(), currentRequest.String(), quantityInSpec.String())
	}

	return errutil.NewAggregate(errs)
}

func (p *pvcResizer) beforeResizeForPod(ctx *componentVolumeContext, resizePod *corev1.Pod, volumes []*volume) error {
	logPrefix := fmt.Sprintf("before resizing volumes of Pod %s/%s for %q", resizePod.Namespace, resizePod.Name, ctx.ComponentID())

	switch ctx.status.GetMemberType() {
	case v1alpha1.TiKVMemberType:
		tc, ok := ctx.cluster.(*v1alpha1.TidbCluster)
		if !ok {
			return fmt.Errorf("%s: cluster is not tidb cluster", logPrefix)
		}

		// remove leader eviction ann from resized pods and ensure the store is UP.
		// leader eviction ann of the lastest pod is removed in `endResize`.
		for _, podVolume := range ctx.actualPodVolumes {
			pod := podVolume.pod
			if pod.Name == resizePod.Name {
				break
			}

			updated, err := updateResizeAnnForTiKVPod(p.deps.KubeClientset, false, pod)
			if err != nil {
				return err
			}
			if updated {
				klog.Infof("%s: remove leader eviction annotation from pod %s/%s", logPrefix, pod.Namespace, pod.Name)
			}
			for _, store := range tc.Status.TiKV.Stores {
				if store.PodName == pod.Name && store.State != v1alpha1.TiKVStateUp {
					return fmt.Errorf("%s: store %s of pod %s is not ready", logPrefix, store.ID, store.PodName)
				}
			}
		}

		// add leader eviction ann to the pod and wait the leader count to be 0
		updated, err := updateResizeAnnForTiKVPod(p.deps.KubeClientset, true, resizePod)
		if err != nil {
			return err
		}
		if updated {
			klog.Infof("%s: add leader eviction annotation to pod", logPrefix)
		}
		for _, store := range tc.Status.TiKV.Stores {
			if store.PodName == resizePod.Name {
				if store.LeaderCount == 0 {
					klog.V(4).Infof("%s: leader count of store %s become 0", logPrefix, store.ID)
					return nil
				} else {
					return controller.RequeueErrorf("%s: wait for leader count of store %s to be 0", logPrefix, store.ID)
				}
			}
		}

		return fmt.Errorf("%s: can't find store in tc", logPrefix)
	}

	return nil
}

func (p *pvcResizer) beginResize(ctx *componentVolumeContext) error {
	ctx.status.SetCondition(metav1.Condition{
		Type:    v1alpha1.ComponentVolumeResizing,
		Status:  metav1.ConditionTrue,
		Reason:  "BeginResizing",
		Message: "Set resizing condition to begin resizing",
	})
	klog.Infof("begin resizing for %s: set resizing condition", ctx.ComponentID())
	return controller.RequeueErrorf("set condition before resizing volumes for %s", ctx.ComponentID())
}

func (p *pvcResizer) endResize(ctx *componentVolumeContext) error {
	switch ctx.status.GetMemberType() {
	case v1alpha1.TiKVMemberType:
		// ensure all eviction annotations are removed
		for _, podVolume := range ctx.actualPodVolumes {
			updated, err := updateResizeAnnForTiKVPod(p.deps.KubeClientset, false, podVolume.pod)
			if err != nil {
				return fmt.Errorf("remove leader eviction annotation from pod failed: %s", err)
			}
			if updated {
				klog.Infof("end resizing for %s: remove leader eviction annotation from pod %s/%s",
					ctx.ComponentID(), podVolume.pod.Namespace, podVolume.pod.Name)
			}
		}
	}

	ctx.status.SetCondition(metav1.Condition{
		Type:    v1alpha1.ComponentVolumeResizing,
		Status:  metav1.ConditionFalse,
		Reason:  "EndResizing",
		Message: "All volumes are resized",
	})
	klog.Infof("end resizing for %s: update resizing condition", ctx.ComponentID())
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
		volumes := []*volume{}

		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				pvc, err := findPVC(vol.PersistentVolumeClaim.ClaimName)
				if err != nil {
					klog.Warningf("Failed to find PVC %s of Pod %s/%s, maybe some labels are lost",
						vol.PersistentVolumeClaim.ClaimName, pod.Namespace, pod.Name)
					continue
				}
				volumes = append(volumes, &volume{
					name: v1alpha1.StorageVolumeName(vol.Name),
					pvc:  pvc.DeepCopy(),
				})
			}
		}

		result = append(result, &podVolumeContext{
			pod:     pod.DeepCopy(),
			volumes: volumes,
		})
	}

	// sort by pod name to ensure the order is stable
	sort.Slice(result, func(i, j int) bool {
		name1, name2 := result[i].pod.Name, result[j].pod.Name
		if len(name1) != len(name2) {
			return len(name1) < len(name2)
		}
		return name1 < name2
	})

	return result, nil
}

func isVolumeExpansionSupported(lister storagelister.StorageClassLister, storageClassName string) (bool, error) {
	sc, err := lister.Get(storageClassName)
	if err != nil {
		return false, err
	}
	if sc.AllowVolumeExpansion == nil {
		return false, nil
	}
	return *sc.AllowVolumeExpansion, nil
}

func updateResizeAnnForTiKVPod(client kubernetes.Interface, need bool, pod *corev1.Pod) (bool /*updated*/, error) {
	_, exist := pod.Annotations[v1alpha1.EvictLeaderAnnKeyForResize]

	if need && !exist {
		if pod.Annotations == nil {
			pod.Annotations = map[string]string{}
		}
		pod.Annotations[v1alpha1.EvictLeaderAnnKeyForResize] = v1alpha1.EvictLeaderValueNone
		_, err := client.CoreV1().Pods(pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{})
		if err != nil {
			return false, fmt.Errorf("add leader eviction annotation to pod %s/%s failed: %s", pod.Namespace, pod.Name, err)
		}
		return true, nil
	}
	if !need && exist {
		delete(pod.Annotations, v1alpha1.EvictLeaderAnnKeyForResize)
		_, err := client.CoreV1().Pods(pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{})
		if err != nil {
			return false, fmt.Errorf("remove leader eviction annotation from pod %s/%s failed: %s", pod.Namespace, pod.Name, err)
		}
		return true, nil
	}

	return false, nil
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
