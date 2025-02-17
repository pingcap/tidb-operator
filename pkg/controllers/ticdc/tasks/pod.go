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
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/image"
	"github.com/pingcap/tidb-operator/pkg/overlay"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/ticdcapi/v1"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/statefulset"
)

const (
	metricsPath = "/metrics"
)

func TaskPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Pod", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)

		expected := newPod(state)
		if state.Pod() == nil {
			// Pod needs PD address as a parameter
			if state.Cluster().Status.PD == "" {
				return task.Wait().With("wait until pd's status is set")
			}
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't create pod of ticdc: %w", err)
			}

			state.SetPod(expected)
			return task.Complete().With("pod is created")
		}

		res := k8s.ComparePods(state.Pod(), expected)
		curHash, expectHash := state.Pod().Labels[v1alpha1.LabelKeyConfigHash], expected.Labels[v1alpha1.LabelKeyConfigHash]
		configChanged := curHash != expectHash
		logger.Info("compare pod", "result", res, "configChanged", configChanged, "currentConfigHash", curHash, "expectConfigHash", expectHash)

		if res == k8s.CompareResultRecreate || (configChanged &&
			state.TiCDC().Spec.UpdateStrategy.Config == v1alpha1.ConfigUpdateStrategyRestart) {
			if state.Healthy || statefulset.IsPodReady(state.Pod()) {
				wait, err := preDeleteCheck(ctx, logger, state.TiCDCClient)
				if err != nil {
					return task.Fail().With("can't delete pod of ticdc: %v", err)
				}

				if wait {
					return task.Wait().With("wait for ticdc caputure to be drained")
				}
			}
			logger.Info("will recreate the pod")
			if err := c.Delete(ctx, state.Pod()); err != nil {
				return task.Fail().With("can't delete pod of ticdc: %w", err)
			}

			state.PodIsTerminating = true
			return task.Wait().With("pod is deleting")
		} else if res == k8s.CompareResultUpdate {
			logger.Info("will update the pod in place")
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't apply pod of ticdc: %w", err)
			}

			state.SetPod(expected)
		}

		return task.Complete().With("pod is synced")
	})
}

func newPod(state *ReconcileContext) *corev1.Pod {
	cluster := state.Cluster()
	ticdc := state.TiCDC()
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: coreutil.PodName[scope.TiCDC](ticdc),
					},
				},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      v1alpha1.VolumeNameConfig,
			MountPath: v1alpha1.DirPathConfigTiCDC,
		},
	}

	for i := range ticdc.Spec.Volumes {
		vol := &ticdc.Spec.Volumes[i]
		name := VolumeName(vol.Name)
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PersistentVolumeClaimName(coreutil.PodName[scope.TiCDC](ticdc), vol.Name),
				},
			},
		})

		for i := range vol.Mounts {
			mount := &vol.Mounts[i]
			vm := VolumeMount(name, mount)
			mounts = append(mounts, *vm)
		}
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		vols = append(vols, corev1.Volume{
			Name: v1alpha1.VolumeNameClusterTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: coreutil.TLSClusterSecretName[scope.TiCDC](ticdc),
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameClusterTLS,
			MountPath: v1alpha1.DirPathClusterTLSTiCDC,
			ReadOnly:  true,
		})
	}

	for _, secretName := range ticdc.Spec.Security.TLS.SinkTLSSecretNames {
		vols = append(vols, corev1.Volume{
			Name: secretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      secretName,
			MountPath: fmt.Sprintf("%s/%s", v1alpha1.DirPathTiCDCSinkTLS, secretName),
			ReadOnly:  true,
		})
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ticdc.Namespace,
			Name:      coreutil.PodName[scope.TiCDC](ticdc),
			Labels: maputil.Merge(ticdc.Labels, map[string]string{
				v1alpha1.LabelKeyInstance:   ticdc.Name,
				v1alpha1.LabelKeyConfigHash: state.ConfigHash,
				v1alpha1.LabelKeyClusterID:  cluster.Status.ID,
			}, k8s.LabelsK8sApp(cluster.Name, v1alpha1.LabelValComponentTiCDC)),
			Annotations: maputil.Merge(ticdc.GetAnnotations(), k8s.AnnoProm(coreutil.TiCDCPort(ticdc), metricsPath)),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ticdc, v1alpha1.SchemeGroupVersion.WithKind("TiCDC")),
			},
		},
		Spec: corev1.PodSpec{
			Hostname:     coreutil.PodName[scope.TiCDC](ticdc),
			Subdomain:    ticdc.Spec.Subdomain,
			NodeSelector: ticdc.Spec.Topology,
			Containers: []corev1.Container{
				{
					Name:            v1alpha1.ContainerNameTiCDC,
					Image:           image.TiCDC.Image(ticdc.Spec.Image, ticdc.Spec.Version),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"/cdc",
						"server",
						"--pd",
						cluster.Status.PD,
						"--config",
						filepath.Join(v1alpha1.DirPathConfigTiCDC, v1alpha1.FileNameConfig),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          v1alpha1.TiCDCPortName,
							ContainerPort: coreutil.TiCDCPort(ticdc),
						},
					},
					VolumeMounts: mounts,
					Resources:    k8s.GetResourceRequirements(ticdc.Spec.Resources),
				},
			},
			Volumes: vols,
		},
	}

	if ticdc.Spec.Overlay != nil {
		overlay.OverlayPod(pod, ticdc.Spec.Overlay.Pod)
	}

	k8s.CalculateHashAndSetLabels(pod)
	return pod
}

func preDeleteCheck(
	ctx context.Context,
	logger logr.Logger,
	cdcClient ticdcapi.TiCDCClient,
) (bool, error) {
	// TODO(csuzhangxc): update to use prestop-checker with `gracefulShutdownTimeout`

	// both `ResignOwner` and `DrainCapture` will check the capture count
	resigned, err := cdcClient.ResignOwner(ctx)
	if err != nil {
		return false, err
	}
	if !resigned {
		logger.Info("wait for resigning owner")
		return true, nil
	}

	tableCount, err := cdcClient.DrainCapture(ctx)
	if err != nil {
		return false, err
	}
	if tableCount > 0 {
		logger.Info("wait for draining capture", "tableCount", tableCount)
		return true, nil
	}

	return false, nil
}
