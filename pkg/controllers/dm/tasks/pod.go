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

const metricsPath = "/metrics"

func TaskPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Pod", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)

		expected := newPod(state.Cluster(), state.DM())
		pod := state.Pod()
		if pod == nil {
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't create pod of dm: %v", err)
			}
			state.SetPod(expected)
			return task.Complete().With("pod is created")
		}

		if !reloadable.CheckDMPod(state.DM(), pod) {
			logger.Info("will recreate the pod")
			if err := c.Delete(ctx, pod); err != nil {
				return task.Fail().With("can't delete pod of dm: %v", err)
			}
			state.DeletePod(pod)
			return task.Wait().With("pod is deleting")
		}

		logger.Info("will update the pod in place")
		if err := c.Apply(ctx, expected); err != nil {
			return task.Fail().With("can't apply pod of dm: %v", err)
		}

		state.SetPod(expected)
		return task.Complete().With("pod is synced")
	})
}

func newPod(cluster *v1alpha1.Cluster, dm *v1alpha1.DM) *corev1.Pod {
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: coreutil.PodName[scope.DM](dm),
					},
				},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      v1alpha1.VolumeNameConfig,
			MountPath: v1alpha1.DirPathConfigDMMaster,
		},
	}

	// DataVolume mount
	dataVolName := dmVolumeName(dm.Spec.DataVolume.Name)
	dataClaimName := coreutil.PersistentVolumeClaimName[scope.DM](dm, dm.Spec.DataVolume.Name)
	vols = append(vols, corev1.Volume{
		Name: dataVolName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: dataClaimName,
			},
		},
	})

	dataMountPath := v1alpha1.VolumeMountDMDataDefaultPath
	var dataSubPath string
	for i := range dm.Spec.DataVolume.Mounts {
		mount := &dm.Spec.DataVolume.Mounts[i]
		if mount.Type == v1alpha1.VolumeMountTypeDMData && mount.MountPath != "" {
			dataMountPath = mount.MountPath
			dataSubPath = mount.SubPath
			break
		}
	}
	mounts = append(mounts, corev1.VolumeMount{
		Name:      dataVolName,
		MountPath: dataMountPath,
		SubPath:   dataSubPath,
	})

	// Additional volumes from dm.Spec.Volumes
	for i := range dm.Spec.Volumes {
		vol := &dm.Spec.Volumes[i]
		name := dmVolumeName(vol.Name)
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: coreutil.PersistentVolumeClaimName[scope.DM](dm, vol.Name),
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
		vols = append(vols, *coreutil.ClusterTLSVolume[scope.DM](dm))
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameClusterTLS,
			MountPath: v1alpha1.DirPathClusterTLSDMMaster,
			ReadOnly:  true,
		})
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dm.Namespace,
			Name:      coreutil.PodName[scope.DM](dm),
			Labels: maputil.Merge(
				coreutil.PodLabels[scope.DM](dm),
				// legacy labels in v1
				map[string]string{
					v1alpha1.LabelKeyClusterID: cluster.Status.ID,
				},
				// TODO: remove it
				k8s.LabelsK8sApp(cluster.Name, v1alpha1.LabelValComponentDMMaster),
				map[string]string{
					// Keep the legacy DM app name for user monitoring collectors that select DM metrics by this label.
					k8s.LabelKeyK8sAppName: k8s.LabelValK8sAppNameDMCluster,
				},
			),
			Annotations: maputil.Merge(k8s.AnnoProm(coreutil.DMPort(dm), metricsPath)),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(dm, v1alpha1.SchemeGroupVersion.WithKind("DM")),
			},
		},
		Spec: corev1.PodSpec{
			Hostname:     coreutil.PodName[scope.DM](dm),
			Subdomain:    dm.Spec.Subdomain,
			NodeSelector: dm.Spec.Topology,
			Containers: []corev1.Container{
				{
					Name:            v1alpha1.ContainerNameDMMaster,
					Image:           image.DM.Image(dm.Spec.Image, dm.Spec.Version),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"/dm-master",
						"--config",
						filepath.Join(v1alpha1.DirPathConfigDMMaster, v1alpha1.FileNameConfig),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          v1alpha1.DMPortName,
							ContainerPort: coreutil.DMPort(dm),
						},
						{
							Name:          v1alpha1.DMPeerPortName,
							ContainerPort: coreutil.DMPeerPort(dm),
						},
					},
					VolumeMounts: mounts,
					Resources:    k8s.GetResourceRequirements(dm.Spec.Resources),
				},
			},
			Volumes: vols,
		},
	}

	if dm.Spec.Overlay != nil {
		overlay.OverlayPod(pod, dm.Spec.Overlay.Pod)
	}

	reloadable.MustEncodeLastDMTemplate(dm, pod)
	return pod
}

// dmVolumeName returns the volume name for a DM volume
func dmVolumeName(volName string) string {
	return metav1alpha1.VolNamePrefix + volName
}
