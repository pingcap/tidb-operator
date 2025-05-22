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

func WaitForTiProxysHealthy(ctx context.Context, c client.Client, proxyg *v1alpha1.TiProxyGroup, timeout time.Duration) error {
	list := v1alpha1.TiProxyList{}
	return WaitForList(ctx, c, &list, func() error {
		if len(list.Items) != int(*proxyg.Spec.Replicas) {
			return fmt.Errorf("db %s/%s replicas %d not equal to %d", proxyg.Namespace, proxyg.Name, len(list.Items), *proxyg.Spec.Replicas)
		}
		for i := range list.Items {
			db := &list.Items[i]
			if db.Generation != db.Status.ObservedGeneration {
				return fmt.Errorf("db %s/%s is not synced", db.Namespace, db.Name)
			}
			if !meta.IsStatusConditionPresentAndEqual(db.Status.Conditions, v1alpha1.CondReady, metav1.ConditionTrue) {
				return fmt.Errorf("db %s/%s is not healthy", db.Namespace, db.Name)
			}
		}

		return nil
	}, timeout, client.InNamespace(proxyg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   proxyg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     proxyg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiProxy,
	})
}
