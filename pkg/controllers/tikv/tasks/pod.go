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
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	kvcfg "github.com/pingcap/tidb-operator/pkg/configs/tikv"
	"github.com/pingcap/tidb-operator/pkg/image"
	"github.com/pingcap/tidb-operator/pkg/overlay"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
)

const (
	MinGracePeriodSeconds = 30
	// Assume that approximately 200 regions are transferred for 1s
	RegionsPerSecond = 200
)

func TaskPodSuspend(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("PodSuspend", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		if rtx.Pod == nil {
			return task.Complete().With("pod has been deleted")
		}
		if err := c.Delete(rtx, rtx.Pod); err != nil {
			return task.Fail().With("can't delete pod of tikv: %w", err)
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

//nolint:gocyclo // refactor if possible
func (t *TaskPod) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	expected := t.newPod(rtx.Cluster, rtx.TiKVGroup, rtx.TiKV, rtx.ConfigHash)
	if rtx.Pod == nil {
		if err := t.Client.Apply(rtx, expected); err != nil {
			return task.Fail().With("can't apply pod of tikv: %w", err)
		}

		rtx.Pod = expected
		return task.Complete().With("pod is created")
	}

	// minimize the deletion grace period seconds
	if !rtx.Pod.GetDeletionTimestamp().IsZero() {
		sec := rtx.Pod.GetDeletionGracePeriodSeconds()

		regionCount := 0
		if rtx.Store != nil {
			regionCount = rtx.Store.RegionCount
		}
		gracePeriod := int64(regionCount/RegionsPerSecond + 1)
		if gracePeriod < MinGracePeriodSeconds {
			gracePeriod = MinGracePeriodSeconds
		}

		if sec != nil && rtx.Store != nil && *sec > gracePeriod {
			if err := t.Client.Delete(ctx, rtx.Pod, client.GracePeriodSeconds(gracePeriod)); err != nil {
				return task.Fail().With("cannot minimize the shutdown timeout: %w", err)
			}
		}

		// key will be requeued after the pod is changed
		return task.Complete().With("pod is deleting")
	}

	res := k8s.ComparePods(rtx.Pod, expected)
	curHash, expectHash := rtx.Pod.Labels[v1alpha1.LabelKeyConfigHash], expected.Labels[v1alpha1.LabelKeyConfigHash]
	configChanged := curHash != expectHash
	t.Logger.Info("compare pod", "result", res, "configChanged", configChanged, "currentConfigHash", curHash, "expectConfigHash", expectHash)

	if res == k8s.CompareResultRecreate || (configChanged &&
		rtx.TiKVGroup.Spec.ConfigUpdateStrategy == v1alpha1.ConfigUpdateStrategyRollingUpdate) {
		t.Logger.Info("will recreate the pod")
		if err := t.Client.Delete(rtx, rtx.Pod); err != nil {
			return task.Fail().With("can't delete pod of tikv: %w", err)
		}

		rtx.PodIsTerminating = true
		return task.Complete().With("pod is deleting")
	} else if res == k8s.CompareResultUpdate {
		t.Logger.Info("will update the pod in place")
		if err := t.Client.Apply(rtx, expected); err != nil {
			return task.Fail().With("can't apply pod of tikv: %w", err)
		}

		// write apply result back to ctx
		rtx.Pod = expected
	}

	return task.Complete().With("pod is synced")
}

func (t *TaskPod) newPod(cluster *v1alpha1.Cluster, kvg *v1alpha1.TiKVGroup, tikv *v1alpha1.TiKV, configHash string) *corev1.Pod {
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ConfigMapName(tikv.Name),
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
			MountPath: v1alpha1.DirNameConfigTiKV,
		},
		{
			Name:      v1alpha1.VolumeNamePrestopChecker,
			MountPath: v1alpha1.DirNamePrestop,
		},
	}

	for i := range tikv.Spec.Volumes {
		vol := &tikv.Spec.Volumes[i]
		name := v1alpha1.NamePrefix + "tikv"
		if vol.Name != "" {
			name = name + "-" + vol.Name
		}
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PersistentVolumeClaimName(tikv.Name, vol.Name),
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      name,
			MountPath: vol.Path,
		})
	}

	if cluster.IsTLSClusterEnabled() {
		groupName := tikv.Labels[v1alpha1.LabelKeyGroup]
		vols = append(vols, corev1.Volume{
			Name: v1alpha1.TiKVClusterTLSVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: cluster.TLSClusterSecretName(groupName),
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.TiKVClusterTLSVolumeName,
			MountPath: v1alpha1.TiKVClusterTLSMountPath,
			ReadOnly:  true,
		})

		if kvg.MountClusterClientSecret() {
			vols = append(vols, corev1.Volume{
				Name: v1alpha1.ClusterTLSClientVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: cluster.ClusterClientTLSSecretName(),
					},
				},
			})
			mounts = append(mounts, corev1.VolumeMount{
				Name:      v1alpha1.ClusterTLSClientVolumeName,
				MountPath: v1alpha1.ClusterTLSClientMountPath,
				ReadOnly:  true,
			})
		}
	}

	var preStopImage *string
	if tikv.Spec.PreStop != nil {
		preStopImage = tikv.Spec.PreStop.Image
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tikv.Namespace,
			Name:      tikv.Name,
			Labels: maputil.Merge(tikv.Labels, map[string]string{
				v1alpha1.LabelKeyInstance:   tikv.Name,
				v1alpha1.LabelKeyConfigHash: configHash,
			}),
			Annotations: maputil.Copy(tikv.GetAnnotations()),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tikv, v1alpha1.SchemeGroupVersion.WithKind("TiKV")),
			},
		},
		Spec: corev1.PodSpec{
			// TODO: make the max grace period seconds configurable
			//nolint:mnd // refactor to use a constant
			TerminationGracePeriodSeconds: ptr.To[int64](65535),
			Hostname:                      tikv.Name,
			Subdomain:                     tikv.Spec.Subdomain,
			NodeSelector:                  tikv.Spec.Topology,
			InitContainers: []corev1.Container{
				{
					// TODO: support hot reload checker
					// NOTE: now sidecar cannot be restarted because of this https://github.com/kubernetes/kubernetes/pull/126525.
					Name:            v1alpha1.ContainerNamePrestopChecker,
					Image:           image.PrestopChecker.Image(preStopImage),
					ImagePullPolicy: corev1.PullIfNotPresent,
					// RestartPolicy:   ptr.To(corev1.ContainerRestartPolicyAlways),
					Command: []string{
						"/bin/sh",
						"-c",
						"cp /prestop-checker " + v1alpha1.DirNamePrestop + "/;",
						// + "sleep infinity",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      v1alpha1.VolumeNamePrestopChecker,
							MountPath: v1alpha1.DirNamePrestop,
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
						filepath.Join(v1alpha1.DirNameConfigTiKV, v1alpha1.ConfigFileName),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          v1alpha1.TiKVPortNameClient,
							ContainerPort: tikv.GetClientPort(),
						},
						{
							Name:          v1alpha1.TiKVPortNameStatus,
							ContainerPort: tikv.GetStatusPort(),
						},
					},
					VolumeMounts: mounts,
					Resources:    k8s.GetResourceRequirements(tikv.Spec.Resources),
					Lifecycle: &corev1.Lifecycle{
						// TODO: change to a real pre stop action
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"/bin/sh",
									"-c",
									t.buildPrestopCheckScript(cluster, tikv),
								},
							},
						},
					},
				},
			},
			Volumes: vols,
		},
	}

	if tikv.Spec.Overlay != nil {
		overlay.OverlayPod(pod, tikv.Spec.Overlay.Pod)
	}

	k8s.CalculateHashAndSetLabels(pod)
	return pod
}

func (*TaskPod) buildPrestopCheckScript(cluster *v1alpha1.Cluster, tikv *v1alpha1.TiKV) string {
	sb := strings.Builder{}
	sb.WriteString(v1alpha1.DirNamePrestop)
	sb.WriteString("/prestop-checker")
	sb.WriteString(" -pd ")
	sb.WriteString(cluster.Status.PD)
	sb.WriteString(" -addr ")
	sb.WriteString(kvcfg.GetAdvertiseClientURLs(tikv))

	if cluster.IsTLSClusterEnabled() {
		sb.WriteString(" -ca ")
		sb.WriteString(v1alpha1.TiKVClusterTLSMountPath)
		sb.WriteString("/ca.crt")
		sb.WriteString(" -tls ")
		sb.WriteString(v1alpha1.TiKVClusterTLSMountPath)
		sb.WriteString("/tls.crt")
		sb.WriteString(" -key ")
		sb.WriteString(v1alpha1.TiKVClusterTLSMountPath)
		sb.WriteString("/tls.key")
	}

	sb.WriteString(" > /proc/1/fd/1")

	return sb.String()
}
