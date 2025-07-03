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
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/image"
	"github.com/pingcap/tidb-operator/pkg/overlay"
	"github.com/pingcap/tidb-operator/pkg/reloadable"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/statefulset"
)

const (
	defaultReadinessProbeInitialDelaySeconds = 5

	metricsPath = "/metrics"
)

func TaskPod(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Pod", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		expected := newPod(state.Cluster(), state.PD(), state.FeatureGates(), state.ClusterID, state.MemberID)
		pod := state.Pod()
		if pod == nil {
			// We have to refresh cache of members to make sure a pd without pod is unhealthy.
			// If the healthy info is out of date, the operator may mark this pd up-to-date unexpectedly
			// and begin to update the next PD.
			if state.IsHealthy() {
				state.PDClient.Members().Refresh()
				return task.Wait().With("wait until pd's status becomes unhealthy")
			}
			if err := c.Apply(ctx, expected); err != nil {
				return task.Fail().With("can't create pod of pd: %v", err)
			}
			state.SetPod(expected)
			return task.Complete().With("pod is synced")
		}

		if !reloadable.CheckPDPod(state.PD(), pod) {
			// NOTE: both rtx.Healthy and rtx.Pod are not always newest
			// So pre delete check may also be skipped in some cases, for example,
			// the PD is just started.
			if state.IsHealthy() || statefulset.IsPodReady(pod) {
				wait, err := preDeleteCheck(ctx, logger, state.PDClient, state.PD(), state.PDSlice(), state.IsLeader)
				if err != nil {
					return task.Fail().With("can't delete pod of pd: %v", err)
				}

				if wait {
					return task.Wait().With("wait for pd leader being transferred")
				}
			}

			logger.Info("will delete the pod to recreate", "name", pod.Name, "namespace", pod.Namespace, "UID", pod.UID)

			if err := c.Delete(ctx, pod); err != nil {
				return task.Fail().With("can't delete pod of pd: %v", err)
			}

			state.DeletePod(pod)

			return task.Wait().With("pod is deleting")
		}

		logger.Info("will update the pod in place")
		if err := c.Apply(ctx, expected); err != nil {
			return task.Fail().With("can't apply pod of pd: %v", err)
		}
		state.SetPod(expected)

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

	if len(peers) == 1 {
		logger.Info("no need to transfer leader because there is only one pd")
		return false, nil
	}

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

func newPod(cluster *v1alpha1.Cluster, pd *v1alpha1.PD, g features.Gates, clusterID, memberID string) *corev1.Pod {
	vols := []corev1.Volume{
		{
			Name: v1alpha1.VolumeNameConfig,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: coreutil.PodName[scope.PD](pd),
					},
				},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      v1alpha1.VolumeNameConfig,
			MountPath: v1alpha1.DirPathConfigPD,
		},
	}

	for i := range pd.Spec.Volumes {
		vol := &pd.Spec.Volumes[i]
		name := VolumeName(vol.Name)
		vols = append(vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PersistentVolumeClaimName(coreutil.PodName[scope.PD](pd), vol.Name),
				},
			},
		})
		for i := range vol.Mounts {
			mount := VolumeMount(name, &vol.Mounts[i])
			mounts = append(mounts, *mount)
		}
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		vols = append(vols, corev1.Volume{
			Name: v1alpha1.VolumeNameClusterTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: coreutil.TLSClusterSecretName[scope.PD](pd),
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      v1alpha1.VolumeNameClusterTLS,
			MountPath: v1alpha1.DirPathClusterTLSPD,
			ReadOnly:  true,
		})
	}

	var cmd []string
	cmd = append(cmd, "/pd-server")
	if pd.Spec.Mode == v1alpha1.PDModeMS {
		cmd = append(cmd, "services", "api")
	}
	cmd = append(cmd, "--config", filepath.Join(v1alpha1.DirPathConfigPD, v1alpha1.FileNameConfig))

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pd.Namespace,
			Name:      coreutil.PodName[scope.PD](pd),
			Labels: maputil.Merge(
				coreutil.PodLabels[scope.PD](pd),
				// legacy labels in v1
				map[string]string{
					v1alpha1.LabelKeyClusterID: clusterID,
					v1alpha1.LabelKeyMemberID:  memberID,
				},
				// TODO: remove it
				k8s.LabelsK8sApp(cluster.Name, v1alpha1.LabelValComponentPD),
			),
			Annotations: maputil.Merge(k8s.AnnoProm(coreutil.PDClientPort(pd), metricsPath)),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(pd, v1alpha1.SchemeGroupVersion.WithKind("PD")),
			},
		},
		Spec: corev1.PodSpec{
			Hostname:     coreutil.PodName[scope.PD](pd),
			Subdomain:    pd.Spec.Subdomain,
			NodeSelector: pd.Spec.Topology,
			Containers: []corev1.Container{
				{
					Name:            v1alpha1.ContainerNamePD,
					Image:           image.PD.Image(pd.Spec.Image, pd.Spec.Version),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         cmd,
					Ports: []corev1.ContainerPort{
						{
							Name:          v1alpha1.PDPortNameClient,
							ContainerPort: coreutil.PDClientPort(pd),
						},
						{
							Name:          v1alpha1.PDPortNamePeer,
							ContainerPort: coreutil.PDPeerPort(pd),
						},
					},
					VolumeMounts: mounts,
					Resources:    k8s.GetResourceRequirements(pd.Spec.Resources),
				},
			},
			Volumes: vols,
		},
	}

	if !g.Enabled(metav1alpha1.DisablePDDefaultReadinessProbe) {
		pod.Spec.Containers[0].ReadinessProbe = buildPDReadinessProbe(coreutil.PDClientPort(pd))
	}
	if g.Enabled(metav1alpha1.UsePDReadyAPI) {
		pod.Spec.Containers[0].ReadinessProbe = buildPDReadinessProbeWithReadyAPI(cluster, coreutil.PDClientPort(pd))
	}

	if pd.Spec.Overlay != nil {
		overlay.OverlayPod(pod, pd.Spec.Overlay.Pod)
	}

	reloadable.MustEncodeLastPDTemplate(pd, pod)
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

func buildPDReadinessProbeWithReadyAPI(cluster *v1alpha1.Cluster, port int32) *corev1.Probe {
	tlsClusterEnabled := coreutil.IsTLSClusterEnabled(cluster)

	scheme := "http"
	if tlsClusterEnabled {
		scheme = "https"
	}

	readinessURL := fmt.Sprintf("%s://127.0.0.1:%d/pd/api/v2/ready", scheme, port)
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
		cacert := path.Join(v1alpha1.DirPathClusterTLSPD, corev1.ServiceAccountRootCAKey)
		cert := path.Join(v1alpha1.DirPathClusterTLSPD, corev1.TLSCertKey)
		key := path.Join(v1alpha1.DirPathClusterTLSPD, corev1.TLSPrivateKeyKey)
		command = append(command, "--cacert", cacert, "--cert", cert, "--key", key)
	}

	return &corev1.Probe{
		InitialDelaySeconds: defaultReadinessProbeInitialDelaySeconds,
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: command,
			},
		},
	}
}
