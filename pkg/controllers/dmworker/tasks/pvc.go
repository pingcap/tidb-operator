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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

func PVCNewer() common.PVCNewer[*v1alpha1.DMWorker] {
	return common.PVCNewerFunc[*v1alpha1.DMWorker](
		func(cluster *v1alpha1.Cluster, dw *v1alpha1.DMWorker, fg features.Gates) []*corev1.PersistentVolumeClaim {
			var pvcs []*corev1.PersistentVolumeClaim

			// RelayVolume PVC (optional relay log storage for dm-worker)
			if relayVol := dw.Spec.RelayVolume; relayVol != nil {
				relayPVC := &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      coreutil.PersistentVolumeClaimName[scope.DMWorker](dw, relayVol.Name),
						Namespace: dw.Namespace,
						Labels:    coreutil.PersistentVolumeClaimLabels[scope.DMWorker](dw, relayVol.Name),
						OwnerReferences: []metav1.OwnerReference{
							*metav1.NewControllerRef(dw, v1alpha1.SchemeGroupVersion.WithKind("DMWorker")),
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: relayVol.Storage,
							},
						},
						StorageClassName: relayVol.StorageClassName,
					},
				}
				if fg.Enabled(meta.VolumeAttributesClass) {
					relayPVC.Spec.VolumeAttributesClassName = relayVol.VolumeAttributesClassName
				}
				if cluster.Status.ID != "" {
					relayPVC.Labels[v1alpha1.LabelKeyClusterID] = cluster.Status.ID
				}
				pvcs = append(pvcs, relayPVC)
			}

			// Additional volumes
			additionalPVCs := coreutil.PVCs[scope.DMWorker](
				cluster,
				dw,
				coreutil.EnableVAC(fg.Enabled(meta.VolumeAttributesClass)),
				coreutil.PVCPatchFunc(func(_ *v1alpha1.Volume, pvc *corev1.PersistentVolumeClaim) {
					if cluster.Status.ID != "" {
						pvc.Labels[v1alpha1.LabelKeyClusterID] = cluster.Status.ID
					}
				}),
			)
			pvcs = append(pvcs, additionalPVCs...)

			return pvcs
		},
	)
}
