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

const (
	metricsPath = "/metrics"
)

func TaskPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Pod", func(ctx context.Context) task.Result {
		ck := state.Cluster()
		obj := state.Object()
		logger := logr.FromContextOrDiscard(ctx)
		expected := newPod(ck, obj)
		pod := state.Pod()
		if pod == nil {
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't create pod of router: %v", err)
			}
			state.SetPod(expected)
			return task.Complete().With("pod is synced")
		}

		if !reloadable.CheckRouterPod(obj, pod) {
			logger.Info("will delete the pod to recreate", "name", pod.Name, "namespace", pod.Namespace, "UID", pod.UID)

			if err := c.Delete(ctx, pod); err != nil {
				return task.Fail().With("can't delete pod of router: %v", err)
			}

			state.DeletePod(pod)
			return task.Wait().With("pod is deleting")
		}

		logger.Info("will update the pod in place")
		if err := c.Apply(ctx, expected); err != nil {
			return task.Fail().With("can't apply pod of router: %v", err)
		}
		state.SetPod(expected)

		return task.Complete().With("pod is synced")
	})
}

func newPod(cluster *v1alpha1.Cluster, router *v1alpha1.Router) *corev1.Pod {
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: coreutil.PodName[scope.Router](router),
					},
				},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      v1alpha1.VolumeNameConfig,
			MountPath: v1alpha1.DirPathConfigRouter,
		},
	}

	for i := range router.Spec.Volumes {
		vol := &router.Spec.Volumes[i]
		name := VolumeName(vol.Name)
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: coreutil.PersistentVolumeClaimName[scope.Router](router, vol.Name),
				},
			},
		})
		for i := range vol.Mounts {
			mount := VolumeMount(name, &vol.Mounts[i])
			mounts = append(mounts, *mount)
		}
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		vols = append(vols, *coreutil.ClusterTLSVolume[scope.Router](router))
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameClusterTLS,
			MountPath: v1alpha1.DirPathClusterTLSRouter,
			ReadOnly:  true,
		})
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   router.Namespace,
			Name:        coreutil.PodName[scope.Router](router),
			Labels:      coreutil.PodLabels[scope.Router](router),
			Annotations: maputil.Merge(k8s.AnnoProm(coreutil.RouterClientPort(router), metricsPath)),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(router, v1alpha1.SchemeGroupVersion.WithKind("Router")),
			},
		},
		Spec: corev1.PodSpec{
			Hostname:     coreutil.PodName[scope.Router](router),
			Subdomain:    router.Spec.Subdomain,
			NodeSelector: router.Spec.Topology,
			Containers: []corev1.Container{
				{
					Name:            v1alpha1.ContainerNameRouter,
					Image:           image.Router.Image(router.Spec.Image, router.Spec.Version),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"/pd-server",
						"services",
						"router",
						"--config",
						filepath.Join(v1alpha1.DirPathConfigRouter, v1alpha1.FileNameConfig),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          v1alpha1.RouterPortNameClient,
							ContainerPort: coreutil.RouterClientPort(router),
						},
					},
					VolumeMounts:   mounts,
					ReadinessProbe: buildReadinessProbe(cluster, coreutil.RouterClientPort(router)),
					Resources:      k8s.GetResourceRequirements(router.Spec.Resources),
				},
			},
			Volumes: vols,
		},
	}

	if router.Spec.Overlay != nil {
		overlay.OverlayPod(pod, router.Spec.Overlay.Pod)
	}

	reloadable.MustEncodeLastRouterTemplate(router, pod)
	return pod
}

func buildReadinessProbe(cluster *v1alpha1.Cluster, port int32) *corev1.Probe {
	tlsClusterEnabled := coreutil.IsTLSClusterEnabled(cluster)

	scheme := "http"
	if tlsClusterEnabled {
		scheme = "https"
	}

	readinessURL := fmt.Sprintf("%s://127.0.0.1:%d/status", scheme, port)
	command := []string{
		"curl",
		readinessURL,
		"--fail",
		"-sS",
		"--location",
	}

	if tlsClusterEnabled {
		cacert := path.Join(v1alpha1.DirPathClusterTLSRouter, corev1.ServiceAccountRootCAKey)
		cert := path.Join(v1alpha1.DirPathClusterTLSRouter, corev1.TLSCertKey)
		key := path.Join(v1alpha1.DirPathClusterTLSRouter, corev1.TLSPrivateKeyKey)
		command = append(command, "--cacert", cacert, "--cert", cert, "--key", key)
	}

	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{Command: command},
		},
	}
}
