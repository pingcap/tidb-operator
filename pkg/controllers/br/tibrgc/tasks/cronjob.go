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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1alpha1br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/overlay"
	t "github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func taskDeleteT2CronJob(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("DeleteT2CronJob", func(ctx context.Context) t.Result {
		t2Cronjob := &batchv1.CronJob{
			ObjectMeta: v1.ObjectMeta{
				Name:      T2CronjobName(rtx.TiBRGC()),
				Namespace: rtx.NamespacedName().Namespace,
			},
		}
		err := rtx.Client().Delete(ctx, t2Cronjob)
		if err != nil {
			if errors.IsNotFound(err) {
				return t.Complete().With("t2 cronjob not found, skip")
			}
			return t.Fail().With("failed to delete t2 cronjob: %s", err.Error())
		}
		return t.Complete().With("t2 cronjob deleted")
	})
}

func taskDeleteT3CronJob(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("DeleteT3CronJob", func(ctx context.Context) t.Result {
		t3Cronjob := &batchv1.CronJob{
			ObjectMeta: v1.ObjectMeta{
				Name:      T3CronjobName(rtx.TiBRGC()),
				Namespace: rtx.NamespacedName().Namespace,
			},
		}
		err := rtx.Client().Delete(ctx, t3Cronjob)
		if err != nil {
			if errors.IsNotFound(err) {
				return t.Complete().With("t3 cronjob not found, skip")
			}
			return t.Fail().With("failed to delete t3 cronjob: %s", err.Error())
		}
		return t.Complete().With("t3 cronjob deleted")
	})
}

func taskApplyT2CronJob(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("ApplyT2CronJob", func(ctx context.Context) t.Result {
		t2Cronjob := assembleT2Cronjob(rtx)
		err := rtx.Client().Apply(ctx, t2Cronjob)
		if err != nil {
			return t.Fail().With("failed to apply t2 cronjob: %s", err.Error())
		}
		return t.Complete().With("t2 cronjob applied")
	})
}

func assembleT2Cronjob(rtx *ReconcileContext) *batchv1.CronJob {
	t2Strategy := rtx.TiBRGC().Spec.GetT2Strategy()

	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Labels: T2ConjobLabels(rtx.TiBRGC()),
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyOnFailure,
			Containers: []corev1.Container{
				{
					Name:         string(v1alpha1br.TieredStorageStrategyNameToT2Storage),
					Image:        GetImage(rtx.TiBRGC()),
					Resources:    getStrategyResources(t2Strategy),
					Command:      getT2CronjobCommand(rtx, t2Strategy),
					VolumeMounts: getT2CronjobVolumeMounts(rtx.TLSEnabled()),
				},
			},
			Volumes: getT2CronjobVolumes(rtx.TLSEnabled(), rtx.TiBRGC()),
		},
	}

	podOverlay := rtx.TiBRGC().Spec.GetOverlay(v1alpha1br.TieredStorageStrategyNameToT2Storage)
	if podOverlay != nil && podOverlay.Pod != nil {
		overlay.OverlayPod(pod, podOverlay.Pod)
	}

	t2Cronjob := &batchv1.CronJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      T2CronjobName(rtx.TiBRGC()),
			Namespace: rtx.NamespacedName().Namespace,
			Labels:    T2ConjobLabels(rtx.TiBRGC()),
			OwnerReferences: []v1.OwnerReference{
				*v1.NewControllerRef(rtx.TiBRGC(), v1alpha1br.SchemeGroupVersion.WithKind("TiBRGC")),
			},
		},
		Spec: batchv1.CronJobSpec{
			ConcurrencyPolicy:          batchv1.ForbidConcurrent,
			FailedJobsHistoryLimit:     ptr.To(int32(1)),
			SuccessfulJobsHistoryLimit: ptr.To(int32(3)),
			Suspend:                    ptr.To(false),
			Schedule: func(s string) string {
				if s == "" {
					return "0 0 * * *"
				}
				return s
			}(t2Strategy.Schedule),
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					BackoffLimit: ptr.To(T2DefaultBackoffLimit),
					Template: corev1.PodTemplateSpec{
						ObjectMeta: pod.ObjectMeta,
						Spec:       pod.Spec,
					},
				},
			},
		},
	}

	return t2Cronjob
}

