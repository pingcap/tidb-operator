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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/pkg/volumes"
)

func TaskPVC(state *ReconcileContext, logger logr.Logger, c client.Client, vm volumes.Modifier) task.Task {
	return task.NameTaskFunc("PVC", func(ctx context.Context) task.Result {
		pvcs := newPVCs(state)
		if wait, err := volumes.SyncPVCs(ctx, c, pvcs, vm, logger); err != nil {
			return task.Fail().With("failed to sync pvcs: %w", err)
		} else if wait {
			return task.Complete().With("waiting for pvcs to be synced")
		}

		return task.Complete().With("pvcs are synced")
	})
}

func newPVCs(state *ReconcileContext) []*corev1.PersistentVolumeClaim {
	cluster := state.Cluster()
	tidb := state.TiDB()
	pvcs := make([]*corev1.PersistentVolumeClaim, 0, len(tidb.Spec.Volumes))
	for i := range tidb.Spec.Volumes {
		vol := &tidb.Spec.Volumes[i]
		pvcs = append(pvcs, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PersistentVolumeClaimName(coreutil.PodName[scope.TiDB](tidb), vol.Name),
				Namespace: tidb.Namespace,
				Labels: maputil.Merge(tidb.Labels, map[string]string{
					v1alpha1.LabelKeyInstance:   tidb.Name,
					v1alpha1.LabelKeyClusterID:  cluster.Status.ID,
					v1alpha1.LabelKeyVolumeName: vol.Name,
				}, k8s.LabelsK8sApp(cluster.Name, v1alpha1.LabelValComponentTiDB)),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(tidb, v1alpha1.SchemeGroupVersion.WithKind("TiDB")),
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: vol.Storage,
					},
				},
				StorageClassName:          vol.StorageClassName,
				VolumeAttributesClassName: vol.VolumeAttributesClassName,
			},
		})
	}

	return pvcs
}
