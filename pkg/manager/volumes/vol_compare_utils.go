// Copyright 2023 PingCAP, Inc.
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
	"sort"

	"github.com/pingcap/errors"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	storagelister "k8s.io/client-go/listers/storage/v1"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type DesiredVolume struct {
	Name v1alpha1.StorageVolumeName
	Size resource.Quantity
	// it may be nil if there is no permission to get storage class
	StorageClass *storagev1.StorageClass
	// it is sc name specified by user
	// the sc may not exist
	StorageClassName *string
}

// get storage class name from tc
// it may return empty because sc is unset or no permission to verify the existence of sc
func (v *DesiredVolume) GetStorageClassName() string {
	if v.StorageClassName == nil {
		return ""
	}
	return *v.StorageClassName
}

func (v *DesiredVolume) GetStorageSize() resource.Quantity {
	return v.Size
}

type volCompareUtils struct {
	deps *controller.Dependencies
	sf   *selectorFactory
}

func newVolCompareUtils(deps *controller.Dependencies) *volCompareUtils {
	return &volCompareUtils{
		deps: deps,
		sf:   MustNewSelectorFactory(),
	}
}

type componentVolumeContext struct {
	context.Context
	tc     *v1alpha1.TidbCluster
	status v1alpha1.ComponentStatus

	shouldEvict bool

	pods []*corev1.Pod
	sts  *appsv1.StatefulSet

	desiredVolumes []DesiredVolume
}

func (c *componentVolumeContext) ComponentID() string {
	return fmt.Sprintf("%s/%s:%s", c.tc.GetNamespace(), c.tc.GetName(), c.status.MemberType())
}

func (u *volCompareUtils) BuildContextForTC(tc *v1alpha1.TidbCluster, status v1alpha1.ComponentStatus) (*componentVolumeContext, error) {
	comp := status.MemberType()

	ctx := &componentVolumeContext{
		Context: context.TODO(),
		tc:      tc,
		status:  status,
	}

	vs, err := u.GetDesiredVolumes(tc, comp)
	if err != nil {
		return nil, err
	}
	ctx.desiredVolumes = vs

	sts, err := u.getStsOfComponent(tc, comp)
	if err != nil {
		return nil, err
	}

	pods, err := u.getPodsOfComponent(tc, comp)
	if err != nil {
		return nil, err
	}

	ctx.pods = pods
	ctx.sts = sts
	ctx.shouldEvict = comp == v1alpha1.TiKVMemberType

	return ctx, nil
}

func (u *volCompareUtils) getPodsOfComponent(tc *v1alpha1.TidbCluster, mt v1alpha1.MemberType) ([]*corev1.Pod, error) {
	selector, err := u.sf.NewSelector(tc.GetInstanceName(), mt)
	if err != nil {
		return nil, err
	}

	ns := tc.GetNamespace()

	pods, err := u.deps.PodLister.Pods(ns).List(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to list Pods: %w", err)
	}

	sort.Slice(pods, func(i, k int) bool {
		a, b := pods[i].Name, pods[k].Name
		if len(a) != len(b) {
			return len(a) < len(b)
		}
		return a < b
	})

	return pods, nil
}

func (u *volCompareUtils) getStsOfComponent(cluster v1alpha1.Cluster, mt v1alpha1.MemberType) (*appsv1.StatefulSet, error) {
	ns := cluster.GetNamespace()
	stsName := controller.MemberName(cluster.GetName(), mt)

	sts, err := u.deps.StatefulSetLister.StatefulSets(ns).Get(stsName)
	if err != nil {
		return nil, fmt.Errorf("get sts %s/%s failed: %w", ns, stsName, err)
	}

	return sts, nil
}

