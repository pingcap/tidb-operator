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
	storagelister "k8s.io/client-go/listers/storage/v1"
	klog "k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes/delegation"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes/delegation/aws"
)

type PodVolumeModifier interface {
	GetDesiredVolumes(tc *v1alpha1.TidbCluster, mt v1alpha1.MemberType) ([]DesiredVolume, error)
	GetActualVolumes(pod *corev1.Pod, vs []DesiredVolume) ([]ActualVolume, error)

	ShouldModify(actual []ActualVolume) bool
	Modify(actual []ActualVolume) error
}

type DesiredVolume struct {
	Name         v1alpha1.StorageVolumeName
	Size         resource.Quantity
	StorageClass *storagev1.StorageClass
}

func (v *DesiredVolume) GetStorageClassName() string {
	if v.StorageClass == nil {
		return ""
	}
	return v.StorageClass.Name
}

func (v *DesiredVolume) GetStorageSize() resource.Quantity {
	return v.Size
}

type ActualVolume struct {
	Desired      *DesiredVolume
	PVC          *corev1.PersistentVolumeClaim
	PV           *corev1.PersistentVolume
	StorageClass *storagev1.StorageClass
	Phase        VolumePhase
}

func (v *ActualVolume) GetStorageClassName() string {
	sc := ignoreNil(v.PVC.Spec.StorageClassName)

	scAnno, ok := v.PVC.Annotations[annoKeyPVCStatusStorageClass]
	if ok {
		sc = scAnno
	}

	return sc
}

func (v *ActualVolume) GetStorageSize() resource.Quantity {
	return getStorageSize(v.PVC.Status.Capacity)
}

type podVolModifier struct {
	deps *controller.Dependencies

	modifiers map[string]delegation.VolumeModifier
}

func NewPodVolumeModifier(deps *controller.Dependencies) PodVolumeModifier {
	return &podVolModifier{
		deps: deps,
		modifiers: map[string]delegation.VolumeModifier{
			"ebs.csi.aws.com": aws.NewEBSModifier(deps.AWSConfig),
		},
	}
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
	desiredVolumes := []DesiredVolume{}
	scLister := p.deps.StorageClassLister

	storageVolumes := []v1alpha1.StorageVolume{}
	var defaultSc *storagev1.StorageClass
	switch mt {
	case v1alpha1.PDMemberType:
		sc, err := getStorageClass(tc.Spec.PD.StorageClassName, scLister)
		if err != nil {
			return nil, err
		}
		defaultSc = sc
		d := DesiredVolume{
			Name:         v1alpha1.GetStorageVolumeName("", mt),
			Size:         getStorageSize(tc.Spec.PD.Requests),
			StorageClass: sc,
		}
		desiredVolumes = append(desiredVolumes, d)

		storageVolumes = tc.Spec.PD.StorageVolumes

	case v1alpha1.TiDBMemberType:
		sc, err := getStorageClass(tc.Spec.TiDB.StorageClassName, scLister)
		if err != nil {
			return nil, err
		}
		defaultSc = sc
		storageVolumes = tc.Spec.TiDB.StorageVolumes

	case v1alpha1.TiKVMemberType:
		sc, err := getStorageClass(tc.Spec.TiKV.StorageClassName, scLister)
		if err != nil {
			return nil, err
		}
		defaultSc = sc
		d := DesiredVolume{
			Name:         v1alpha1.GetStorageVolumeName("", mt),
			Size:         getStorageSize(tc.Spec.TiKV.Requests),
			StorageClass: sc,
		}
		desiredVolumes = append(desiredVolumes, d)

		storageVolumes = tc.Spec.TiKV.StorageVolumes

	case v1alpha1.TiFlashMemberType:
		for i, claim := range tc.Spec.TiFlash.StorageClaims {
			sc, err := getStorageClass(claim.StorageClassName, scLister)
			if err != nil {
				return nil, err
			}
			d := DesiredVolume{
				Name:         v1alpha1.GetStorageVolumeNameForTiFlash(i),
				Size:         getStorageSize(claim.Resources.Requests),
				StorageClass: sc,
			}
			desiredVolumes = append(desiredVolumes, d)
		}

	case v1alpha1.TiCDCMemberType:
		sc, err := getStorageClass(tc.Spec.TiCDC.StorageClassName, scLister)
		if err != nil {
			return nil, err
		}
		defaultSc = sc
		storageVolumes = tc.Spec.TiCDC.StorageVolumes

	case v1alpha1.PumpMemberType:
		sc, err := getStorageClass(tc.Spec.Pump.StorageClassName, scLister)
		if err != nil {
			return nil, err
		}
		defaultSc = sc
		d := DesiredVolume{
			Name:         v1alpha1.GetStorageVolumeName("", mt),
			Size:         getStorageSize(tc.Spec.Pump.Requests),
			StorageClass: sc,
		}
		desiredVolumes = append(desiredVolumes, d)
	default:
		return nil, fmt.Errorf("unsupported member type %s", mt)
	}

	for _, sv := range storageVolumes {
		if quantity, err := resource.ParseQuantity(sv.StorageSize); err == nil {
			sc, err := getStorageClass(sv.StorageClassName, scLister)
			if err != nil {
				return nil, err
			}
			if sc == nil {
				sc = defaultSc
			}
			d := DesiredVolume{
				Name:         v1alpha1.GetStorageVolumeName(sv.Name, mt),
				Size:         quantity,
				StorageClass: sc,
			}

			desiredVolumes = append(desiredVolumes, d)

		} else {
			klog.Warningf("StorageVolume %q in %s/%s .spec.%s is invalid", sv.Name, tc.GetNamespace(), tc.GetName(), mt)
		}
	}

	return desiredVolumes, nil
}

