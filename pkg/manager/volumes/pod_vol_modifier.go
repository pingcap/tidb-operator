// Copyright 2022 PingCAP, Inc.
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

package volumes

import (
	"context"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errutil "k8s.io/apimachinery/pkg/util/errors"
	klog "k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes/delegation"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes/delegation/aws"
)

type PodVolumeModifier interface {
	GetDesiredVolumes(tc *v1alpha1.TidbCluster, mt v1alpha1.MemberType) ([]DesiredVolume, error)
	GetActualVolumes(pod *corev1.Pod, vs []DesiredVolume) ([]ActualVolume, error)

	ShouldModify(actual []ActualVolume) bool
	Modify(actual []ActualVolume) error
}

type ActualVolume struct {
	Desired *DesiredVolume
	PVC     *corev1.PersistentVolumeClaim
	Phase   VolumePhase
	// it may be nil if there is no permission to get pvc
	PV *corev1.PersistentVolume
	// it may be nil if there is no permission to get storage class
	StorageClass *storagev1.StorageClass
}

// get storage class name from current pvc
func (v *ActualVolume) GetStorageClassName() string {
	return getStorageClassNameFromPVC(v.PVC)
}

func (v *ActualVolume) GetStorageSize() resource.Quantity {
	return getStorageSize(v.PVC.Status.Capacity)
}

type podVolModifier struct {
	deps      *controller.Dependencies
	utils     *volCompareUtils
	modifiers map[string]delegation.VolumeModifier
}

func NewPodVolumeModifier(deps *controller.Dependencies) PodVolumeModifier {
	m := &podVolModifier{
		deps:      deps,
		utils:     newVolCompareUtils(deps),
		modifiers: map[string]delegation.VolumeModifier{},
	}
	if features.DefaultFeatureGate.Enabled(features.VolumeModifying) {
		m.modifiers["ebs.csi.aws.com"] = aws.NewEBSModifier(deps.AWSConfig)
	}

	return m
}

func (p *podVolModifier) ShouldModify(actual []ActualVolume) bool {
	for i := range actual {
		vol := &actual[i]
		switch vol.Phase {
		case VolumePhasePreparing, VolumePhaseModifying:
			return true
		}
	}

	return false
}

func (p *podVolModifier) Modify(actual []ActualVolume) error {
	ctx := context.TODO()

	errs := []error{}

	for i := range actual {
		vol := &actual[i]
		klog.Infof("try to sync volume %s/%s, phase: %s", vol.PVC.Namespace, vol.PVC.Name, vol.Phase)

		switch vol.Phase {
		case VolumePhasePreparing:
			if err := p.modifyPVCAnnoSpec(ctx, vol, false); err != nil {
				errs = append(errs, err)
				continue
			}

			fallthrough
		case VolumePhaseModifying:
			wait, err := p.modifyVolume(ctx, vol)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if wait {
				errs = append(errs, fmt.Errorf("wait for volume modification completed"))
				continue
			}
			// try to resize fs
			synced, err := p.syncPVCSize(ctx, vol)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if !synced {
				errs = append(errs, fmt.Errorf("wait for fs resize completed"))
				continue
			}
			if err := p.modifyPVCAnnoStatus(ctx, vol); err != nil {
				errs = append(errs, err)
			}
		case VolumePhasePending, VolumePhaseModified, VolumePhaseCannotModify:
		}

	}

	return errutil.NewAggregate(errs)
}

// TODO: it should be refactored
func (p *podVolModifier) GetDesiredVolumes(tc *v1alpha1.TidbCluster, mt v1alpha1.MemberType) ([]DesiredVolume, error) {
	// pass-thru to expose from utils directly.
	return p.utils.GetDesiredVolumes(tc, mt)
}

func (p *podVolModifier) GetActualVolumes(pod *corev1.Pod, vs []DesiredVolume) ([]ActualVolume, error) {
	vols := []ActualVolume{}

	for i := range pod.Spec.Volumes {
		vol := &pod.Spec.Volumes[i]
		actual, err := p.NewActualVolumeOfPod(vs, pod.Namespace, vol)
		if err != nil {
			return nil, err
		}
		if actual == nil {
			continue
		}

		vols = append(vols, *actual)
	}

	return vols, nil
}

func (p *podVolModifier) NewActualVolumeOfPod(vs []DesiredVolume, ns string, vol *corev1.Volume) (*ActualVolume, error) {
	pvc, err := p.utils.getPVC(ns, vol)
	if err != nil {
		return nil, err
	}
	if pvc == nil {
		return nil, nil
	}

	// TODO: fix the case when pvc is pending
	pv, err := p.utils.getBoundPVFromPVC(pvc)
	if err != nil {
		return nil, err
	}

	sc, err := p.utils.getStorageClassFromPVC(pvc)
	if err != nil {
		return nil, err
	}

	// no desired volume, it may be a volume which is unmanaged by operator
	desired := getDesiredVolumeByName(vs, v1alpha1.StorageVolumeName(vol.Name))
	if desired == nil {
		return nil, nil
	}

	actual := ActualVolume{
		Desired:      desired,
		PVC:          pvc,
		PV:           pv,
		StorageClass: sc,
	}

	phase := p.getVolumePhase(&actual)
	actual.Phase = phase

	return &actual, nil
}

func upgradeRevision(pvc *corev1.PersistentVolumeClaim) {
	rev := 1
	str, ok := pvc.Annotations[annoKeyPVCSpecRevision]
	if ok {
		oldRev, err := strconv.Atoi(str)
		if err != nil {
			klog.Warningf("revision format err: %v, reset to 0", err)
			oldRev = 0
		}
		rev = oldRev + 1
	}

	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}

	pvc.Annotations[annoKeyPVCSpecRevision] = strconv.Itoa(rev)
}

