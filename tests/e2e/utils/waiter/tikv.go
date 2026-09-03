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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

func WaitForTiKVsHealthy(ctx context.Context, c client.Client, kvg *v1alpha1.TiKVGroup, timeout time.Duration) error {
	list := v1alpha1.TiKVList{}
	return WaitForList(ctx, c, &list, func() error {
		errs := []error{}
		if len(list.Items) != int(*kvg.Spec.Replicas) {
			errs = append(errs, fmt.Errorf("kv %s/%s replicas %d not equal to %d", kvg.Namespace, kvg.Name, len(list.Items), *kvg.Spec.Replicas))
		}
		for i := range list.Items {
			kv := &list.Items[i]
			if err := checkInstanceStatus(v1alpha1.LabelValComponentTiKV, kv.Name, kv.Namespace, kv.Generation, kv.Status.CommonStatus); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.NewAggregate(errs)
	}, timeout, client.InNamespace(kvg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   kvg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     kvg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
	})
}

// WatchUntilTiKVsRestartedAfterCacheTTL watches TiKV leader eviction and recovery
// transitions and verifies every recreated Pod observes the configured cache TTL.
func WatchUntilTiKVsRestartedAfterCacheTTL(
	ctx context.Context,
	c client.Client,
	kvg *v1alpha1.TiKVGroup,
	timeout time.Duration,
	synced chan struct{},
) error {
	replicas := 0
	if kvg.Spec.Replicas != nil {
		replicas = int(*kvg.Spec.Replicas)
	}
	cacheTTLSeconds := int64(-1)
	if kvg.Spec.CacheTTLSeconds != nil {
		cacheTTLSeconds = *kvg.Spec.CacheTTLSeconds
	}

	leadersEvictedAt := make(map[string]time.Time, replicas)
	comparedTiKVs := make(map[string]struct{}, replicas)
	return WatchUntilInstanceList[scope.TiKVGroup](ctx, c, kvg, func(tikv *v1alpha1.TiKV) (bool, error) {
		cond := meta.FindStatusCondition(tikv.Status.Conditions, v1alpha1.TiKVCondLeadersEvicted)
		if cond == nil {
			return false, nil
		}

		if cond.Status == metav1.ConditionTrue {
			if tikv.Spec.CacheTTLSeconds == nil {
				return false, fmt.Errorf("tikv %s/%s has no cacheTTLSeconds", tikv.Namespace, tikv.Name)
			}
			if *tikv.Spec.CacheTTLSeconds != cacheTTLSeconds {
				return false, fmt.Errorf("tikv %s/%s cacheTTLSeconds is %d, want %d", tikv.Namespace, tikv.Name, *tikv.Spec.CacheTTLSeconds, cacheTTLSeconds)
			}
			if cond.LastTransitionTime.IsZero() {
				return false, fmt.Errorf("tikv %s/%s has no leader eviction transition time", tikv.Namespace, tikv.Name)
			}
			if _, recorded := leadersEvictedAt[tikv.Name]; !recorded {
				leadersEvictedAt[tikv.Name] = cond.LastTransitionTime.Time
			}
			return false, nil
		}

		if cond.Status != metav1.ConditionFalse || cond.Reason != v1alpha1.ReasonNotEvicted {
			return false, nil
		}
		if _, compared := comparedTiKVs[tikv.Name]; compared {
			return false, nil
		}
		evictedAt, recorded := leadersEvictedAt[tikv.Name]
		if !recorded {
			return false, nil
		}

		pod, err := apicall.GetPod[scope.TiKV](ctx, c, tikv)
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		earliestRestartAt := evictedAt.Add(time.Duration(cacheTTLSeconds) * time.Second)
		if pod.CreationTimestamp.Time.Before(earliestRestartAt) {
			return false, fmt.Errorf(
				"tikv pod %s/%s was recreated at %s before cache TTL expired at %s",
				pod.Namespace,
				pod.Name,
				pod.CreationTimestamp,
				earliestRestartAt,
			)
		}

		comparedTiKVs[tikv.Name] = struct{}{}
		return len(comparedTiKVs) == replicas, nil
	}, timeout, synced)
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

func WaitForTiKVOfflineCompleted(expectTiKV *v1alpha1.TiKV) func(kv *v1alpha1.TiKV) (bool, error) {
	// Capture the identity to avoid races when the caller reuses expectTiKV for API reads.
	ns := expectTiKV.GetNamespace()
	name := expectTiKV.GetName()
	uid := expectTiKV.GetUID()
	return func(kv *v1alpha1.TiKV) (bool, error) {
		if kv.GetName() != name || kv.GetNamespace() != ns || kv.GetUID() != uid {
			return false, nil
		}
		if meta.IsStatusConditionPresentAndEqual(kv.Status.Conditions, v1alpha1.StoreOfflinedConditionType, metav1.ConditionTrue) {
			return true, nil
		}
		return false, nil
	}
}