func taskApplyT3CronJob(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("ApplyT3CronJob", func(ctx context.Context) t.Result {
		t3Cronjob := assembleT3Cronjob(rtx)
		err := rtx.Client().Apply(ctx, t3Cronjob)
		if err != nil {
			return t.Fail().With("failed to apply t3 cronjob: %s", err.Error())
		}
		return t.Complete().With("t3 cronjob applied")
	})
}

func assembleT3Cronjob(rtx *ReconcileContext) *batchv1.CronJob {
	t3Strategy := rtx.TiBRGC().Spec.GetT3Strategy()

	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Labels: T3ConjobLabels(rtx.TiBRGC()),
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyOnFailure,
			Containers: []corev1.Container{
				{
					Name:         string(v1alpha1br.TieredStorageStrategyNameToT3Storage),
					Image:        GetImage(rtx.TiBRGC()),
					Resources:    getStrategyResources(t3Strategy),
					Command:      getT3CronjobCommand(rtx.Cluster().Status.PD, rtx.TLSEnabled(), t3Strategy),
					VolumeMounts: getT3CronjobVolumeMounts(rtx.TLSEnabled(), t3Strategy),
				},
			},
			Volumes: getT3CronjobVolumes(rtx.TLSEnabled(), rtx.TiBRGC()),
		},
	}

	podOverlay := rtx.TiBRGC().Spec.GetOverlay(v1alpha1br.TieredStorageStrategyNameToT3Storage)
	if podOverlay != nil && podOverlay.Pod != nil {
		overlay.OverlayPod(pod, podOverlay.Pod)
	}

	t3Cronjob := &batchv1.CronJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      T3CronjobName(rtx.TiBRGC()),
			Namespace: rtx.NamespacedName().Namespace,
			Labels:    T3ConjobLabels(rtx.TiBRGC()),
			OwnerReferences: []v1.OwnerReference{
				*v1.NewControllerRef(rtx.TiBRGC(), v1alpha1br.SchemeGroupVersion.WithKind("TiBRGC")),
			},
		},
		Spec: batchv1.CronJobSpec{
			ConcurrencyPolicy:          batchv1.ForbidConcurrent,
			FailedJobsHistoryLimit:     ptr.To(int32(1)),
			SuccessfulJobsHistoryLimit: ptr.To(int32(3)),
			Suspend:                    ptr.To(false),
			Schedule: func(s string) string {
				if s == "" {
					return "30 0 * * *"
				}
				return s
			}(t3Strategy.Schedule),
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					BackoffLimit: ptr.To(T3DefaultBackoffLimit),
					Template: corev1.PodTemplateSpec{
						ObjectMeta: pod.ObjectMeta,
						Spec:       pod.Spec,
					},
				},
			},
		},
	}

	return t3Cronjob
}

func getStrategyResources(strategy *v1alpha1br.TieredStorageStrategy) corev1.ResourceRequirements {
	var cpu, memory *resource.Quantity
	if strategy.Resources.CPU != nil {
		cpu = strategy.Resources.CPU
	}
	if strategy.Resources.Memory != nil {
		memory = strategy.Resources.Memory
	}

	reqs := corev1.ResourceRequirements{}
	if cpu != nil || memory != nil {
		reqs.Requests = make(corev1.ResourceList)
		reqs.Limits = make(corev1.ResourceList)
	}
	if cpu != nil {
		reqs.Requests[corev1.ResourceCPU] = *cpu
		reqs.Limits[corev1.ResourceCPU] = *cpu
	}
	if memory != nil {
		reqs.Requests[corev1.ResourceMemory] = *memory
		reqs.Limits[corev1.ResourceMemory] = *memory
	}

	return reqs
}

