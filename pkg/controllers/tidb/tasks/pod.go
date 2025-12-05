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
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/v2/pkg/utils/map"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

const (
	// gracefulCloseConnectionsTimeout is the amount of time tidb-server wait for the ongoing txt to finished.
	// The value is fixed in tidb-server.
	gracefulCloseConnectionsTimeout = 15

	// bufferSeconds is the extra seconds to wait for the pod to be terminated.
	bufferSeconds = 5
	// preStopSleepSeconds is the seconds to sleep before the container is terminated.
	defaultPreStopSleepSeconds int32 = 10

	// defaultReadinessProbeInitialDelaySeconds is the default initial delay seconds for readiness probe.
	// This is the same value as TiDB Operator v1.
	defaultReadinessProbeInitialDelaySeconds = 10

	metricsPath = "/metrics"
)

func TaskPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Pod", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)

		expected := newPod(state.Cluster(), state.TiDB(), state.FeatureGates())
		pod := state.Pod()
		if pod == nil {
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't create pod of tidb: %w", err)
			}

			state.SetPod(expected)
			return task.Complete().With("pod is created")
		}

		if !reloadable.CheckTiDBPod(state.TiDB(), pod) {
			logger.Info("will recreate the pod")
			if err := c.Delete(ctx, pod); err != nil {
				return task.Fail().With("can't delete pod of tidb: %w", err)
			}

			state.DeletePod(pod)
			return task.Wait().With("pod is restarting")
		}

		logger.Info("try to update the pod in place")
		if err := c.Apply(ctx, expected); err != nil {
			return task.Fail().With("can't apply pod of tidb: %w", err)
		}

		state.SetPod(expected)

		return task.Complete().With("pod is synced")
	})
}

