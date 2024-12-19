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
	"fmt"
	"path/filepath"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	tiflashcfg "github.com/pingcap/tidb-operator/pkg/configs/tiflash"
	"github.com/pingcap/tidb-operator/pkg/image"
	"github.com/pingcap/tidb-operator/pkg/overlay"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
)

func TaskPodSuspend(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("PodSuspend", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		if rtx.Pod == nil {
			return task.Complete().With("pod has been deleted")
		}
		if err := c.Delete(rtx, rtx.Pod); err != nil {
			return task.Fail().With("can't delete pod of pd: %w", err)
		}
		rtx.PodIsTerminating = true
		return task.Wait().With("pod is deleting")
	})
}

type TaskPod struct {
	Client client.Client
	Logger logr.Logger
}

func NewTaskPod(logger logr.Logger, c client.Client) task.Task[ReconcileContext] {
	return &TaskPod{
		Client: c,
		Logger: logger,
	}
}

func (*TaskPod) Name() string {
	return "Pod"
}

func (t *TaskPod) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	expected := t.newPod(rtx.Cluster, rtx.TiFlash, rtx.ConfigHash)
	if rtx.Pod == nil {
		if err := t.Client.Apply(rtx, expected); err != nil {
			return task.Fail().With("can't apply pod of tiflash: %w", err)
		}

		rtx.Pod = expected
		return task.Complete().With("pod is created")
	}

	res := k8s.ComparePods(rtx.Pod, expected)
	curHash, expectHash := rtx.Pod.Labels[v1alpha1.LabelKeyConfigHash], expected.Labels[v1alpha1.LabelKeyConfigHash]
	configChanged := curHash != expectHash
	t.Logger.Info("compare pod", "result", res, "configChanged", configChanged, "currentConfigHash", curHash, "expectConfigHash", expectHash)

	if res == k8s.CompareResultRecreate || (configChanged &&
		rtx.TiFlash.Spec.UpdateStrategy.Config == v1alpha1.ConfigUpdateStrategyRestart) {
		t.Logger.Info("will recreate the pod")
		if err := t.Client.Delete(rtx, rtx.Pod); err != nil {
			return task.Fail().With("can't delete pod of tiflash: %w", err)
		}

		rtx.PodIsTerminating = true
		return task.Complete().With("pod is deleting")
	} else if res == k8s.CompareResultUpdate {
		t.Logger.Info("will update the pod in place")
		if err := t.Client.Apply(rtx, expected); err != nil {
			return task.Fail().With("can't apply pod of tiflash: %w", err)
		}
		rtx.Pod = expected
	}

	return task.Complete().With("pod is synced")
}

