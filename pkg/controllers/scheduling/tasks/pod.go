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
	"path"
	"path/filepath"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/image"
	"github.com/pingcap/tidb-operator/v2/pkg/overlay"
	"github.com/pingcap/tidb-operator/v2/pkg/reloadable"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/v2/pkg/utils/map"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

const (
	metricsPath = "/metrics"
)

func TaskPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Pod", func(ctx context.Context) task.Result {
		ck := state.Cluster()
		obj := state.Object()
		logger := logr.FromContextOrDiscard(ctx)
		expected := newPod(ck, obj, state.FeatureGates())
		pod := state.Pod()
		if pod == nil {
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't create pod of scheduling: %v", err)
			}
			state.SetPod(expected)
			return task.Complete().With("pod is synced")
		}

		if !reloadable.CheckSchedulingPod(obj, pod) {
			// TODO: transfer leader ?
			logger.Info("will delete the pod to recreate", "name", pod.Name, "namespace", pod.Namespace, "UID", pod.UID)

			if err := c.Delete(ctx, pod); err != nil {
				return task.Fail().With("can't delete pod of scheduling: %v", err)
			}

			state.DeletePod(pod)

			return task.Wait().With("pod is deleting")
		}

		logger.Info("will update the pod in place")
		if err := c.Apply(ctx, expected); err != nil {
			return task.Fail().With("can't apply pod of scheduling: %v", err)
		}
		state.SetPod(expected)

		return task.Complete().With("pod is synced")
	})
}

func newPod(cluster *v1alpha1.Cluster, scheduling *v1alpha1.Scheduling, fg features.Gates) *corev1.Pod {
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: coreutil.PodName[scope.Scheduling](scheduling),
					},
				},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      v1alpha1.VolumeNameConfig,
			MountPath: v1alpha1.DirPathConfigScheduling,
		},
	}

	for i := range scheduling.Spec.Volumes {
		vol := &scheduling.Spec.Volumes[i]
		name := VolumeName(vol.Name)
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: coreutil.PersistentVolumeClaimName[scope.Scheduling](scheduling, vol.Name),
				},
			},
		})
		for i := range vol.Mounts {
			mount := VolumeMount(name, &vol.Mounts[i])
			mounts = append(mounts, *mount)
		}
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		vols = append(vols, *coreutil.ClusterTLSVolume[scope.Scheduling](scheduling))
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameClusterTLS,
			MountPath: v1alpha1.DirPathClusterTLSScheduling,
			ReadOnly:  true,
		})
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   scheduling.Namespace,
			Name:        coreutil.PodName[scope.Scheduling](scheduling),
			Labels:      coreutil.PodLabels[scope.Scheduling](scheduling),
			Annotations: maputil.Merge(k8s.AnnoProm(coreutil.SchedulingClientPort(scheduling), metricsPath)),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(scheduling, v1alpha1.SchemeGroupVersion.WithKind("Scheduling")),
			},
		},
		Spec: corev1.PodSpec{
			Hostname:     coreutil.PodName[scope.Scheduling](scheduling),
			Subdomain:    scheduling.Spec.Subdomain,
			NodeSelector: scheduling.Spec.Topology,
			Containers: []corev1.Container{
				{
					Name:            v1alpha1.ContainerNameScheduling,
					Image:           image.Scheduling.Image(scheduling.Spec.Image, scheduling.Spec.Version),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"/pd-server",
						"services",
						"scheduling",
						"--config",
						filepath.Join(v1alpha1.DirPathConfigScheduling, v1alpha1.FileNameConfig),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          v1alpha1.SchedulingPortNameClient,
							ContainerPort: coreutil.SchedulingClientPort(scheduling),
						},
					},
					VolumeMounts: mounts,
					Resources:    k8s.GetResourceRequirements(scheduling.Spec.Resources),
				},
			},
			Volumes: vols,
		},
	}

	if fg.Enabled(metav1alpha1.UseSchedulingReadyAPI) {
		pod.Spec.Containers[0].ReadinessProbe = buildReadinessProbe(cluster, coreutil.SchedulingClientPort(scheduling))
	}

	if scheduling.Spec.Overlay != nil {
		overlay.OverlayPod(pod, scheduling.Spec.Overlay.Pod)
	}

	reloadable.MustEncodeLastSchedulingTemplate(scheduling, pod)
	return pod
}

func buildReadinessProbe(cluster *v1alpha1.Cluster, port int32) *corev1.Probe {
	tlsClusterEnabled := coreutil.IsTLSClusterEnabled(cluster)

	scheme := "http"
	if tlsClusterEnabled {
		scheme = "https"
	}

	readinessURL := fmt.Sprintf("%s://127.0.0.1:%d/health", scheme, port)
	var command []string
	command = append(command, "curl", readinessURL,
		// Fail silently (no output at all) on server errors
		// without this if the server return 500, the exist code will be 0
		// and probe is success.
		"--fail",
		// Silent mode, only print error
		"-sS",
		// follow 301 or 302 redirect
		"--location")

	if tlsClusterEnabled {
		cacert := path.Join(v1alpha1.DirPathClusterTLSScheduling, corev1.ServiceAccountRootCAKey)
		cert := path.Join(v1alpha1.DirPathClusterTLSScheduling, corev1.TLSCertKey)
		key := path.Join(v1alpha1.DirPathClusterTLSScheduling, corev1.TLSPrivateKeyKey)
		command = append(command, "--cacert", cacert, "--cert", cert, "--key", key)
	}

	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: command,
			},
		},
	}
}