func getTLSVolume(tibrgc *v1alpha1br.TiBRGC) corev1.Volume {
	return corev1.Volume{
		Name: TLSVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  SecretName(tibrgc),
				DefaultMode: ptr.To(SecretAccessMode),
			},
		},
	}
}

func getT2CronjobCommand(rtx *ReconcileContext, strategy *v1alpha1br.TieredStorageStrategy) []string {
	command := []string{
		"/cse-ctl",
		"dfs-gc",
		"--pd",
		rtx.Cluster().Status.PD,
		"--start-time-safe-interval",
		fmt.Sprintf("%dd", strategy.TimeThresholdDays),
	}

	if len(strategy.Options) > 0 {
		command = append(command, strategy.Options...)
	}

	if rtx.TLSEnabled() {
		command = append(command, TLSCmdArgs...)
	}

	return command
}

func getT3CronjobCommand(pdAddr string, tlsEnabled bool, strategy *v1alpha1br.TieredStorageStrategy) []string {
	command := []string{
		"/cse-ctl",
		"archive",
		"--pd",
		pdAddr,
		"--start-archive-duration",
		fmt.Sprintf("%dd", strategy.TimeThresholdDays),
	}

	// data dir
	hasDataDir := false
	for _, vol := range strategy.Volumes {
		if vol.Name == "data" {
			if len(vol.Mounts) > 0 {
				path := vol.Mounts[0].MountPath
				if vol.Mounts[0].SubPath != "" {
					path = path + "/" + vol.Mounts[0].SubPath
				}
				command = append(command, "--data-dir", path)
				hasDataDir = true
			}
			break
		}
	}

	if !hasDataDir {
		command = append(command, "--data-dir", "/data")
	}

	if len(strategy.Options) > 0 {
		command = append(command, strategy.Options...)
	}

	if tlsEnabled {
		command = append(command, TLSCmdArgs...)
	}

	return command
}

func getT2CronjobVolumeMounts(tlsEnabled bool) []corev1.VolumeMount {
	if tlsEnabled {
		return []corev1.VolumeMount{tlsVolumeMount}
	}
	return nil
}

func getT2CronjobVolumes(tlsEnabled bool, tibrgc *v1alpha1br.TiBRGC) []corev1.Volume {
	if tlsEnabled {
		return []corev1.Volume{getTLSVolume(tibrgc)}
	}
	return nil
}

func getT3CronjobVolumeMounts(tlsEnabled bool, strategy *v1alpha1br.TieredStorageStrategy) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount
	if tlsEnabled {
		volumeMounts = append(volumeMounts, tlsVolumeMount)
	}

	if strategy != nil && len(strategy.Volumes) > 0 {
		for _, vol := range strategy.Volumes {
			for _, m := range vol.Mounts {
				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					Name:      vol.Name,
					MountPath: m.MountPath,
					SubPath:   m.SubPath,
				})
			}
		}
	}
	return volumeMounts
}

func getT3CronjobVolumes(tlsEnabled bool, tibrgc *v1alpha1br.TiBRGC) []corev1.Volume {
	var volumes []corev1.Volume
	if tlsEnabled {
		volumes = append(volumes, getTLSVolume(tibrgc))
	}

	strategy := tibrgc.Spec.GetT3Strategy()
	if strategy == nil || len(strategy.Volumes) == 0 {
		return volumes
	}

	for _, vol := range strategy.Volumes {
		pvc := &corev1.PersistentVolumeClaim{
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
		pvcOverlay := tibrgc.Spec.GetPVCOverlay(v1alpha1br.TieredStorageStrategyNameToT3Storage, vol.Name)
		overlay.OverlayPersistentVolumeClaim(pvc, pvcOverlay)

		theVolume := corev1.Volume{
			Name: vol.Name,
			VolumeSource: corev1.VolumeSource{
				Ephemeral: &corev1.EphemeralVolumeSource{
					VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{
						ObjectMeta: pvc.ObjectMeta,
						Spec:       pvc.Spec,
					},
				},
			},
		}

		volumes = append(volumes, theVolume)
	}

	return volumes
}
