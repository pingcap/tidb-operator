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
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/image"
	"github.com/pingcap/tidb-operator/v2/pkg/overlay"
	"github.com/pingcap/tidb-operator/v2/pkg/reloadable"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/ticdcapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/v2/pkg/utils/map"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/third_party/kubernetes/pkg/controller/statefulset"
)

const (
	metricsPath = "/metrics"

	defaultGracefulShutdownSeconds = 30
)

func TaskPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Pod", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)

		expected := newPod(state.Cluster(), state.TiCDC())
		pod := state.Pod()
		if pod == nil {
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

		if !reloadable.CheckTiCDCPod(state.TiCDC(), pod) {
			if state.IsHealthy() || statefulset.IsPodReady(pod) {
				wait, err := preDeleteCheck(ctx, logger, state.TiCDCClient)
				if err != nil {
					return task.Fail().With("can't delete pod of ticdc: %v", err)
				}

				if wait {
					return task.Retry(task.DefaultRequeueAfter).With("wait for ticdc capture to be drained")
				}
			}
			logger.Info("will recreate the pod")
			if err := c.Delete(ctx, pod); err != nil {
				return task.Fail().With("can't delete pod of ticdc: %w", err)
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

func newPod(cluster *v1alpha1.Cluster, ticdc *v1alpha1.TiCDC) *corev1.Pod {
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
			MountPath: v1alpha1.DirPathConfigTiCDC,
		},
		{
			Name:      v1alpha1.VolumeNamePrestopChecker,
			MountPath: v1alpha1.DirPathPrestop,
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
		vols = append(vols, *coreutil.ClusterTLSVolume[scope.TiCDC](ticdc))
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameClusterTLS,
			MountPath: v1alpha1.DirPathClusterTLSTiCDC,
			ReadOnly:  true,
		})
	}

	var preStopImage *string
	if ticdc.Spec.PreStop != nil {
		preStopImage = ticdc.Spec.PreStop.Image
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ticdc.Namespace,
			Name:      coreutil.PodName[scope.TiCDC](ticdc),
			Labels: maputil.Merge(
				coreutil.PodLabels[scope.TiCDC](ticdc),
				// legacy labels in v1
				map[string]string{
					v1alpha1.LabelKeyClusterID: cluster.Status.ID,
				},
				// TODO: remove it
				k8s.LabelsK8sApp(cluster.Name, v1alpha1.LabelValComponentTiCDC),
			),
			Annotations: maputil.Merge(k8s.AnnoProm(coreutil.TiCDCPort(ticdc), metricsPath)),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ticdc, v1alpha1.SchemeGroupVersion.WithKind("TiCDC")),
			},
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: ptr.To[int64](defaultGracefulShutdownSeconds),
			Hostname:                      coreutil.PodName[scope.TiCDC](ticdc),
			Subdomain:                     ticdc.Spec.Subdomain,
			NodeSelector:                  ticdc.Spec.Topology,
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
					Lifecycle: &corev1.Lifecycle{
						// TODO: change to a real pre stop action
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"/bin/sh",
									"-c",
									buildPrestopCheckScript(cluster, ticdc),
								},
							},
						},
					},
				},
			},
			Volumes: vols,
		},
	}

	if ticdc.Spec.Overlay != nil {
		overlay.OverlayPod(pod, ticdc.Spec.Overlay.Pod)
	}

	reloadable.MustEncodeLastTiCDCTemplate(ticdc, pod)
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

func buildPrestopCheckScript(cluster *v1alpha1.Cluster, ticdc *v1alpha1.TiCDC) string {
	sb := strings.Builder{}
	sb.WriteString(v1alpha1.DirPathPrestop)
	sb.WriteString("/prestop-checker")
	sb.WriteString(" -mode ")
	sb.WriteString(" ticdc ")
	sb.WriteString(" -ticdc-status-addr ")
	sb.WriteString(coreutil.TiCDCAdvertiseURL(ticdc))

	if coreutil.IsTLSClusterEnabled(cluster) {
		sb.WriteString(" -tls ")
		sb.WriteString(" -ca ")
		sb.WriteString(v1alpha1.DirPathClusterTLSTiCDC)
		sb.WriteString("/ca.crt")
		sb.WriteString(" -cert ")
		sb.WriteString(v1alpha1.DirPathClusterTLSTiCDC)
		sb.WriteString("/tls.crt")
		sb.WriteString(" -key ")
		sb.WriteString(v1alpha1.DirPathClusterTLSTiCDC)
		sb.WriteString("/tls.key")
	}

	sb.WriteString(" > /proc/1/fd/1")

	return sb.String()
}
