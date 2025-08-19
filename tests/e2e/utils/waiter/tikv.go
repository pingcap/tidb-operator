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

func EvictLeaderBeforeStoreIsRemoving(deleting int) func(kv *v1alpha1.TiKV) (bool, error) {
	done := map[string]struct{}{}
	return func(kv *v1alpha1.TiKV) (bool, error) {
		// Check if the TiKV is being deleted (has deletion timestamp)
		if !kv.GetDeletionTimestamp().IsZero() {
			// TiKV is being deleted, check if leaders are evicted
			if meta.IsStatusConditionTrue(kv.Status.Conditions, v1alpha1.TiKVCondLeadersEvicted) {
				// Leaders are evicted
				done[kv.GetName()] = struct{}{}
				if len(done) == deleting {
					return true, nil
				}
			}

			// Check timeout for deletion process
			delTime := kv.GetDeletionTimestamp()
			if delTime.Add(time.Minute * 5).Before(time.Now()) {
				// Timeout, consider it as done
				done[kv.GetName()] = struct{}{}
				if len(done) == deleting {
					return true, nil
				}
			}

			// TiKV is being deleted but leaders are not evicted yet - continue waiting
			return false, nil
		}

		// Check if the store is in removing state
		if kv.Status.State == v1alpha1.StoreStateRemoving || kv.Status.State == v1alpha1.StoreStateRemoved {
			if meta.IsStatusConditionTrue(kv.Status.Conditions, v1alpha1.TiKVCondLeadersEvicted) {
				// Leaders are evicted
				done[kv.GetName()] = struct{}{}
				if len(done) == deleting {
					return true, nil
				}
			}

			// Check if store is removed (no need to wait for leader eviction)
			if kv.Status.State == v1alpha1.StoreStateRemoved {
				done[kv.GetName()] = struct{}{}
				if len(done) == deleting {
					return true, nil
				}
			}

			return false, fmt.Errorf("store state is %v but leaders are not all evicted", kv.Status.State)
		}

		return false, nil
	}
}
