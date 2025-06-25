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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/pingcap/tidb-operator/pkg/client"
	t "github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func condPVCDeclared(rtx *ReconcileContext) t.Condition {
	return t.CondFunc(func() bool {
		return len(rtx.TiBR().Spec.Volumes) != 0
	})
}

func taskPatchPVCWithOwnerReferences(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("PatchPVCWithOwnerReferences", func(ctx context.Context) t.Result {
		// get all PVC belong to TiBR
		pvcs := &corev1.PersistentVolumeClaimList{}
		err := rtx.Client().List(ctx, pvcs, &client.ListOptions{
			Namespace:     rtx.NamespacedName().Namespace,
			LabelSelector: labels.SelectorFromSet(TiBRSubResourceLabels(rtx.TiBR())),
		})
		if err != nil {
			return t.Fail().With("failed to list PVCs: %s", err.Error())
		}
		if len(pvcs.Items) < len(rtx.TiBR().Spec.Overlay.PersistentVolumeClaims) {
			return t.Retry(2 * time.Second).With("Not all PVCs created yet, retrying...")
		}
		// check if owner references existed, patch owner references if not exist
		for i := range pvcs.Items {
			it := &pvcs.Items[i]
			if it.GetOwnerReferences() == nil || len(it.GetOwnerReferences()) == 0 {
				it.SetOwnerReferences([]v1.OwnerReference{
					*v1.NewControllerRef(rtx.StatefulSet(), appsv1.SchemeGroupVersion.WithKind("StatefulSet")),
				})
				err = rtx.Client().Update(ctx, it)
				if err != nil {
					return t.Fail().With("failed to patch PVC %s with owner references: %s", it.Name, err.Error())
				}
			}
		}

		return t.Complete().With("owner references patched for PVCs")
	})
}
