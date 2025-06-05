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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
)

func WaitForTiKVsHealthy(ctx context.Context, c client.Client, kvg *v1alpha1.TiKVGroup, timeout time.Duration) error {
	list := v1alpha1.TiKVList{}
	return WaitForList(ctx, c, &list, func() error {
		if len(list.Items) != int(*kvg.Spec.Replicas) {
			return fmt.Errorf("kv %s/%s replicas %d not equal to %d", kvg.Namespace, kvg.Name, len(list.Items), *kvg.Spec.Replicas)
		}
		for i := range list.Items {
			kv := &list.Items[i]
			if err := checkInstanceStatus(v1alpha1.LabelValComponentTiKV, kv.Name, kv.Namespace, kv.Generation, kv.Status.CommonStatus); err != nil {
				return err
			}
		}

		return nil
	}, timeout, client.InNamespace(kvg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   kvg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     kvg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
	})
}
