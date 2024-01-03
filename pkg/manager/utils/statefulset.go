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

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
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
	stsEqual, podTemplateCheckedAndNotEqual := util.StatefulSetEqual(*newSet, *oldSet)
	if stsEqual && !isOrphan {
		return nil
	}

	set := *oldSet

	// update specs for sts
	*set.Spec.Replicas = *newSet.Spec.Replicas
	set.Spec.UpdateStrategy = newSet.Spec.UpdateStrategy
	set.Labels = newSet.Labels
	set.Annotations = newSet.Annotations
	set.Spec.Template = *newSet.Spec.Template.DeepCopy() // do a copy to avoid changing the original TidbCluster CR
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

	if !podTemplateCheckedAndNotEqual {
		// if the pod template is check and not equal, it means the pod template is changed
		// there is no need to hack the volume mode as it can be updated in this round
		// this should be done after SetStatefulSetLastAppliedConfigAnnotation,
		// then this volumeMode set by defaults won't be stored in the annotation
		hackEphemeralVolumeMode(oldSet, &set)
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

// hackEphemeralVolumeMode appends exstings ephemeral volume mode to asts so that no unexpected rolling update will be triggered.
// before https://github.com/pingcap/advanced-statefulset/pull/96, some asts may have volume mode in ephemeral volume,
// but after that, no volume mode in ephemeral volume will be set by defaults,
// so we need to append the volume mode to the new asts spec if it exists in old spec.
func hackEphemeralVolumeMode(oldSts *apps.StatefulSet, newSts *apps.StatefulSet) {
	if !features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		// only need this hack for AdvancedStatefulSet
		return
	}

	volumeModels := make(map[string]corev1.PersistentVolumeMode)
	for _, volume := range oldSts.Spec.Template.Spec.Volumes {
		if volume.Ephemeral != nil && volume.Ephemeral.VolumeClaimTemplate != nil &&
			volume.Ephemeral.VolumeClaimTemplate.Spec.VolumeMode != nil {
			volumeModels[volume.Name] = *volume.Ephemeral.VolumeClaimTemplate.Spec.VolumeMode
		}
	}
	if len(volumeModels) == 0 {
		return // no need to hack
	}

	for _, volume := range newSts.Spec.Template.Spec.Volumes {
		if volumeMode, ok := volumeModels[volume.Name]; ok && volume.Ephemeral != nil && volume.Ephemeral.VolumeClaimTemplate != nil &&
			volume.Ephemeral.VolumeClaimTemplate.Spec.VolumeMode == nil {
			volume.Ephemeral.VolumeClaimTemplate.Spec.VolumeMode = &volumeMode
			klog.Infof("hack volume mode %s for volume %s in sts %s/%s", volumeMode, volume.Name, newSts.Namespace, newSts.Name)
		}
	}
}