func (u *volCompareUtils) IsStatefulSetSynced(ctx *componentVolumeContext, sts *appsv1.StatefulSet, errOnNewVol bool) (bool, error) {
	var newVolErr error = nil
	for _, volTemplate := range sts.Spec.VolumeClaimTemplates {
		volName := v1alpha1.StorageVolumeName(volTemplate.Name)
		size := getStorageSize(volTemplate.Spec.Resources.Requests)
		desired := getDesiredVolumeByName(ctx.desiredVolumes, volName)
		if desired == nil {
			// TODO: verify behaviour is correctly intended, this was previously being ignored. Now explicit error in
			// pvc modifier..
			errStr := fmt.Sprintf("volume %s in sts for cluster %s does not exist in desired volumes", volName, ctx.ComponentID())
			if errOnNewVol {
				newVolErr = errors.New(errStr)
			} else {
				klog.Warningf(errStr)
			}
			return false, newVolErr
		}
		if size.Cmp(desired.Size) != 0 {
			return false, nil
		}
		scName := volTemplate.Spec.StorageClassName
		if !isStorageClassMatched(desired.StorageClass, ignoreNil(scName)) {
			return false, nil
		}
	}
	// TODO: verify new behaviour is fine with pvc_modifier, previously number of volumes not checked.
	if len(sts.Spec.VolumeClaimTemplates) != len(ctx.desiredVolumes) {
		errStr := fmt.Sprintf("Number of volumes mismatch desired %d vs sts %d for cluster %s", len(ctx.desiredVolumes), len(sts.Spec.VolumeClaimTemplates), ctx.ComponentID())
		if errOnNewVol {
			newVolErr = errors.New(errStr)
		} else {
			klog.Warningf(errStr)
		}
		return false, newVolErr
	}

	return true, nil
}

func (u *volCompareUtils) IsPodSyncedForReplacement(ctx *componentVolumeContext, pod *corev1.Pod) (bool, error) {
	// Does not check underlying StorageClass for modifications, only matches PVC specs for use by pvc replacer.
	ns := pod.Namespace
	podPvcCount := 0
	for i := range pod.Spec.Volumes {
		vol := &pod.Spec.Volumes[i]
		pvc, err := u.getPVC(ns, vol)
		if err != nil {
			return false, err
		}
		if pvc == nil {
			continue // Not a PVC don't compare.
		}
		podPvcCount++
		desired := getDesiredVolumeByName(ctx.desiredVolumes, v1alpha1.StorageVolumeName(vol.Name))
		if desired == nil {
			// Extra un-desired volume in pod, needs to be removed.
			return false, nil
		}
		desiredScName := desired.GetStorageClassName()
		// If desired sc name is unspecified , any sc used by pvc is okay. (Default storage class).
		if desiredScName != "" && ignoreNil(pvc.Spec.StorageClassName) != desiredScName {
			// StorageClass change. Note: not checking underlying storage class of PV.
			return false, nil
		}
		pvcQuantity := getStorageSize(pvc.Spec.Resources.Requests)
		if pvcQuantity.Cmp(desired.Size) != 0 {
			// Sizes don't match.
			return false, nil
		}
	}
	if podPvcCount != len(ctx.desiredVolumes) {
		// Mismatched number of volumes between desired & pod.
		return false, nil
	}
	return true, nil
}

