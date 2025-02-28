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

package waiter

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
)

func WaitForTiCDCsHealthy(ctx context.Context, c client.Client, cg *v1alpha1.TiCDCGroup, timeout time.Duration) error {
	list := v1alpha1.TiCDCList{}
	return WaitForList(ctx, c, &list, func() error {
		if len(list.Items) != int(*cg.Spec.Replicas) {
			return fmt.Errorf("cdc %s/%s replicas %d not equal to %d", cg.Namespace, cg.Name, len(list.Items), *cg.Spec.Replicas)
		}
		for i := range list.Items {
			cdc := &list.Items[i]
			if cdc.Generation != cdc.Status.ObservedGeneration {
				return fmt.Errorf("cdc %s/%s is not synced", cdc.Namespace, cdc.Name)
			}
			if !meta.IsStatusConditionPresentAndEqual(cdc.Status.Conditions, v1alpha1.CondReady, metav1.ConditionTrue) {
				return fmt.Errorf("cdc %s/%s is not healthy", cdc.Namespace, cdc.Name)
			}
		}

		return nil
	}, timeout, client.InNamespace(cg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   cg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     cg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiCDC,
	})
}