func newPod(cluster *v1alpha1.Cluster, tidb *v1alpha1.TiDB, g features.Gates) *corev1.Pod {
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: coreutil.PodName[scope.TiDB](tidb),
					},
				},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      v1alpha1.VolumeNameConfig,
			MountPath: v1alpha1.DirPathConfigTiDB,
		},
	}

	var slowLogMount *corev1.VolumeMount
	for i := range tidb.Spec.Volumes {
		vol := &tidb.Spec.Volumes[i]
		name := VolumeName(vol.Name)
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PersistentVolumeClaimName(coreutil.PodName[scope.TiDB](tidb), vol.Name),
				},
			},
		})

		for i := range vol.Mounts {
			mount := &vol.Mounts[i]
			vm := VolumeMount(name, mount)
			mounts = append(mounts, *vm)
			if mount.Type == v1alpha1.VolumeMountTypeTiDBSlowLog {
				slowLogMount = vm
			}
		}
	}

	if coreutil.IsTiDBMySQLTLSEnabled(tidb) {
		vols = append(vols, *coreutil.TiDBMySQLTLSVolume(tidb))
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameMySQLTLS,
			MountPath: v1alpha1.DirPathMySQLTLS,
			ReadOnly:  true,
		})
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		vols = append(vols, *coreutil.ClusterTLSVolume[scope.TiDB](tidb))

		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameClusterTLS,
			MountPath: v1alpha1.DirPathClusterTLSTiDB,
			ReadOnly:  true,
		})
	}

	if cluster.Spec.BootstrapSQL != nil {
		vols = append(vols, corev1.Volume{
			Name: v1alpha1.VolumeNameBootstrapSQL,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cluster.Spec.BootstrapSQL.Name,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  v1alpha1.ConfigMapKeyBootstrapSQL,
							Path: v1alpha1.FileNameBootstrapSQL,
						},
					},
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameBootstrapSQL,
			MountPath: v1alpha1.DirPathBootstrapSQL,
			ReadOnly:  true,
		})
	}

	if coreutil.IsTokenBasedAuthEnabled(tidb) {
		vols = append(vols, corev1.Volume{
			Name: v1alpha1.VolumeNameTiDBAuthToken,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: coreutil.AuthTokenJWKSSecretName(tidb),
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameTiDBAuthToken,
			MountPath: v1alpha1.DirPathTiDBAuthToken,
			ReadOnly:  true,
		})
	}

	if coreutil.IsSEMEnabled(tidb) {
		vols = append(vols, corev1.Volume{
			Name: v1alpha1.VolumeNameSEM,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: coreutil.SEMConfigMapName(tidb),
					},
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameSEM,
			MountPath: v1alpha1.DirPathSEMConfig,
			ReadOnly:  true,
		})
	}

	if g.Enabled(metav1alpha1.SessionTokenSigning) {
		vols = append(vols, corev1.Volume{
			Name: v1alpha1.VolumeNameTiDBSessionTokenSigningTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: coreutil.SessionTokenSigningCertSecretName(cluster, tidb),
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameTiDBSessionTokenSigningTLS,
			MountPath: v1alpha1.DirPathTiDBSessionTokenSigningTLS,
			ReadOnly:  true,
		})
	}

	var slowLogContainer *corev1.Container
	if coreutil.IsSeparateSlowLogEnabled(tidb) {
		// no persistent slowlog volume
		if slowLogMount == nil {
			vol, mount := defaultSlowLogVolumeAndMount()
			vols = append(vols, *vol)
			mounts = append(mounts, *mount)
			slowLogMount = mount
		}
		slowLogContainer = buildSlowLogContainer(tidb, slowLogMount, g)
	}

	sleepSeconds := defaultPreStopSleepSeconds
	if tidb.Spec.PreStop != nil {
		sleepSeconds = tidb.Spec.PreStop.SleepSeconds
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tidb.Namespace,
			Name:      coreutil.PodName[scope.TiDB](tidb),
			Labels: maputil.Merge(
				coreutil.PodLabels[scope.TiDB](tidb),
				// legacy labels in v1
				map[string]string{
					v1alpha1.LabelKeyClusterID: cluster.Status.ID,
				},
				// TODO: remove it
				k8s.LabelsK8sApp(cluster.Name, v1alpha1.LabelValComponentTiDB),
			),

			Annotations: maputil.Merge(
				k8s.AnnoProm(coreutil.TiDBStatusPort(tidb), metricsPath),
			),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tidb, v1alpha1.SchemeGroupVersion.WithKind("TiDB")),
			},
		},
		Spec: corev1.PodSpec{
			Hostname:     coreutil.PodName[scope.TiDB](tidb),
			Subdomain:    tidb.Spec.Subdomain,
			NodeSelector: tidb.Spec.Topology,
			Containers: []corev1.Container{
				{
					Name:            v1alpha1.ContainerNameTiDB,
					Image:           image.TiDB.Image(tidb.Spec.Image, tidb.Spec.Version),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"/tidb-server",
						"--config",
						filepath.Join(v1alpha1.DirPathConfigTiDB, v1alpha1.FileNameConfig),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          v1alpha1.TiDBPortNameClient,
							ContainerPort: coreutil.TiDBClientPort(tidb),
						},
						{
							Name:          v1alpha1.TiDBPortNameStatus,
							ContainerPort: coreutil.TiDBStatusPort(tidb),
						},
					},
					VolumeMounts: mounts,
					Resources:    k8s.GetResourceRequirements(tidb.Spec.Resources),
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
						ProbeHandler:        buildTiDBReadinessProbeHandler(cluster, tidb, coreutil.TiDBClientPort(tidb), coreutil.TiDBStatusPort(tidb)),
						InitialDelaySeconds: defaultReadinessProbeInitialDelaySeconds,
					},
				},
			},
			Volumes:                       vols,
			TerminationGracePeriodSeconds: ptr.To(int64(sleepSeconds + gracefulCloseConnectionsTimeout + bufferSeconds)),
		},
	}

	if slowLogContainer != nil {
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, *slowLogContainer)
	}

	if tidb.Spec.Overlay != nil {
		overlay.OverlayPod(pod, tidb.Spec.Overlay.Pod)
	}

	reloadable.MustEncodeLastTiDBTemplate(tidb, pod)

	return pod
}