// TODO: it should be refactored
func (u *volCompareUtils) GetDesiredVolumes(tc *v1alpha1.TidbCluster, mt v1alpha1.MemberType) ([]DesiredVolume, error) {
	desiredVolumes := []DesiredVolume{}
	scLister := u.deps.StorageClassLister

	storageVolumes := []v1alpha1.StorageVolume{}
	var defaultScName *string
	switch mt {
	case v1alpha1.TiProxyMemberType:
		defaultScName = tc.Spec.TiProxy.StorageClassName
		d := DesiredVolume{
			Name:             v1alpha1.GetStorageVolumeName("", mt),
			Size:             getStorageSize(tc.Spec.TiProxy.Requests),
			StorageClassName: defaultScName,
		}
		desiredVolumes = append(desiredVolumes, d)

		storageVolumes = tc.Spec.TiProxy.StorageVolumes
	case v1alpha1.PDMemberType:
		defaultScName = tc.Spec.PD.StorageClassName
		d := DesiredVolume{
			Name:             v1alpha1.GetStorageVolumeName("", mt),
			Size:             getStorageSize(tc.Spec.PD.Requests),
			StorageClassName: defaultScName,
		}
		desiredVolumes = append(desiredVolumes, d)

		storageVolumes = tc.Spec.PD.StorageVolumes

	case v1alpha1.TiDBMemberType:
		defaultScName = tc.Spec.TiDB.StorageClassName
		storageVolumes = tc.Spec.TiDB.StorageVolumes

	case v1alpha1.TiKVMemberType:
		defaultScName = tc.Spec.TiKV.StorageClassName
		d := DesiredVolume{
			Name:             v1alpha1.GetStorageVolumeName("", mt),
			Size:             getStorageSize(tc.Spec.TiKV.Requests),
			StorageClassName: defaultScName,
		}
		desiredVolumes = append(desiredVolumes, d)

		storageVolumes = tc.Spec.TiKV.StorageVolumes

	case v1alpha1.TiFlashMemberType:
		for i, claim := range tc.Spec.TiFlash.StorageClaims {
			d := DesiredVolume{
				Name:             v1alpha1.GetStorageVolumeNameForTiFlash(i),
				Size:             getStorageSize(claim.Resources.Requests),
				StorageClassName: claim.StorageClassName,
			}
			desiredVolumes = append(desiredVolumes, d)
		}

	case v1alpha1.TiCDCMemberType:
		defaultScName = tc.Spec.TiCDC.StorageClassName
		storageVolumes = tc.Spec.TiCDC.StorageVolumes

	case v1alpha1.PumpMemberType:
		defaultScName = tc.Spec.Pump.StorageClassName
		d := DesiredVolume{
			Name:             v1alpha1.GetStorageVolumeName("", mt),
			Size:             getStorageSize(tc.Spec.Pump.Requests),
			StorageClassName: defaultScName,
		}
		desiredVolumes = append(desiredVolumes, d)
	default:
		return nil, fmt.Errorf("unsupported member type %s", mt)
	}

	for _, sv := range storageVolumes {
		if quantity, err := resource.ParseQuantity(sv.StorageSize); err == nil {
			d := DesiredVolume{
				Name:             v1alpha1.GetStorageVolumeName(sv.Name, mt),
				Size:             quantity,
				StorageClassName: sv.StorageClassName,
			}
			if d.StorageClassName == nil {
				d.StorageClassName = defaultScName
			}

			desiredVolumes = append(desiredVolumes, d)

		} else {
			klog.Warningf("StorageVolume %q in %s/%s .spec.%s is invalid", sv.Name, tc.GetNamespace(), tc.GetName(), mt)
		}
	}

	if scLister != nil {
		for i := range desiredVolumes {
			if desiredVolumes[i].StorageClassName != nil {
				sc, err := getStorageClass(desiredVolumes[i].StorageClassName, scLister)
				if err != nil {
					return nil, fmt.Errorf("cannot get sc %s", *desiredVolumes[i].StorageClassName)
				}
				desiredVolumes[i].StorageClass = sc
			}
		}
	}

	return desiredVolumes, nil
}

func isStorageClassMatched(sc *storagev1.StorageClass, scName string) bool {
	if sc == nil {
		// cannot get sc or sc is unset
		return true
	}
	if sc.Name == scName {
		return true
	}

	return false
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

func getStorageSize(r corev1.ResourceList) resource.Quantity {
	return r[corev1.ResourceStorage]
}

func ignoreNil(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (u *volCompareUtils) getPVC(ns string, vol *corev1.Volume) (*corev1.PersistentVolumeClaim, error) {
	if vol.PersistentVolumeClaim == nil {
		return nil, nil
	}

	pvc, err := u.deps.PVCLister.PersistentVolumeClaims(ns).Get(vol.PersistentVolumeClaim.ClaimName)
	if err != nil {
		return nil, err
	}

	return pvc, nil
}
