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
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/image"
	"github.com/pingcap/tidb-operator/pkg/overlay"
	"github.com/pingcap/tidb-operator/pkg/reloadable"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	// preStopSleepSeconds is the seconds to sleep before the container is terminated.
	defaultPreStopSleepSeconds int32 = 10

	// defaultReadinessProbeInitialDelaySeconds is the default initial delay seconds for readiness probe.
	defaultReadinessProbeInitialDelaySeconds = 5

	metricsPath = "/metrics"
)

func TaskPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Pod", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)

		expected := newPod(state.Cluster(), state.TiProxy())
		pod := state.Pod()
		if pod == nil {
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't create pod of tiproxy: %w", err)
			}

			state.SetPod(expected)
			return task.Complete().With("pod is created")
		}

		if !reloadable.CheckTiProxyPod(state.TiProxy(), pod) {
			logger.Info("will recreate the pod")
			if err := c.Delete(ctx, pod); err != nil {
				return task.Fail().With("can't delete pod of tiproxy: %w", err)
			}

			state.DeletePod(pod)
			return task.Wait().With("pod is restarting")
		}

		logger.Info("try to update the pod in place")
		if err := c.Apply(ctx, expected); err != nil {
			return task.Fail().With("can't apply pod of tiproxy: %w", err)
		}

		state.SetPod(expected)

		return task.Complete().With("pod is synced")
	})
}

func newPod(cluster *v1alpha1.Cluster, tiproxy *v1alpha1.TiProxy) *corev1.Pod {
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: coreutil.PodName[scope.TiProxy](tiproxy),
					},
				},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      v1alpha1.VolumeNameConfig,
			MountPath: v1alpha1.DirPathConfigTiProxy,
		},
	}

	for i := range tiproxy.Spec.Volumes {
		vol := &tiproxy.Spec.Volumes[i]
		name := VolumeName(vol.Name)
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PersistentVolumeClaimName(coreutil.PodName[scope.TiProxy](tiproxy), vol.Name),
				},
			},
		})

		for i := range vol.Mounts {
			mount := &vol.Mounts[i]
			vm := VolumeMount(name, mount)
			mounts = append(mounts, *vm)
		}
	}

	if coreutil.IsTiProxyMySQLTLSEnabled(tiproxy) {
		vols = append(vols, *coreutil.TiProxyMySQLTLSVolume(tiproxy))
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameTiProxyMySQLTLS,
			MountPath: v1alpha1.DirPathTiProxyMySQLTLS,
			ReadOnly:  true,
		})
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		vols = append(vols, *coreutil.ClusterTLSVolume[scope.TiProxy](tiproxy))
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameClusterTLS,
			MountPath: v1alpha1.DirPathClusterTLSTiProxy,
			ReadOnly:  true,
		})
	}

	if coreutil.IsTiProxyHTTPServerTLSEnabled(cluster, tiproxy) {
		vols = append(vols, *coreutil.TiProxyHTTPServerTLSVolume(tiproxy))
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameTiProxyHTTPTLS,
			MountPath: v1alpha1.DirPathTiProxyHTTPTLS,
			ReadOnly:  true,
		})
	}

	if coreutil.IsTiProxyBackendTLSEnabled(tiproxy) {
		vol := coreutil.TiProxyBackendTLSVolume(tiproxy)
		if vol != nil {
			vols = append(vols, *vol)
			mounts = append(mounts, corev1.VolumeMount{
				Name:      v1alpha1.VolumeNameTiProxyTiDBTLS,
				MountPath: v1alpha1.DirPathTiProxyTiDBTLS,
				ReadOnly:  true,
			})
		}
	}

	sleepSeconds := defaultPreStopSleepSeconds
	if tiproxy.Spec.PreStop != nil {
		sleepSeconds = tiproxy.Spec.PreStop.SleepSeconds
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tiproxy.Namespace,
			Name:      coreutil.PodName[scope.TiProxy](tiproxy),
			Labels: maputil.Merge(
				coreutil.PodLabels[scope.TiProxy](tiproxy),
				// legacy labels in v1
				map[string]string{
					v1alpha1.LabelKeyClusterID: cluster.Status.ID,
				},
				// TODO: remove it
				k8s.LabelsK8sApp(cluster.Name, v1alpha1.LabelValComponentTiProxy),
			),
			Annotations: maputil.Merge(
				k8s.AnnoProm(coreutil.TiProxyAPIPort(tiproxy), metricsPath),
			),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tiproxy, v1alpha1.SchemeGroupVersion.WithKind("TiProxy")),
			},
		},
		Spec: corev1.PodSpec{
			Hostname:     coreutil.PodName[scope.TiProxy](tiproxy),
			Subdomain:    tiproxy.Spec.Subdomain,
			NodeSelector: tiproxy.Spec.Topology,
			Containers: []corev1.Container{
				{
					Name:            v1alpha1.ContainerNameTiProxy,
					Image:           image.TiProxy.Image(tiproxy.Spec.Image, tiproxy.Spec.Version),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"/bin/tiproxy",
						"--config",
						filepath.Join(v1alpha1.DirPathConfigTiProxy, v1alpha1.FileNameConfig),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          v1alpha1.TiProxyPortNameClient,
							ContainerPort: coreutil.TiProxyClientPort(tiproxy),
						},
						{
							Name:          v1alpha1.TiProxyPortNameAPI,
							ContainerPort: coreutil.TiProxyAPIPort(tiproxy),
						},
						{
							Name:          v1alpha1.TiProxyPortNamePeer,
							ContainerPort: coreutil.TiProxyPeerPort(tiproxy),
						},
					},
					VolumeMounts: mounts,
					Resources:    k8s.GetResourceRequirements(tiproxy.Spec.Resources),
					Lifecycle: &corev1.Lifecycle{
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"/bin/sh",
									"-c",
									fmt.Sprintf("sleep %d", sleepSeconds),
								},
							},
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: buildTiProxyReadinessProbeHandler(
							cluster, tiproxy,
							coreutil.TiProxyClientPort(tiproxy),
							coreutil.TiProxyAPIPort(tiproxy)),
						InitialDelaySeconds: defaultReadinessProbeInitialDelaySeconds,
					},
				},
			},
			Volumes: vols,
		},
	}

	if tiproxy.Spec.Overlay != nil {
		overlay.OverlayPod(pod, tiproxy.Spec.Overlay.Pod)
	}

	reloadable.MustEncodeLastTiProxyTemplate(tiproxy, pod)

	return pod
}

