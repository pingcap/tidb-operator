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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/overlay"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/pkg/volumes"
)

func TaskPVC(state *ReconcileContext, logger logr.Logger, c client.Client, vm volumes.ModifierFactory) task.Task {
	return task.NameTaskFunc("PVC", func(ctx context.Context) task.Result {
		cluster := state.Cluster()
		tiflash := state.TiFlash()
		pvcs := newPVCs(cluster, tiflash, state.Store, state.FeatureGates())
		if wait, err := volumes.SyncPVCs(ctx, c, pvcs, vm.New(state.FeatureGates()), logger); err != nil {
			return task.Fail().With("failed to sync pvcs: %w", err)
		} else if wait {
			return task.Retry(task.DefaultRequeueAfter).With("waiting for pvcs to be synced")
		}

		return task.Complete().With("pvcs are synced")
	})
}

func newPVCs(cluster *v1alpha1.Cluster, tiflash *v1alpha1.TiFlash, store *pdv1.Store, fg features.Gates) []*corev1.PersistentVolumeClaim {
	pvcs := make([]*corev1.PersistentVolumeClaim, 0, len(tiflash.Spec.Volumes))
	nameToIndex := map[string]int{}
	for i := range tiflash.Spec.Volumes {
		vol := &tiflash.Spec.Volumes[i]
		nameToIndex[vol.Name] = i
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PersistentVolumeClaimName(coreutil.PodName[scope.TiFlash](tiflash), vol.Name),
				Namespace: tiflash.Namespace,
				Labels: maputil.Merge(
					coreutil.PersistentVolumeClaimLabels[scope.TiFlash](tiflash, vol.Name),
					// TODO: remove it
					k8s.LabelsK8sApp(cluster.Name, v1alpha1.LabelValComponentTiFlash),
				),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(tiflash, v1alpha1.SchemeGroupVersion.WithKind("TiFlash")),
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
				StorageClassName: vol.StorageClassName,
			},
		}

		// Only set VolumeAttributesClassName when the feature gate is enabled
		if fg.Enabled(meta.VolumeAttributesClass) {
			pvc.Spec.VolumeAttributesClassName = vol.VolumeAttributesClassName
		}
		// legacy labels in v1
		if cluster.Status.ID != "" {
			pvc.Labels[v1alpha1.LabelKeyClusterID] = cluster.Status.ID
		}
		if store != nil && store.ID != "" {
			pvc.Labels[v1alpha1.LabelKeyStoreID] = store.ID
		}

		pvcs = append(pvcs, pvc)
	}

	if tiflash.Spec.Overlay != nil {
		for _, o := range tiflash.Spec.Overlay.PersistentVolumeClaims {
			index, ok := nameToIndex[o.Name]
			if !ok {
				// TODO(liubo02): it should be validated
				panic("vol name" + o.Name + "doesn't exist")
			}

			overlay.OverlayPersistentVolumeClaim(pvcs[index], &o.PersistentVolumeClaim)
		}
	}

	return pvcs
}