func (*TaskPod) newPod(cluster *v1alpha1.Cluster, tiflash *v1alpha1.TiFlash, configHash string) *corev1.Pod {
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tiflash.PodName(),
					},
				},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      v1alpha1.VolumeNameConfig,
			MountPath: v1alpha1.DirNameConfigTiFlash,
		},
	}

	var firstMount *corev1.VolumeMount
	for i := range tiflash.Spec.Volumes {
		vol := &tiflash.Spec.Volumes[i]
		name := v1alpha1.NamePrefix + "tiflash"
		if vol.Name != "" {
			name = name + "-" + vol.Name
		}
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					// the format is "data{i}-tiflash-xxx" to compatible with TiDB Operator v1
					ClaimName: PersistentVolumeClaimName(tiflash.PodName(), i),
				},
			},
		})
		mount := corev1.VolumeMount{
			Name:      name,
			MountPath: vol.Path,
		}
		mounts = append(mounts, mount)
		if i == 0 {
			firstMount = &mount
		}
	}

	if cluster.IsTLSClusterEnabled() {
		vols = append(vols, corev1.Volume{
			Name: v1alpha1.TiFlashClusterTLSVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tiflash.TLSClusterSecretName(),
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.TiFlashClusterTLSVolumeName,
			MountPath: v1alpha1.TiFlashClusterTLSMountPath,
			ReadOnly:  true,
		})
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tiflash.Namespace,
			Name:      tiflash.PodName(),
			Labels: maputil.Merge(tiflash.Labels, map[string]string{
				v1alpha1.LabelKeyInstance:   tiflash.Name,
				v1alpha1.LabelKeyConfigHash: configHash,
			}),
			Annotations: maputil.Copy(tiflash.GetAnnotations()),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tiflash, v1alpha1.SchemeGroupVersion.WithKind("TiFlash")),
			},
		},
		Spec: corev1.PodSpec{
			Hostname:     tiflash.PodName(),
			Subdomain:    tiflash.Spec.Subdomain,
			NodeSelector: tiflash.Spec.Topology,
			InitContainers: []corev1.Container{
				*buildLogTailerContainer(tiflash, v1alpha1.TiFlashServerLogContainerName, tiflashcfg.GetServerLogPath(tiflash), firstMount),
				*buildLogTailerContainer(tiflash, v1alpha1.TiFlashErrorLogContainerName, tiflashcfg.GetErrorLogPath(tiflash), firstMount),
			},
			Containers: []corev1.Container{
				{
					Name:            v1alpha1.ContainerNameTiFlash,
					Image:           image.TiFlash.Image(tiflash.Spec.Image, tiflash.Spec.Version),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"/tiflash/tiflash",
						"server",
						"--config-file",
						filepath.Join(v1alpha1.DirNameConfigTiFlash, v1alpha1.ConfigFileName),
					},
					Ports: []corev1.ContainerPort{
						// no `tcp_port` and `http_port` as they are are deprecated in tiflash since v7.1.0.
						// ref: https://github.com/pingcap/tidb-operator/pull/5075
						// and also no `interserver_http_port`
						{
							Name:          v1alpha1.TiFlashPortNameFlash,
							ContainerPort: tiflash.GetFlashPort(),
						},
						{
							Name:          v1alpha1.TiFlashPortNameMetrics,
							ContainerPort: tiflash.GetMetricsPort(),
						},
						{
							Name:          v1alpha1.TiFlashPortNameProxy,
							ContainerPort: tiflash.GetProxyPort(),
						},
						{
							// no this port in v1
							Name:          v1alpha1.TiFlashPortNameProxyStatus,
							ContainerPort: tiflash.GetProxyStatusPort(),
						},
					},
					VolumeMounts: mounts,
					Resources:    k8s.GetResourceRequirements(tiflash.Spec.Resources),
				},
			},
			Volumes: vols,
		},
	}

	if tiflash.Spec.Overlay != nil {
		overlay.OverlayPod(pod, tiflash.Spec.Overlay.Pod)
	}

	k8s.CalculateHashAndSetLabels(pod)
	return pod
}

func buildLogTailerContainer(tiflash *v1alpha1.TiFlash, containerName, logFile string, mount *corev1.VolumeMount) *corev1.Container {
	img := v1alpha1.DefaultHelperImage
	if tiflash.Spec.LogTailer != nil && tiflash.Spec.LogTailer.Image != nil && *tiflash.Spec.LogTailer.Image != "" {
		img = *tiflash.Spec.LogTailer.Image
	}
	restartPolicy := corev1.ContainerRestartPolicyAlways // sidecar container in `initContainers`
	c := &corev1.Container{
		Name:          containerName,
		Image:         img,
		RestartPolicy: &restartPolicy,
		VolumeMounts:  []corev1.VolumeMount{*mount},
		Command: []string{
			"sh",
			"-c",
			fmt.Sprintf("touch %s; tail -n0 -F %s;", logFile, logFile),
		},
	}
	if tiflash.Spec.LogTailer != nil {
		c.Resources = k8s.GetResourceRequirements(tiflash.Spec.LogTailer.Resources)
	}
	return c
}