// TODO(liubo02): extract to namer pkg
func buildTiDBReadinessProbeHandler(cluster *v1alpha1.Cluster, tidb *v1alpha1.TiDB, clientPort, statusPort int32) corev1.ProbeHandler {
	probeType := v1alpha1.TCPProbeType // default to TCP probe
	if tidb.Spec.Probes.Readiness != nil && tidb.Spec.Probes.Readiness.Type != nil {
		probeType = *tidb.Spec.Probes.Readiness.Type
	}

	if probeType == v1alpha1.CommandProbeType {
		return corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: buildTiDBProbeCommand(cluster, statusPort),
			},
		}
	}

	return corev1.ProbeHandler{
		TCPSocket: &corev1.TCPSocketAction{
			Port: intstr.FromInt32(clientPort),
		},
	}
}

func buildTiDBProbeCommand(cluster *v1alpha1.Cluster, statusPort int32) (command []string) {
	scheme := "http"
	if coreutil.IsTLSClusterEnabled(cluster) {
		scheme = "https"
	}
	host := "127.0.0.1"

	readinessURL := fmt.Sprintf("%s://%s:%d/status", scheme, host, statusPort)
	command = append(command, "curl", readinessURL,
		// Fail silently (no output at all) on server errors
		// without this if the server return 500, the exist code will be 0
		// and probe is success.
		"--fail",
		// follow 301 or 302 redirect
		"--location")

	if coreutil.IsTLSClusterEnabled(cluster) {
		cacert := path.Join(v1alpha1.DirPathClusterTLSTiDB, corev1.ServiceAccountRootCAKey)
		cert := path.Join(v1alpha1.DirPathClusterTLSTiDB, corev1.TLSCertKey)
		key := path.Join(v1alpha1.DirPathClusterTLSTiDB, corev1.TLSPrivateKeyKey)
		command = append(command, "--cacert", cacert, "--cert", cert, "--key", key)
	}
	return
}

func defaultSlowLogVolumeAndMount() (*corev1.Volume, *corev1.VolumeMount) {
	return &corev1.Volume{
			Name: v1alpha1.VolumeNameTiDBSlowLogDefault,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}, &corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameTiDBSlowLogDefault,
			MountPath: v1alpha1.DirPathTiDBSlowLogDefault,
		}
}

func buildSlowLogContainer(tidb *v1alpha1.TiDB, mount *corev1.VolumeMount, fg features.Gates) *corev1.Container {
	slowlogFile := path.Join(mount.MountPath, v1alpha1.FileNameTiDBSlowLog)
	img := image.Helper.Image(nil)

	if tidb.Spec.SlowLog != nil {
		img = image.Helper.Image(tidb.Spec.SlowLog.Image)
	}

	cmd := []string{
		"sh",
		"-c",
	}

	if fg.Enabled(metav1alpha1.TerminableLogTailer) {
		// 1. Need to trap TERM or the sidecar container cannot be terminated
		// 2. Need to sleep 3 to avoid losing last logs
		cmd = append(cmd, fmt.Sprintf(`trap "sleep 3; exit 0" TERM; touch %s; tail -n0 -F %s & wait $!`, slowlogFile, slowlogFile))
	} else {
		cmd = append(cmd, fmt.Sprintf("touch %s; tail -n0 -F %s;", slowlogFile, slowlogFile))
	}

	restartPolicy := corev1.ContainerRestartPolicyAlways // sidecar container in `initContainers`
	c := &corev1.Container{
		Name:          v1alpha1.ContainerNameTiDBSlowLog,
		Image:         img,
		RestartPolicy: &restartPolicy,
		VolumeMounts:  []corev1.VolumeMount{*mount},
		Command:       cmd,
	}
	if tidb.Spec.SlowLog != nil {
		c.Resources = k8s.GetResourceRequirements(tidb.Spec.SlowLog.Resources)
	}
	return c
}