// TODO(liubo02): extract to namer pkg
func buildTiProxyReadinessProbeHandler(
	cluster *v1alpha1.Cluster,
	tiproxy *v1alpha1.TiProxy,
	clientPort, statusPort int32,
) corev1.ProbeHandler {
	probeType := v1alpha1.CommandProbeType // default to command probe
	if tiproxy.Spec.Probes.Readiness != nil && tiproxy.Spec.Probes.Readiness.Type != nil {
		probeType = *tiproxy.Spec.Probes.Readiness.Type
	}

	// If TLS is enabled, we can only use command probe type
	if coreutil.IsTiProxyHTTPServerTLSEnabled(cluster, tiproxy) {
		probeType = v1alpha1.CommandProbeType
	}

	if probeType == v1alpha1.CommandProbeType {
		return corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: buildTiProxyProbeCommand(cluster, tiproxy, statusPort),
			},
		}
	}

	return corev1.ProbeHandler{
		TCPSocket: &corev1.TCPSocketAction{
			Port: intstr.FromInt32(clientPort),
		},
	}
}

func buildTiProxyProbeCommand(cluster *v1alpha1.Cluster, tiproxy *v1alpha1.TiProxy, statusPort int32) (command []string) {
	tlsEnabled := coreutil.IsTiProxyHTTPServerTLSEnabled(cluster, tiproxy)

	scheme := "http"
	if tlsEnabled {
		scheme = "https"
	}
	readinessURL := fmt.Sprintf("%s://127.0.0.1:%d/api/debug/health", scheme, statusPort)
	command = append(command, "curl", readinessURL,
		// Fail silently (no output at all) on server errors
		// without this if the server return 500, the exist code will be 0
		// and probe is success.
		"--fail",
		// Silent mode, only print error
		"-sS",
		// follow 301 or 302 redirect
		"--location")

	if tlsEnabled {
		cacert := path.Join(v1alpha1.DirPathTiProxyHTTPTLS, corev1.ServiceAccountRootCAKey)
		command = append(command, "--cacert", cacert)
		if !coreutil.IsTiProxyHTTPServerNoClientCert(tiproxy) {
			cert := path.Join(v1alpha1.DirPathTiProxyHTTPTLS, corev1.TLSCertKey)
			key := path.Join(v1alpha1.DirPathTiProxyHTTPTLS, corev1.TLSPrivateKeyKey)
			command = append(command, "--cert", cert, "--key", key)
		}
	}
	return
}
