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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/image"
	"github.com/pingcap/tidb-operator/pkg/overlay"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	// bufferSeconds is the extra seconds to wait for the pod to be deleted.
	bufferSeconds = 3
	// preStopSleepSeconds is the seconds to sleep before the pod is deleted.
	preStopSleepSeconds = 10

	// defaultReadinessProbeInitialDelaySeconds is the default initial delay seconds for readiness probe.
	// This is the same value as TiDB Operator v1.
	defaultReadinessProbeInitialDelaySeconds = 10
)

func TaskPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Pod", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)

		expected := newPod(state.Cluster(), state.TiDB(), state.GracefulWaitTimeInSeconds, state.ConfigHash)
		if state.Pod() == nil {
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't create pod of tidb: %w", err)
			}

			state.SetPod(expected)
			return task.Complete().With("pod is created")
		}

		res := k8s.ComparePods(state.Pod(), expected)
		curHash, expectHash := state.Pod().Labels[v1alpha1.LabelKeyConfigHash], expected.Labels[v1alpha1.LabelKeyConfigHash]
		configChanged := curHash != expectHash
		logger.Info("compare pod", "result", res, "configChanged", configChanged, "currentConfigHash", curHash, "expectConfigHash", expectHash)

		if res == k8s.CompareResultRecreate || (configChanged &&
			state.TiDB().Spec.UpdateStrategy.Config == v1alpha1.ConfigUpdateStrategyRestart) {
			logger.Info("will recreate the pod")
			if err := c.Delete(ctx, state.Pod()); err != nil {
				return task.Fail().With("can't delete pod of tidb: %w", err)
			}

			state.PodIsTerminating = true
			return task.Wait().With("pod is deleting")
		} else if res == k8s.CompareResultUpdate {
			logger.Info("will update the pod in place")
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't apply pod of tidb: %w", err)
			}

			state.SetPod(expected)
		}

		return task.Complete().With("pod is synced")
	})
}

func newPod(cluster *v1alpha1.Cluster,
	tidb *v1alpha1.TiDB, gracePeriod int64, configHash string,
) *corev1.Pod {
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tidb.PodName(),
					},
				},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      v1alpha1.VolumeNameConfig,
			MountPath: v1alpha1.DirNameConfigTiDB,
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
					ClaimName: PersistentVolumeClaimName(tidb.PodName(), vol.Name),
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

	if tidb.IsMySQLTLSEnabled() {
		vols = append(vols, corev1.Volume{
			Name: v1alpha1.TiDBSQLTLSVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tidb.MySQLTLSSecretName(),
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.TiDBSQLTLSVolumeName,
			MountPath: v1alpha1.TiDBSQLTLSMountPath,
			ReadOnly:  true,
		})
	}

	if cluster.IsTLSClusterEnabled() {
		vols = append(vols, corev1.Volume{
			Name: v1alpha1.TiDBClusterTLSVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tidb.TLSClusterSecretName(),
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.TiDBClusterTLSVolumeName,
			MountPath: v1alpha1.TiDBClusterTLSMountPath,
			ReadOnly:  true,
		})
	}

	if tidb.IsBootstrapSQLEnabled() {
		vols = append(vols, corev1.Volume{
			Name: v1alpha1.BootstrapSQLVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tidb.Spec.Security.BootstrapSQL.Name,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  v1alpha1.BootstrapSQLConfigMapKey,
							Path: v1alpha1.BootstrapSQLFileName,
						},
					},
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.BootstrapSQLVolumeName,
			MountPath: v1alpha1.BootstrapSQLFilePath,
			ReadOnly:  true,
		})
	}

	if tidb.IsTokenBasedAuthEnabled() {
		vols = append(vols, corev1.Volume{
			Name: v1alpha1.TiDBAuthTokenVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tidb.AuthTokenJWKSSecretName(),
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.TiDBAuthTokenVolumeName,
			MountPath: v1alpha1.TiDBAuthTokenPath,
			ReadOnly:  true,
		})
	}

	var slowLogContainer *corev1.Container
	if tidb.IsSeparateSlowLogEnabled() {
		// no persistent slowlog volume
		if slowLogMount == nil {
			vol, mount := defaultSlowLogVolumeAndMount()
			vols = append(vols, *vol)
			mounts = append(mounts, *mount)
			slowLogMount = mount
		}
		slowLogContainer = buildSlowLogContainer(tidb, slowLogMount)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tidb.Namespace,
			Name:      tidb.PodName(),
			Labels: maputil.Merge(tidb.Labels, map[string]string{
				v1alpha1.LabelKeyInstance:   tidb.Name,
				v1alpha1.LabelKeyConfigHash: configHash,
			}),
			Annotations: maputil.Copy(tidb.GetAnnotations()),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tidb, v1alpha1.SchemeGroupVersion.WithKind("TiDB")),
			},
		},
		Spec: corev1.PodSpec{
			Hostname:     tidb.PodName(),
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
						filepath.Join(v1alpha1.DirNameConfigTiDB, v1alpha1.ConfigFileName),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          v1alpha1.TiDBPortNameClient,
							ContainerPort: tidb.GetClientPort(),
						},
						{
							Name:          v1alpha1.TiDBPortNameStatus,
							ContainerPort: tidb.GetStatusPort(),
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
									fmt.Sprintf("sleep %d", preStopSleepSeconds),
								},
							},
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler:        buildTiDBReadinessProbHandler(cluster, tidb, tidb.GetClientPort(), tidb.GetStatusPort()),
						InitialDelaySeconds: defaultReadinessProbeInitialDelaySeconds,
					},
				},
			},
			Volumes:                       vols,
			TerminationGracePeriodSeconds: ptr.To(gracePeriod + preStopSleepSeconds + bufferSeconds),
		},
	}

	if slowLogContainer != nil {
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, *slowLogContainer)
	}

	if tidb.Spec.Overlay != nil {
		overlay.OverlayPod(pod, tidb.Spec.Overlay.Pod)
	}

	k8s.CalculateHashAndSetLabels(pod)
	return pod
}

