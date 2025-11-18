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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/image"
	"github.com/pingcap/tidb-operator/v2/pkg/overlay"
	"github.com/pingcap/tidb-operator/v2/pkg/reloadable"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	tsom "github.com/pingcap/tidb-operator/v2/pkg/timanager/tso"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/v2/pkg/utils/map"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/third_party/kubernetes/pkg/controller/statefulset"
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
				return task.Fail().With("can't create pod of pd: %v", err)
			}
			state.SetPod(expected)
			return task.Complete().With("pod is synced")
		}

		if !reloadable.CheckTSOPod(obj, pod) {
			if statefulset.IsPodReady(pod) {
				if !state.CacheSynced || state.TSOMemberNotFound {
					return task.Fail().With("wait until cache is synced and tso member is registered")
				}
				wait, err := preDeleteCheck(
					ctx,
					logger,
					state.TSOClient,
					state.Object(),
					state.InstanceSlice(),
					state.IsLeader,
				)
				if err != nil {
					return task.Fail().With("pre delete pod of tso failed: %v", err)
				}

				if wait {
					return task.Wait().With("wait for tso leader being transferred")
				}
			}
			logger.Info("will delete the pod to recreate", "name", pod.Name, "namespace", pod.Namespace, "UID", pod.UID)

			if err := c.Delete(ctx, pod); err != nil {
				return task.Fail().With("can't delete pod of tso: %v", err)
			}

			state.DeletePod(pod)

			return task.Wait().With("pod is deleting")
		}

		logger.Info("will update the pod in place")

		if err := c.Apply(ctx, expected); err != nil {
			return task.Fail().With("can't apply pod of tso: %v", err)
		}
		state.SetPod(expected)

		return task.Complete().With("pod is synced")
	})
}

// TODO: simplify it by a better way to send args.
func preDeleteCheck(
	ctx context.Context,
	logger logr.Logger,
	tsoc tsom.TSOClient,
	tso *v1alpha1.TSO,
	peers []*v1alpha1.TSO,
	isLeader bool,
) (bool, error) {
	if len(peers) == 1 {
		logger.Info("no need to transfer leader because there is only one tso")
		return false, nil
	}

	if isLeader {
		peer := coreutil.LongestReadyPeer[scope.TSO](tso, peers)
		if peer == nil {
			return false, fmt.Errorf("no healthy transferee available")
		}

		logger.Info("try to transfer leader", "from", tso.Name, "to", peer.Name)

		if err := tsoc.Underlay().TransferTSOLeader(ctx, peer.Name); err != nil {
			return false, fmt.Errorf("transfer leader failed: %w", err)
		}

		return true, nil
	}

	return false, nil
}

func newPod(cluster *v1alpha1.Cluster, tso *v1alpha1.TSO, _ features.Gates) *corev1.Pod {
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: coreutil.PodName[scope.TSO](tso),
					},
				},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      v1alpha1.VolumeNameConfig,
			MountPath: v1alpha1.DirPathConfigTSO,
		},
	}

	for i := range tso.Spec.Volumes {
		vol := &tso.Spec.Volumes[i]
		name := VolumeName(vol.Name)
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PersistentVolumeClaimName(coreutil.PodName[scope.TSO](tso), vol.Name),
				},
			},
		})
		for i := range vol.Mounts {
			mount := VolumeMount(name, &vol.Mounts[i])
			mounts = append(mounts, *mount)
		}
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		vols = append(vols, *coreutil.ClusterTLSVolume[scope.TSO](tso))
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameClusterTLS,
			MountPath: v1alpha1.DirPathClusterTLSTSO,
			ReadOnly:  true,
		})
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   tso.Namespace,
			Name:        coreutil.PodName[scope.TSO](tso),
			Labels:      coreutil.PodLabels[scope.TSO](tso),
			Annotations: maputil.Merge(k8s.AnnoProm(coreutil.TSOClientPort(tso), metricsPath)),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tso, v1alpha1.SchemeGroupVersion.WithKind("TSO")),
			},
		},
		Spec: corev1.PodSpec{
			Hostname:     coreutil.PodName[scope.TSO](tso),
			Subdomain:    tso.Spec.Subdomain,
			NodeSelector: tso.Spec.Topology,
			Containers: []corev1.Container{
				{
					Name:            v1alpha1.ContainerNameTSO,
					Image:           image.TSO.Image(tso.Spec.Image, tso.Spec.Version),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"/pd-server",
						"services",
						"tso",
						"--config",
						filepath.Join(v1alpha1.DirPathConfigTSO, v1alpha1.FileNameConfig),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          v1alpha1.TSOPortNameClient,
							ContainerPort: coreutil.TSOClientPort(tso),
						},
					},
					VolumeMounts: mounts,
					Resources:    k8s.GetResourceRequirements(tso.Spec.Resources),
				},
			},
			Volumes: vols,
		},
	}

	if tso.Spec.Overlay != nil {
		overlay.OverlayPod(pod, tso.Spec.Overlay.Pod)
	}

	reloadable.MustEncodeLastTSOTemplate(tso, pod)
	return pod
}
