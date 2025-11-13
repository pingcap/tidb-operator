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
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/image"
	"github.com/pingcap/tidb-operator/v2/pkg/overlay"
	"github.com/pingcap/tidb-operator/v2/pkg/reloadable"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/v2/pkg/utils/map"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

const (
	MinGracePeriodSeconds = 30
	// Assume that approximately 200 regions are transferred for 1s
	RegionsPerSecond = 200

	metricsPath = "/metrics"
)

func TaskSuspendPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("PodSuspend", func(ctx context.Context) task.Result {
		pod := state.Pod()
		if pod == nil {
			return task.Complete().With("pod has been deleted")
		}
		regionCount := 0
		if state.Store != nil {
			regionCount = state.Store.RegionCount
		}
		if err := DeletePodWithGracePeriod(ctx, c, pod, regionCount); err != nil {
			return task.Fail().With("can't delete pod of tikv: %w", err)
		}
		state.DeletePod(pod)

		return task.Wait().With("pod is deleting")
	})
}

func TaskPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Pod", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		expected := newPod(state.Cluster(), state.TiKV(), state.Store, state.FeatureGates())
		pod := state.Pod()
		if pod == nil {
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't apply pod of tikv: %w", err)
			}

			state.SetPod(expected)
			return task.Complete().With("pod is created")
		}

		// Now we cannot minimize the deletion grace period because of a bug
		// See https://github.com/kubernetes/kubernetes/issues/83916.
		// TODO(liubo02): support minimize the deletion grace period seconds
		if !pod.GetDeletionTimestamp().IsZero() {
			// key will be requeued after the pod is changed
			return task.Wait().With("pod is deleting")
		}

		if !reloadable.CheckTiKVPod(state.TiKV(), pod) {
			logger.Info("will recreate the pod")
			regionCount := 0
			if state.Store != nil {
				regionCount = state.Store.RegionCount
			}
			if err := DeletePodWithGracePeriod(ctx, c, pod, regionCount); err != nil {
				return task.Fail().With("can't minimize the deletion grace period of pod of tikv: %w", err)
			}

			state.DeletePod(pod)
			return task.Wait().With("pod is deleting")
		}

		logger.Info("will update the pod in place")
		if err := c.Apply(ctx, expected); err != nil {
			return task.Fail().With("can't apply pod of tikv: %w", err)
		}

		// write apply result back to ctx
		state.SetPod(expected)

		return task.Complete().With("pod is synced")
	})
}