// TODO(liubo02): extract to namer pkg

func buildTiDBReadinessProbHandler(cluster *v1alpha1.Cluster, tidb *v1alpha1.TiDB, clientPort, statusPort int32) corev1.ProbeHandler {
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
			Port: intstr.FromInt(int(clientPort)),
		},
	}
}

func buildTiDBProbeCommand(cluster *v1alpha1.Cluster, statusPort int32) (command []string) {
	scheme := "http"
	if cluster.IsTLSClusterEnabled() {
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

	if cluster.IsTLSClusterEnabled() {
		cacert := path.Join(v1alpha1.TiDBClusterTLSMountPath, corev1.ServiceAccountRootCAKey)
		cert := path.Join(v1alpha1.TiDBClusterTLSMountPath, corev1.TLSCertKey)
		key := path.Join(v1alpha1.TiDBClusterTLSMountPath, corev1.TLSPrivateKeyKey)
		command = append(command, "--cacert", cacert, "--cert", cert, "--key", key)
	}
	return
}

func defaultSlowLogVolumeAndMount() (*corev1.Volume, *corev1.VolumeMount) {
	return &corev1.Volume{
			Name: v1alpha1.TiDBDefaultSlowLogVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}, &corev1.VolumeMount{
			Name:      v1alpha1.TiDBDefaultSlowLogVolumeName,
			MountPath: v1alpha1.VolumeMountTiDBSlowLogDefaultPath,
		}
}

func buildSlowLogContainer(tidb *v1alpha1.TiDB, mount *corev1.VolumeMount) *corev1.Container {
	slowlogFile := path.Join(mount.MountPath, v1alpha1.TiDBSlowLogFileName)
	img := v1alpha1.DefaultHelperImage
	if tidb.Spec.SlowLog != nil && tidb.Spec.SlowLog.Image != nil && *tidb.Spec.SlowLog.Image != "" {
		img = *tidb.Spec.SlowLog.Image
	}
	restartPolicy := corev1.ContainerRestartPolicyAlways // sidecar container in `initContainers`
	c := &corev1.Container{
		Name:          v1alpha1.TiDBSlowLogContainerName,
		Image:         img,
		RestartPolicy: &restartPolicy,
		VolumeMounts:  []corev1.VolumeMount{*mount},
		Command: []string{
			"sh",
			"-c",
			fmt.Sprintf("touch %s; tail -n0 -F %s;", slowlogFile, slowlogFile),
		},
	}
	if tidb.Spec.SlowLog != nil {
		c.Resources = k8s.GetResourceRequirements(tidb.Spec.SlowLog.Resources)
	}
	return c
}
