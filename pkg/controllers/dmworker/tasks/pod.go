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
	"path/filepath"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/image"
	"github.com/pingcap/tidb-operator/v2/pkg/overlay"
	"github.com/pingcap/tidb-operator/v2/pkg/reloadable"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/v2/pkg/utils/map"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TaskPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Pod", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)

		expected := newPod(state.Cluster(), state.DMWorker())
		pod := state.Pod()
		if pod == nil {
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't create pod of dm-worker: %v", err)
			}
			state.SetPod(expected)
			return task.Complete().With("pod is created")
		}

		if !reloadable.CheckDMWorkerPod(state.DMWorker(), pod) {
			logger.Info("will recreate the pod")
			if err := c.Delete(ctx, pod); err != nil {
				return task.Fail().With("can't delete pod of dm-worker: %v", err)
			}
			state.DeletePod(pod)
			return task.Wait().With("pod is deleting")
		}

		logger.Info("will update the pod in place")
		if err := c.Apply(ctx, expected); err != nil {
			return task.Fail().With("can't apply pod of dm-worker: %v", err)
		}

		state.SetPod(expected)
		return task.Complete().With("pod is synced")
	})
}

func newPod(cluster *v1alpha1.Cluster, dw *v1alpha1.DMWorker) *corev1.Pod {
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: coreutil.PodName[scope.DMWorker](dw),
					},
				},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      v1alpha1.VolumeNameConfig,
			MountPath: v1alpha1.DirPathConfigDMWorker,
		},
	}

	// RelayVolume mount (optional)
	if relayVol := dw.Spec.RelayVolume; relayVol != nil {
		relayVolName := dmWorkerVolumeName(relayVol.Name)
		relayClaimName := coreutil.PersistentVolumeClaimName[scope.DMWorker](dw, relayVol.Name)
		vols = append(vols, corev1.Volume{
			Name: relayVolName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: relayClaimName,
				},
			},
		})

		relayMountPath := v1alpha1.VolumeMountDMWorkerRelayDefaultPath
		for i := range relayVol.Mounts {
			mount := &relayVol.Mounts[i]
			if mount.Type == v1alpha1.VolumeMountTypeDMWorkerRelay && mount.MountPath != "" {
				relayMountPath = mount.MountPath
				break
			}
		}
		mounts = append(mounts, corev1.VolumeMount{
			Name:      relayVolName,
			MountPath: relayMountPath,
		})
	}

	// Additional volumes from dw.Spec.Volumes
	for i := range dw.Spec.Volumes {
		vol := &dw.Spec.Volumes[i]
		name := dmWorkerVolumeName(vol.Name)
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: coreutil.PersistentVolumeClaimName[scope.DMWorker](dw, vol.Name),
				},
			},
		})
		for j := range vol.Mounts {
			mount := &vol.Mounts[j]
			mounts = append(mounts, corev1.VolumeMount{
				Name:      name,
				MountPath: mount.MountPath,
				SubPath:   mount.SubPath,
			})
		}
	}

	// TLS volume
	if coreutil.IsTLSClusterEnabled(cluster) {
		vols = append(vols, *coreutil.ClusterTLSVolume[scope.DMWorker](dw))
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameClusterTLS,
			MountPath: v1alpha1.DirPathClusterTLSDMWorker,
			ReadOnly:  true,
		})
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dw.Namespace,
			Name:      coreutil.PodName[scope.DMWorker](dw),
			Labels: maputil.Merge(
				coreutil.PodLabels[scope.DMWorker](dw),
				map[string]string{
					v1alpha1.LabelKeyClusterID: cluster.Status.ID,
				},
				k8s.LabelsK8sApp(cluster.Name, v1alpha1.LabelValComponentDMWorker),
			),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(dw, v1alpha1.SchemeGroupVersion.WithKind("DMWorker")),
			},
		},
		Spec: corev1.PodSpec{
			Hostname:     coreutil.PodName[scope.DMWorker](dw),
			Subdomain:    dw.Spec.Subdomain,
			NodeSelector: dw.Spec.Topology,
			Containers: []corev1.Container{
				{
					Name:            v1alpha1.ContainerNameDMWorker,
					Image:           image.DM.Image(dw.Spec.Image, dw.Spec.Version),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"/dm-worker",
						"--config",
						filepath.Join(v1alpha1.DirPathConfigDMWorker, v1alpha1.FileNameConfig),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          v1alpha1.DMWorkerPortName,
							ContainerPort: coreutil.DMWorkerPort(dw),
						},
					},
					VolumeMounts: mounts,
					Resources:    k8s.GetResourceRequirements(dw.Spec.Resources),
				},
			},
			Volumes: vols,
		},
	}

	if dw.Spec.Overlay != nil {
		overlay.OverlayPod(pod, dw.Spec.Overlay.Pod)
	}

	reloadable.MustEncodeLastDMWorkerTemplate(dw, pod)
	return pod
}

func dmWorkerVolumeName(volName string) string {
	return metav1alpha1.VolNamePrefix + volName
}
