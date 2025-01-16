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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/image"
	"github.com/pingcap/tidb-operator/pkg/overlay"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/statefulset"
)

const (
	defaultReadinessProbeInitialDelaySeconds = 5
)

func TaskPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Pod", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		expected := newPod(state.Cluster(), state.PD(), state.ConfigHash)
		if state.Pod() == nil {
			// We have to refresh cache of members to make sure a pd without pod is unhealthy.
			// If the healthy info is out of date, the operator may mark this pd up-to-date unexpectedly
			// and begin to update the next PD.
			if state.Healthy {
				state.PDClient.Members().Refresh()
				return task.Wait().With("wait until pd's status becomes unhealthy")
			}
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't create pod of pd: %v", err)
			}
			state.SetPod(expected)
			return task.Complete().With("pod is synced")
		}

		res := k8s.ComparePods(state.Pod(), expected)
		curHash, expectHash := state.Pod().Labels[v1alpha1.LabelKeyConfigHash], expected.Labels[v1alpha1.LabelKeyConfigHash]
		configChanged := curHash != expectHash
		logger.Info("compare pod", "result", res, "configChanged", configChanged, "currentConfigHash", curHash, "expectConfigHash", expectHash)

		if res == k8s.CompareResultRecreate ||
			(configChanged && state.PD().Spec.UpdateStrategy.Config == v1alpha1.ConfigUpdateStrategyRestart) {
			// NOTE: both rtx.Healthy and rtx.Pod are not always newest
			// So pre delete check may also be skipped in some cases, for example,
			// the PD is just started.
			if state.Healthy || statefulset.IsPodReady(state.Pod()) {
				wait, err := preDeleteCheck(ctx, logger, state.PDClient, state.PD(), state.PDSlice(), state.IsLeader)
				if err != nil {
					return task.Fail().With("can't delete pod of pd: %v", err)
				}

				if wait {
					return task.Wait().With("wait for pd leader being transferred")
				}
			}

			logger.Info("will delete the pod to recreate", "name", state.Pod().Name, "namespace", state.Pod().Namespace, "UID", state.Pod().UID)

			if err := c.Delete(ctx, state.Pod()); err != nil {
				return task.Fail().With("can't delete pod of pd: %v", err)
			}

			state.PodIsTerminating = true

			return task.Wait().With("pod is deleting")
		} else if res == k8s.CompareResultUpdate {
			logger.Info("will update the pod in place")
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't apply pod of pd: %v", err)
			}
			state.SetPod(expected)
		}

		return task.Complete().With("pod is synced")
	})
}

func preDeleteCheck(
	ctx context.Context,
	logger logr.Logger,
	pdc pdm.PDClient,
	pd *v1alpha1.PD,
	peers []*v1alpha1.PD,
	isLeader bool,
) (bool, error) {
	// TODO: add quorum check. After stopping this pd, quorum should not be lost

	if isLeader {
		peer := LongestHealthPeer(pd, peers)
		if peer == "" {
			return false, fmt.Errorf("no healthy transferee available")
		}

		logger.Info("try to transfer leader", "from", pd.Name, "to", peer)

		if err := pdc.Underlay().TransferPDLeader(ctx, peer); err != nil {
			return false, fmt.Errorf("transfer leader failed: %w", err)
		}

		return true, nil
	}

	return false, nil
}

func newPod(cluster *v1alpha1.Cluster, pd *v1alpha1.PD, configHash string) *corev1.Pod {
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: pd.PodName(),
					},
				},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      v1alpha1.VolumeNameConfig,
			MountPath: v1alpha1.DirNameConfigPD,
		},
	}

	for i := range pd.Spec.Volumes {
		vol := &pd.Spec.Volumes[i]
		name := VolumeName(vol.Name)
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PersistentVolumeClaimName(pd.PodName(), vol.Name),
				},
			},
		})
		for i := range vol.Mounts {
			mount := VolumeMount(name, &vol.Mounts[i])
			mounts = append(mounts, *mount)
		}
	}

	if cluster.IsTLSClusterEnabled() {
		vols = append(vols, corev1.Volume{
			Name: v1alpha1.PDClusterTLSVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: pd.TLSClusterSecretName(),
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.PDClusterTLSVolumeName,
			MountPath: v1alpha1.PDClusterTLSMountPath,
			ReadOnly:  true,
		})
	}

	anno := maputil.Copy(pd.GetAnnotations())
	// TODO: should not inherit all labels and annotations into pod
	delete(anno, v1alpha1.AnnoKeyInitialClusterNum)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pd.Namespace,
			Name:      pd.PodName(),
			Labels: maputil.Merge(pd.Labels, map[string]string{
				v1alpha1.LabelKeyInstance:   pd.Name,
				v1alpha1.LabelKeyConfigHash: configHash,
			}),
			Annotations: anno,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(pd, v1alpha1.SchemeGroupVersion.WithKind("PD")),
			},
		},
		Spec: corev1.PodSpec{
			Hostname:     pd.PodName(),
			Subdomain:    pd.Spec.Subdomain,
			NodeSelector: pd.Spec.Topology,
			Containers: []corev1.Container{
				{
					Name:            v1alpha1.ContainerNamePD,
					Image:           image.PD.Image(pd.Spec.Image, pd.Spec.Version),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"/pd-server",
						"--config",
						filepath.Join(v1alpha1.DirNameConfigPD, v1alpha1.ConfigFileName),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          v1alpha1.PDPortNameClient,
							ContainerPort: pd.GetClientPort(),
						},
						{
							Name:          v1alpha1.PDPortNamePeer,
							ContainerPort: pd.GetPeerPort(),
						},
					},
					VolumeMounts:   mounts,
					Resources:      k8s.GetResourceRequirements(pd.Spec.Resources),
					ReadinessProbe: buildPDReadinessProbe(pd.GetClientPort()),
				},
			},
			Volumes: vols,
		},
	}

	if pd.Spec.Overlay != nil {
		overlay.OverlayPod(pod, pd.Spec.Overlay.Pod)
	}

	k8s.CalculateHashAndSetLabels(pod)
	return pod
}

func buildPDReadinessProbe(port int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt32(port),
			},
		},
		InitialDelaySeconds: defaultReadinessProbeInitialDelaySeconds,
	}
}