func newPod(cluster *v1alpha1.Cluster, tikv *v1alpha1.TiKV, store *pdv1.Store, g features.Gates) *corev1.Pod {
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: coreutil.PodName[scope.TiKV](tikv),
					},
				},
			},
		},
		{
			Name: v1alpha1.VolumeNamePrestopChecker,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      v1alpha1.VolumeNameConfig,
			MountPath: v1alpha1.DirPathConfigTiKV,
		},
		{
			Name:      v1alpha1.VolumeNamePrestopChecker,
			MountPath: v1alpha1.DirPathPrestop,
		},
	}

	for i := range tikv.Spec.Volumes {
		vol := &tikv.Spec.Volumes[i]
		name := VolumeName(vol.Name)
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PersistentVolumeClaimName(coreutil.PodName[scope.TiKV](tikv), vol.Name),
				},
			},
		})
		for i := range vol.Mounts {
			mount := VolumeMount(name, &vol.Mounts[i])
			mounts = append(mounts, *mount)
		}
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		vols = append(vols, *coreutil.ClusterTLSVolume[scope.TiKV](tikv))
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameClusterTLS,
			MountPath: v1alpha1.DirPathClusterTLSTiKV,
			ReadOnly:  true,
		})
	}

	var preStopImage *string
	if tikv.Spec.PreStop != nil {
		preStopImage = tikv.Spec.PreStop.Image
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tikv.Namespace,
			Name:      coreutil.PodName[scope.TiKV](tikv),
			Labels: maputil.Merge(
				coreutil.PodLabels[scope.TiKV](tikv),
				// TODO: remove it
				k8s.LabelsK8sApp(cluster.Name, v1alpha1.LabelValComponentTiKV),
			),
			Annotations: maputil.Merge(k8s.AnnoProm(coreutil.TiKVStatusPort(tikv), metricsPath)),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tikv, v1alpha1.SchemeGroupVersion.WithKind("TiKV")),
			},
		},
		Spec: corev1.PodSpec{
			// TODO: make the max grace period seconds configurable
			//nolint:mnd // refactor to use a constant
			TerminationGracePeriodSeconds: ptr.To[int64](65535),
			Hostname:                      coreutil.PodName[scope.TiKV](tikv),
			Subdomain:                     tikv.Spec.Subdomain,
			NodeSelector:                  tikv.Spec.Topology,
			InitContainers: []corev1.Container{
				{
					// TODO: support hot reload checker
					// NOTE: before k8s 1.32, sidecar cannot be restarted because of this https://github.com/kubernetes/kubernetes/pull/126525.
					Name:            v1alpha1.ContainerNamePrestopChecker,
					Image:           image.PrestopChecker.Image(preStopImage),
					ImagePullPolicy: corev1.PullIfNotPresent,
					// RestartPolicy:   ptr.To(corev1.ContainerRestartPolicyAlways),
					Command: []string{
						"/bin/sh",
						"-c",
						"cp /prestop-checker " + v1alpha1.DirPathPrestop + "/;",
						// "sleep infinity",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      v1alpha1.VolumeNamePrestopChecker,
							MountPath: v1alpha1.DirPathPrestop,
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:            v1alpha1.ContainerNameTiKV,
					Image:           image.TiKV.Image(tikv.Spec.Image, tikv.Spec.Version),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"/tikv-server",
						"--config",
						filepath.Join(v1alpha1.DirPathConfigTiKV, v1alpha1.FileNameConfig),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          v1alpha1.TiKVPortNameClient,
							ContainerPort: coreutil.TiKVClientPort(tikv),
						},
						{
							Name:          v1alpha1.TiKVPortNameStatus,
							ContainerPort: coreutil.TiKVStatusPort(tikv),
						},
					},
					VolumeMounts: mounts,
					Resources:    k8s.GetResourceRequirements(tikv.Spec.Resources),
					Lifecycle: &corev1.Lifecycle{
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"/bin/sh",
									"-c",
									buildPrestopCheckScript(cluster, tikv),
								},
							},
						},
					},
				},
			},
			Volumes: vols,
		},
	}

	// legacy labels in v1
	if cluster.Status.ID != "" {
		pod.Labels[v1alpha1.LabelKeyClusterID] = cluster.Status.ID
	}
	if store != nil && store.ID != "" {
		pod.Labels[v1alpha1.LabelKeyStoreID] = store.ID
	}

	if g.Enabled(metav1alpha1.UseTiKVReadyAPI) {
		pod.Spec.Containers[0].ReadinessProbe = buildReadinessProbeWithReadyAPI(cluster, coreutil.TiKVStatusPort(tikv))
	}

	if tikv.Spec.Overlay != nil {
		overlay.OverlayPod(pod, tikv.Spec.Overlay.Pod)
	}

	reloadable.MustEncodeLastTiKVTemplate(tikv, pod)
	return pod
}

func buildPrestopCheckScript(cluster *v1alpha1.Cluster, tikv *v1alpha1.TiKV) string {
	sb := strings.Builder{}
	sb.WriteString(v1alpha1.DirPathPrestop)
	sb.WriteString("/prestop-checker")
	sb.WriteString(" -mode ")
	sb.WriteString(" tikv ")
	sb.WriteString(" -pd ")
	sb.WriteString(cluster.Status.PD)
	sb.WriteString(" -tikv-status-addr ")
	sb.WriteString(coreutil.TiKVAdvertiseStatusURLs(tikv))

	if coreutil.IsTLSClusterEnabled(cluster) {
		sb.WriteString(" -tls ")
		sb.WriteString(" -ca ")
		sb.WriteString(v1alpha1.DirPathClusterTLSTiKV)
		sb.WriteString("/ca.crt")
		sb.WriteString(" -cert ")
		sb.WriteString(v1alpha1.DirPathClusterTLSTiKV)
		sb.WriteString("/tls.crt")
		sb.WriteString(" -key ")
		sb.WriteString(v1alpha1.DirPathClusterTLSTiKV)
		sb.WriteString("/tls.key")
	}

	sb.WriteString(" > /proc/1/fd/1")

	return sb.String()
}

func buildReadinessProbeWithReadyAPI(cluster *v1alpha1.Cluster, port int32) *corev1.Probe {
	tlsClusterEnabled := coreutil.IsTLSClusterEnabled(cluster)

	scheme := "http"
	if tlsClusterEnabled {
		scheme = "https"
	}

	readinessURL := fmt.Sprintf("%s://127.0.0.1:%d/ready", scheme, port)
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
		cacert := path.Join(v1alpha1.DirPathClusterTLSTiKV, corev1.ServiceAccountRootCAKey)
		cert := path.Join(v1alpha1.DirPathClusterTLSTiKV, corev1.TLSCertKey)
		key := path.Join(v1alpha1.DirPathClusterTLSTiKV, corev1.TLSPrivateKeyKey)
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
