// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tasks

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1alphabr "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/overlay"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	t "github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func condStatefulSetExist(rtx *ReconcileContext) t.Condition {
	return t.CondFunc(func() bool {
		return rtx.StatefulSet() != nil
	})
}

func taskCreateStatefulSet(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("CreateStatefulSet", func(ctx context.Context) t.Result {
		err := rtx.Client().Create(ctx, assembleSts(rtx))
		if err != nil {
			return t.Fail().With("failed to create statefulset: %s", err.Error())
		}
		return t.Complete().With("statefulset created")
	})
}

func assembleSts(rtx *ReconcileContext) *appsv1.StatefulSet {
	tibr := rtx.TiBR()
	labels := TiBRSubResourceLabels(rtx.TiBR())

	podSpec := assemblePodSpec(rtx)
	if tibr.Spec.Overlay != nil && tibr.Spec.Overlay.Pod != nil {
		overlay.OverlayPodSpec(podSpec, tibr.Spec.Overlay.Pod.Spec)
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: v1.ObjectMeta{
			Name:      StatefulSetName(tibr),
			Namespace: rtx.NamespacedName().Namespace,
			Labels:    labels,
			OwnerReferences: []v1.OwnerReference{
				*v1.NewControllerRef(rtx.TiBR(), v1alphabr.SchemeGroupVersion.WithKind("TiBR")),
			},
		},

		Spec: appsv1.StatefulSetSpec{
			ServiceName: HeadlessSvcName(tibr),
			Replicas:    ptr.To(StatefulSetReplica),
			Selector: &v1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: labels,
				},
				Spec: *podSpec,
			},
		},
	}

	volumeClaim := assembleVolumeClaimIfNeeded(tibr)
	if volumeClaim != nil {
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*volumeClaim}
	}

	return sts
}

// Generate PVC from user-defined volumes
// Notice: there should be at most one Volume, name as VolumeTiBRData with at most one Mount.
// Notice: need to add mount info to volumeMounts in container definition
func assembleVolumeClaimIfNeeded(tibr *v1alphabr.TiBR) *corev1.PersistentVolumeClaim {
	if len(tibr.Spec.Volumes) == 0 {
		return nil
	}

	vol := tibr.Spec.Volumes[0]
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Name: vol.Name,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: vol.StorageClassName,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: vol.Storage,
				},
			},
			VolumeAttributesClassName: vol.VolumeAttributesClassName,
		},
	}

	// overlay PVC from user-defined volumes by PVC Overlay
	if tibr.Spec.Overlay == nil || tibr.Spec.Overlay.PersistentVolumeClaims == nil {
		return pvc
	}
	for _, ol := range tibr.Spec.Overlay.PersistentVolumeClaims {
		if pvc.Name == ol.Name {
			// Notice: currently only support overlay metadata
			overlay.OverlayPersistentVolumeClaim(pvc, &ol.PersistentVolumeClaim)
			break
		}
	}

	return pvc
}

func assemblePodSpec(rtx *ReconcileContext) *corev1.PodSpec {
	containers := []corev1.Container{assembleServerContainer(rtx)}
	if rtx.TiBR().Spec.AutoSchedule != nil {
		containers = append(containers, assembleAutoBackupContainer(rtx))
	}
	return &corev1.PodSpec{
		Containers: containers,
		Volumes:    assembleVolumes(rtx),
	}
}