func getStorageClass(name *string, scLister storagelister.StorageClassLister) (*storagev1.StorageClass, error) {
	if name == nil {
		return nil, nil
	}
	return scLister.Get(*name)
}

func getDesiredVolumeByName(vs []DesiredVolume, name v1alpha1.StorageVolumeName) *DesiredVolume {
	for i := range vs {
		v := &vs[i]
		if v.Name == name {
			return v
		}
	}

	return nil
}

func (p *podVolModifier) getBoundPVFromPVC(pvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolume, error) {
	name := pvc.Spec.VolumeName

	return p.deps.PVLister.Get(name)
}

func (p *podVolModifier) getStorageClassFromPVC(pvc *corev1.PersistentVolumeClaim) (*storagev1.StorageClass, error) {
	sc := ignoreNil(pvc.Spec.StorageClassName)

	scAnno, ok := pvc.Annotations[annoKeyPVCStatusStorageClass]
	if ok {
		sc = scAnno
	}

	if sc == "" {
		return nil, nil
	}

	return p.deps.StorageClassLister.Get(sc)
}

func (p *podVolModifier) getPVC(ns string, vol *corev1.Volume) (*corev1.PersistentVolumeClaim, error) {
	if vol.PersistentVolumeClaim == nil {
		return nil, nil
	}

	pvc, err := p.deps.PVCLister.PersistentVolumeClaims(ns).Get(vol.PersistentVolumeClaim.ClaimName)
	if err != nil {
		return nil, err
	}

	return pvc, nil
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
	pvc, err := p.getPVC(ns, vol)
	if err != nil {
		return nil, err
	}
	if pvc == nil {
		return nil, nil
	}

	// TODO: fix the case when pvc is pending
	pv, err := p.getBoundPVFromPVC(pvc)
	if err != nil {
		return nil, err
	}

	sc, err := p.getStorageClassFromPVC(pvc)
	if err != nil {
		return nil, err
	}

	desired := getDesiredVolumeByName(vs, v1alpha1.StorageVolumeName(vol.Name))

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
	if oldSc != scName {
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

	pvc.Annotations[annoKeyPVCSpecStorageClass] = scName
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
	scName := ""
	if vol.Desired.StorageClass != nil {
		scName = vol.Desired.StorageClass.Name
	}

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
	pvc.Annotations[annoKeyPVCStatusStorageClass] = pvc.Annotations[annoKeyPVCSpecStorageClass]
	pvc.Annotations[annoKeyPVCStatusStorageSize] = pvc.Annotations[annoKeyPVCSpecStorageSize]

	updated, err := p.deps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(ctx, pvc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	vol.PVC = updated

	return nil
}

func (p *podVolModifier) modifyVolume(ctx context.Context, vol *ActualVolume) (bool, error) {
	m := p.getVolumeModifier(vol.Desired.StorageClass)
	if m == nil {
		// skip modifying volume by delegation.VolumeModifier
		return false, nil
	}

	pvc := vol.PVC.DeepCopy()
	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = vol.Desired.Size

	return m.ModifyVolume(ctx, pvc, vol.PV, vol.Desired.StorageClass)
}

func (p *podVolModifier) getVolumeModifier(sc *storagev1.StorageClass) delegation.VolumeModifier {
	return p.modifiers[sc.Provisioner]
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
