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

func WaitForTSOsHealthy(ctx context.Context, c client.Client, tg *v1alpha1.TSOGroup, timeout time.Duration) error {
	list := v1alpha1.TSOList{}
	return WaitForList(ctx, c, &list, func() error {
		if len(list.Items) != int(*tg.Spec.Replicas) {
			return fmt.Errorf("tso %s/%s replicas %d not equal to %d", tg.Namespace, tg.Name, len(list.Items), *tg.Spec.Replicas)
		}
		for i := range list.Items {
			tso := &list.Items[i]
			if tso.Generation != tso.Status.ObservedGeneration {
				return fmt.Errorf("tso %s/%s is not synced", tso.Namespace, tso.Name)
			}
			if !meta.IsStatusConditionPresentAndEqual(tso.Status.Conditions, v1alpha1.CondReady, metav1.ConditionTrue) {
				return fmt.Errorf("tso %s/%s is not healthy", tso.Namespace, tso.Name)
			}
		}

		return nil
	}, timeout, client.InNamespace(tg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   tg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     tg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTSO,
	})
}