func isPVCSpecMatched(pvc *corev1.PersistentVolumeClaim, scName string, size resource.Quantity) bool {
	isChanged := false
	oldSc := pvc.Annotations[annoKeyPVCSpecStorageClass]
	if scName != "" && oldSc != scName {
		isChanged = true
	}

	oldSize, ok := pvc.Annotations[annoKeyPVCSpecStorageSize]
	if !ok {
		quantity := getStorageSize(pvc.Spec.Resources.Requests)
		oldSize = quantity.String()
	}
	if oldSize != size.String() {
		isChanged = true
	}

	return isChanged
}

func snapshotStorageClassAndSize(pvc *corev1.PersistentVolumeClaim, scName string, size resource.Quantity) bool {
	isChanged := isPVCSpecMatched(pvc, scName, size)

	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}

	if scName != "" {
		pvc.Annotations[annoKeyPVCSpecStorageClass] = scName
	}
	pvc.Annotations[annoKeyPVCSpecStorageSize] = size.String()

	return isChanged
}

func setLastTransitionTimestamp(pvc *corev1.PersistentVolumeClaim) {
	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}

	pvc.Annotations[annoKeyPVCLastTransitionTimestamp] = metav1.Now().Format(time.RFC3339)
}

// upgrade revision and snapshot the expected storageclass and size of volume
func (p *podVolModifier) modifyPVCAnnoSpec(ctx context.Context, vol *ActualVolume, shouldEvict bool) error {
	pvc := vol.PVC.DeepCopy()

	size := vol.Desired.Size
	scName := vol.Desired.GetStorageClassName()

	isChanged := snapshotStorageClassAndSize(pvc, scName, size)
	if isChanged {
		upgradeRevision(pvc)
	}

	if !shouldEvict {
		setLastTransitionTimestamp(pvc)
	}

	updated, err := p.deps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(ctx, pvc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	vol.PVC = updated

	return nil
}

func (p *podVolModifier) syncPVCSize(ctx context.Context, vol *ActualVolume) (bool, error) {
	capacity := vol.PVC.Status.Capacity.Storage()
	requestSize := vol.PVC.Spec.Resources.Requests.Storage()
	if requestSize.Cmp(vol.Desired.Size) == 0 && capacity.Cmp(vol.Desired.Size) == 0 {
		return true, nil

	}

	if requestSize.Cmp(vol.Desired.Size) == 0 {
		return false, nil
	}

	pvc := vol.PVC.DeepCopy()
	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = vol.Desired.Size
	updated, err := p.deps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(ctx, pvc, metav1.UpdateOptions{})
	if err != nil {
		return false, err
	}

	vol.PVC = updated

	return false, nil
}

func (p *podVolModifier) modifyPVCAnnoStatus(ctx context.Context, vol *ActualVolume) error {
	pvc := vol.PVC.DeepCopy()

	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}

	pvc.Annotations[annoKeyPVCStatusRevision] = pvc.Annotations[annoKeyPVCSpecRevision]
	if scName := pvc.Annotations[annoKeyPVCSpecStorageClass]; scName != "" {
		pvc.Annotations[annoKeyPVCStatusStorageClass] = scName
	}
	pvc.Annotations[annoKeyPVCStatusStorageSize] = pvc.Annotations[annoKeyPVCSpecStorageSize]

	updated, err := p.deps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(ctx, pvc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	vol.PVC = updated

	return nil
}

func (p *podVolModifier) modifyVolume(ctx context.Context, vol *ActualVolume) (bool, error) {
	m := p.getVolumeModifier(vol.StorageClass, vol.Desired.StorageClass)
	if m == nil {
		// skip modifying volume by delegation.VolumeModifier
		return false, nil
	}

	pvc := vol.PVC.DeepCopy()
	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = vol.Desired.Size

	return m.ModifyVolume(ctx, pvc, vol.PV, vol.Desired.StorageClass)
}

func (p *podVolModifier) getVolumeModifier(actualSc, desiredSc *storagev1.StorageClass) delegation.VolumeModifier {
	if actualSc == nil || desiredSc == nil {
		return nil
	}
	// sc is not changed
	if actualSc.Name == desiredSc.Name {
		return nil
	}

	return p.modifiers[desiredSc.Provisioner]
}

func isLeaderEvictedOrTimeout(tc *v1alpha1.TidbCluster, pod *corev1.Pod) bool {
	if isLeaderEvictionFinished(tc, pod) {
		return false
	}
	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == pod.Name {
			if store.LeaderCount == 0 {
				klog.V(4).Infof("leader count of store %s become 0", store.ID)
				return true
			}

			if status, exist := tc.Status.TiKV.EvictLeader[pod.Name]; exist && !status.BeginTime.IsZero() {
				timeout := tc.TiKVEvictLeaderTimeout()
				if time.Since(status.BeginTime.Time) > timeout {
					klog.Infof("leader eviction begins at %q but timeout (threshold: %v)", status.BeginTime.Format(time.RFC3339), timeout)
					return true
				}
			}

			return false
		}
	}

	return false
}

func getStorageClassNameFromPVC(pvc *corev1.PersistentVolumeClaim) string {
	sc := ignoreNil(pvc.Spec.StorageClassName)

	scAnno, ok := pvc.Annotations[annoKeyPVCStatusStorageClass]
	if ok && scAnno != "" {
		sc = scAnno
	}

	return sc
}
