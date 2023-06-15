// Copyright 2021 PingCAP, Inc.
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

package utils

import (
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/features"
	apiequality "k8s.io/apimachinery/pkg/api/equality"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"

	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

const (
	// LastAppliedConfigAnnotation is annotation key of last applied configuration
	LastAppliedConfigAnnotation = "pingcap.com/last-applied-configuration"
)

// StatefulSetIsUpgrading confirms whether the statefulSet is upgrading phase
func StatefulSetIsUpgrading(set *apps.StatefulSet) bool {
	if set.Status.CurrentRevision != set.Status.UpdateRevision {
		return true
	}
	if set.Generation > set.Status.ObservedGeneration && *set.Spec.Replicas == set.Status.Replicas {
		return true
	}
	return false
}

// SetStatefulSetLastAppliedConfigAnnotation set last applied config to Statefulset's annotation
func SetStatefulSetLastAppliedConfigAnnotation(set *apps.StatefulSet) error {
	setApply, err := util.Encode(set.Spec)
	if err != nil {
		return err
	}
	if set.Annotations == nil {
		set.Annotations = map[string]string{}
	}
	set.Annotations[LastAppliedConfigAnnotation] = setApply
	return nil
}

func notExistMount(sts *apps.StatefulSet, oldSTS *apps.StatefulSet) map[string]corev1.VolumeMount {
	volumes := make(map[string]struct{})
	for _, v := range sts.Spec.Template.Spec.Volumes {
		volumes[v.Name] = struct{}{}
	}
	// Note VolumeClaimTemplates DO NOT support update and we will ignore it when update
	// the STS.
	for _, pvc := range oldSTS.Spec.VolumeClaimTemplates {
		volumes[pvc.Name] = struct{}{}
	}

	mounts := make(map[string]corev1.VolumeMount)

	for _, c := range sts.Spec.Template.Spec.Containers {
		for _, m := range c.VolumeMounts {
			_, ok := volumes[m.Name]
			if ok {
				continue
			}

			mounts[m.Name] = m
		}
	}

	return mounts
}

func claimsSame(sts *apps.StatefulSet, oldSTS *apps.StatefulSet) error {
	// Check if VolumeClaimTemplates are different either in whole new/removed claims or different spec.
	newLen := len(sts.Spec.VolumeClaimTemplates)
	oldLen := len(oldSTS.Spec.VolumeClaimTemplates)
	if newLen != oldLen {
		return fmt.Errorf("different size of volumeClaimTemplates in old (%d) vs new(%d)", oldLen, newLen)
	}
	oldClaims := make(map[string]corev1.PersistentVolumeClaim)
	for _, pvc := range oldSTS.Spec.VolumeClaimTemplates {
		oldClaims[pvc.Name] = pvc
	}
	for _, pvc := range sts.Spec.VolumeClaimTemplates {
		oldPvc, ok := oldClaims[pvc.Name]
		if ok {
			// only diff storage class & resources, other fields may have spurious diffs due to k8s defaults.
			oldStorageClass, newStorageClass := "", ""
			if oldPvc.Spec.StorageClassName != nil {
				oldStorageClass = *oldPvc.Spec.StorageClassName
			}
			if pvc.Spec.StorageClassName != nil {
				newStorageClass = *pvc.Spec.StorageClassName
			}
			if oldStorageClass != newStorageClass {
				return fmt.Errorf("volume claim %s has new storageclass (%s) vs old (%s)", pvc.Name, newStorageClass, oldStorageClass)
			}
			if !apiequality.Semantic.DeepEqual(oldPvc.Spec.Resources, pvc.Spec.Resources) {
				return fmt.Errorf("volume claim %s has new different resource requierments", pvc.Name)
			}
		} else {
			return fmt.Errorf("new volume claim %s not present in old statefulset", pvc.Name)
		}
	}
	return nil
}

func UpdateStatefulSetWithPrecheck(
	deps *controller.Dependencies,
	tc *v1alpha1.TidbCluster,
	reason string,
	newTiDBSet *apps.StatefulSet,
	oldTiDBSet *apps.StatefulSet,
) error {
	// If the StatefulSet spec contains a volume mount that does not have a matched volume,
	// do not update the StatefulSet, otherwise, it will trigger a rolling update and the
	// rolling update will hang because the new Pod cannot be created after deleting the old one.
	// Emit event and return error here to let the user be aware of this and fix it in the spec
	notExistMount := notExistMount(newTiDBSet, oldTiDBSet)
	if len(notExistMount) > 0 {
		deps.Recorder.Eventf(tc, corev1.EventTypeWarning, reason, "contains volumeMounts that do not have matched volume: %v", notExistMount)
		return fmt.Errorf("contains volumeMounts that do not have matched volume: %v", notExistMount)
	}

	// This will block statefulset upgrade, but we should still let volume replacer run?
	// TODO this section should be obsoleted, because it will silently skip calling this function altogether.
	// TODO REMOVE BEFORE PR, this is failsafe meanwhile.
	if features.DefaultFeatureGate.Enabled(features.VolumeReplacing) {
		// Throw error, to allow volume replacing to take effect instead.
		if err := claimsSame(oldTiDBSet, newTiDBSet); err != nil {
			deps.Recorder.Eventf(tc, corev1.EventTypeWarning, reason, err.Error())
			return err
		}
	}

	return UpdateStatefulSet(deps.StatefulSetControl, tc, newTiDBSet, oldTiDBSet)
}

// UpdateStatefulSet is a template function to update the statefulset of components
func UpdateStatefulSet(setCtl controller.StatefulSetControlInterface, object runtime.Object, newSet, oldSet *apps.StatefulSet) error {
	isOrphan := metav1.GetControllerOf(oldSet) == nil
	if newSet.Annotations == nil {
		newSet.Annotations = map[string]string{}
	}
	if oldSet.Annotations == nil {
		oldSet.Annotations = map[string]string{}
	}

	// Check if an upgrade is needed.
	// If not, early return.
	if util.StatefulSetEqual(*newSet, *oldSet) && !isOrphan {
		return nil
	}

	set := *oldSet

	// update specs for sts
	*set.Spec.Replicas = *newSet.Spec.Replicas
	set.Spec.UpdateStrategy = newSet.Spec.UpdateStrategy
	set.Labels = newSet.Labels
	set.Annotations = newSet.Annotations
	set.Spec.Template = newSet.Spec.Template
	if isOrphan {
		set.OwnerReferences = newSet.OwnerReferences
	}

	var podConfig string
	var hasPodConfig bool
	if oldSet.Spec.Template.Annotations != nil {
		podConfig, hasPodConfig = oldSet.Spec.Template.Annotations[LastAppliedConfigAnnotation]
	}
	if hasPodConfig {
		if set.Spec.Template.Annotations == nil {
			set.Spec.Template.Annotations = map[string]string{}
		}
		set.Spec.Template.Annotations[LastAppliedConfigAnnotation] = podConfig
	}
	v, ok := oldSet.Annotations[label.AnnStsLastSyncTimestamp]
	if ok {
		set.Annotations[label.AnnStsLastSyncTimestamp] = v
	}

	err := SetStatefulSetLastAppliedConfigAnnotation(&set)
	if err != nil {
		return err
	}

	// commit to k8s
	_, err = setCtl.UpdateStatefulSet(object, &set)
	return err
}

// SetUpgradePartition set statefulSet's rolling update partition
func SetUpgradePartition(set *apps.StatefulSet, upgradeOrdinal int32) {
	set.Spec.UpdateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{Partition: &upgradeOrdinal}
	klog.Infof("set %s/%s partition to %d", set.GetNamespace(), set.GetName(), upgradeOrdinal)
}
