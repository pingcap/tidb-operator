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
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	tiflashcfg "github.com/pingcap/tidb-operator/pkg/configs/tiflash"
	"github.com/pingcap/tidb-operator/pkg/image"
	"github.com/pingcap/tidb-operator/pkg/overlay"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	metricsPath = "/metrics"
)

func TaskPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Pod", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		expected := newPod(state)
		if state.Pod() == nil {
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't apply pod of tiflash: %w", err)
			}

			state.SetPod(expected)
			return task.Complete().With("pod is created")
		}

		res := k8s.ComparePods(state.Pod(), expected)
		curHash, expectHash := state.Pod().Labels[v1alpha1.LabelKeyConfigHash], expected.Labels[v1alpha1.LabelKeyConfigHash]
		configChanged := curHash != expectHash
		logger.Info("compare pod", "result", res, "configChanged", configChanged, "currentConfigHash", curHash, "expectConfigHash", expectHash)

		if res == k8s.CompareResultRecreate || (configChanged &&
			state.TiFlash().Spec.UpdateStrategy.Config == v1alpha1.ConfigUpdateStrategyRestart) {
			logger.Info("will recreate the pod")
			if err := c.Delete(ctx, state.Pod()); err != nil {
				return task.Fail().With("can't delete pod of tiflash: %w", err)
			}

			state.PodIsTerminating = true
			return task.Wait().With("pod is deleting")
		} else if res == k8s.CompareResultUpdate {
			logger.Info("will update the pod in place")
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't apply pod of tiflash: %w", err)
			}
			state.SetPod(expected)
		}

		return task.Complete().With("pod is synced")
	})
}

func newPod(state *ReconcileContext) *corev1.Pod {
	cluster := state.Cluster()
	tiflash := state.TiFlash()
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

	var dataMount *corev1.VolumeMount
	var dataDir string
	for i := range tiflash.Spec.Volumes {
		vol := &tiflash.Spec.Volumes[i]
		name := VolumeName(vol.Name)
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PersistentVolumeClaimName(tiflash.PodName(), vol.Name),
				},
			},
		})
		for i := range vol.Mounts {
			mount := &vol.Mounts[i]
			vm := VolumeMount(name, mount)
			if mount.Type == v1alpha1.VolumeMountTypeTiFlashData {
				dataMount = vm
				dataDir = dataMount.MountPath
			}
			mounts = append(mounts, *vm)
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
				v1alpha1.LabelKeyConfigHash: state.ConfigHash,
				v1alpha1.LabelKeyClusterID:  cluster.Status.ID,
				v1alpha1.LabelKeyStoreID:    state.StoreID,
			}, k8s.LabelsK8sApp(cluster.Name, v1alpha1.LabelValComponentTiFlash)),
			Annotations: maputil.Merge(tiflash.GetAnnotations(),
				k8s.AnnoProm(tiflash.GetMetricsPort(), metricsPath),
				k8s.AnnoAdditionalProm("tiflash.proxy", tiflash.GetProxyStatusPort())),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tiflash, v1alpha1.SchemeGroupVersion.WithKind("TiFlash")),
			},
		},
		Spec: corev1.PodSpec{
			Hostname:     tiflash.PodName(),
			Subdomain:    tiflash.Spec.Subdomain,
			NodeSelector: tiflash.Spec.Topology,
			InitContainers: []corev1.Container{
				*buildLogTailerContainer(tiflash, v1alpha1.TiFlashServerLogContainerName, tiflashcfg.GetServerLogPath(dataDir), dataMount),
				*buildLogTailerContainer(tiflash, v1alpha1.TiFlashErrorLogContainerName, tiflashcfg.GetErrorLogPath(dataDir), dataMount),
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
		Command: []string{
			"sh",
			"-c",
			fmt.Sprintf("touch %s; tail -n0 -F %s;", logFile, logFile),
		},
	}
	if mount != nil {
		c.VolumeMounts = append(c.VolumeMounts, *mount)
	}
	if tiflash.Spec.LogTailer != nil {
		c.Resources = k8s.GetResourceRequirements(tiflash.Spec.LogTailer.Resources)
	}
	return c
}