func assembleServerContainer(rtx *ReconcileContext) corev1.Container {
	pdAddr := rtx.Cluster().Status.PD
	// Notice: there should be at most one Volume, name as VolumeTiBRData with at most one Mount.
	dataVolumeMount := genUserDefinedVolumeMount(rtx.TiBR().Spec.Volumes)
	dataDir := v1alphabr.VolumeTiBRDataDefaultMountPath
	if dataVolumeMount != nil {
		dataDir = dataVolumeMount.MountPath
	}

	cmd := []string{
		"/tikv-worker",
		"--config",
		ConfigMountPath + "/" + ConfigFileName,
		"--addr",
		fmt.Sprintf("0.0.0.0:%d", APIServerPort),
		"--pd-endpoints",
		pdAddr,
		"--data-dir",
		dataDir,
	}
	if rtx.TLSEnabled() {
		cmd = append(cmd, TLSCmdArgs...)
	}

	volumeMounts := []corev1.VolumeMount{configVolumeMount}
	if rtx.TLSEnabled() {
		volumeMounts = append(volumeMounts, tlsVolumeMount)
	}

	if dataVolumeMount != nil {
		volumeMounts = append(volumeMounts, *dataVolumeMount)
	}

	return corev1.Container{
		Name:         v1alphabr.ContainerAPIServer,
		Image:        GetImage(rtx.TiBR()),
		Command:      cmd,
		VolumeMounts: volumeMounts,
		// Notice: only set resources for api-server container, auto-backup container will use default resources
		// since auto-backup container uses not much resource and logic in auto-backup container will be merged into api-server container in the future
		Resources: k8s.GetResourceRequirements(rtx.TiBR().Spec.Resources),
	}
}

func genUserDefinedVolumeMount(defs []v1alpha1.Volume) *corev1.VolumeMount {
	if len(defs) == 0 {
		return nil
	}

	def := defs[0]

	vm := &corev1.VolumeMount{
		Name:      v1alphabr.VolumeTiBRData,
		MountPath: v1alphabr.VolumeTiBRDataDefaultMountPath,
	}

	// Name is the connection between Volume and VolumeMount
	if def.Name != "" {
		vm.Name = def.Name
	}

	if len(def.Mounts) != 0 && def.Mounts[0].MountPath != "" {
		vm.MountPath = def.Mounts[0].MountPath
	}
	return vm
}

func assembleAutoBackupContainer(rtx *ReconcileContext) corev1.Container {
	// cmd
	period := calculatePeriodSeconds(rtx.TiBR().Spec.AutoSchedule.Type)
	pdAddr := rtx.Cluster().Status.PD
	cmd := []string{
		"/cse-ctl",
		"backup",
		"--lightweight",
		"--interval",
		period,
		"--tolerate-err",
		"1",
		"--pd",
		pdAddr,
	}
	if rtx.TLSEnabled() {
		cmd = append(cmd, TLSCmdArgs...)
	}

	// volume
	var volumeMounts []corev1.VolumeMount
	if rtx.TLSEnabled() {
		volumeMounts = append(volumeMounts, tlsVolumeMount)
	}

	return corev1.Container{
		Name:         v1alphabr.ContainerAutoBackup,
		Image:        GetImage(rtx.TiBR()),
		Command:      cmd,
		VolumeMounts: volumeMounts,
	}
}

func calculatePeriodSeconds(ty v1alphabr.TiBRAutoScheduleType) string {
	switch ty {
	case v1alphabr.TiBRAutoScheduleTypePerDay:
		return "86400" // 24 hours in seconds
	case v1alphabr.TiBRAutoScheduleTypePerHour:
		return "3600" // 1 hour in seconds
	case v1alphabr.TiBRAutoScheduleTypePerMinute:
		return "60"
	default:
		return "60"
	}
}

func assembleVolumes(rtx *ReconcileContext) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: ConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ConfigMapName(rtx.TiBR()),
					},
				},
			},
		},
	}
	if rtx.TLSEnabled() {
		volumes = append(volumes, corev1.Volume{
			Name: TLSVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  SecretName(rtx.TiBR()),
					DefaultMode: ptr.To(SecretAccessMode),
				},
			},
		})
	}
	return volumes
}

func taskDeleteStatefulSet(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("DeleteStatefulSet", func(ctx context.Context) t.Result {
		sts := &appsv1.StatefulSet{
			ObjectMeta: v1.ObjectMeta{
				Name:      StatefulSetName(rtx.TiBR()),
				Namespace: rtx.NamespacedName().Namespace,
			},
		}

		err := rtx.Client().Delete(ctx, sts)
		if err != nil {
			return t.Fail().With("failed to delete statefulset: %s", err.Error())
		}
		return t.Complete().With("statefulset deleted")
	})
}
