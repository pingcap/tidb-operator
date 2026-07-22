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

package tikvgroup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func TestPlacementPolicyEventHandlerEnqueuesTiKVGroupRefs(t *testing.T) {
	ctx := context.Background()
	r := &Reconciler{}
	eventHandler := r.PlacementPolicyEventHandler()
	policy := newTiKVGroupPlacementPolicy("p", "g1", "g2")

	createQueue := newTiKVGroupRequestQueue()
	eventHandler.Create(ctx, event.TypedCreateEvent[client.Object]{Object: policy}, createQueue)
	assert.ElementsMatch(t, []reconcile.Request{
		newTiKVGroupRequest("ns", "g1"),
		newTiKVGroupRequest("ns", "g2"),
	}, drainTiKVGroupRequestQueue(createQueue))

	unchangedQueue := newTiKVGroupRequestQueue()
	unchanged := policy.DeepCopy()
	unchanged.Labels = map[string]string{"k": "v"}
	eventHandler.Update(ctx, event.TypedUpdateEvent[client.Object]{ObjectOld: policy, ObjectNew: unchanged}, unchangedQueue)
	assert.Empty(t, drainTiKVGroupRequestQueue(unchangedQueue))

	removedRefQueue := newTiKVGroupRequestQueue()
	removedRef := newTiKVGroupPlacementPolicy("p", "g2")
	eventHandler.Update(ctx, event.TypedUpdateEvent[client.Object]{ObjectOld: policy, ObjectNew: removedRef}, removedRefQueue)
	assert.ElementsMatch(t, []reconcile.Request{
		newTiKVGroupRequest("ns", "g1"),
		newTiKVGroupRequest("ns", "g2"),
	}, drainTiKVGroupRequestQueue(removedRefQueue))

	deleteQueue := newTiKVGroupRequestQueue()
	eventHandler.Delete(ctx, event.TypedDeleteEvent[client.Object]{Object: policy}, deleteQueue)
	assert.ElementsMatch(t, []reconcile.Request{
		newTiKVGroupRequest("ns", "g1"),
		newTiKVGroupRequest("ns", "g2"),
	}, drainTiKVGroupRequestQueue(deleteQueue))
}

func newTiKVGroupPlacementPolicy(name string, groups ...string) *v1alpha1.PlacementPolicy {
	refs := make([]v1alpha1.PlacementPolicyGroupRef, 0, len(groups))
	for _, group := range groups {
		refs = append(refs, v1alpha1.PlacementPolicyGroupRef{
			Group: v1alpha1.GroupName,
			Kind:  "TiKVGroup",
			Name:  group,
		})
	}

	return &v1alpha1.PlacementPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      name,
		},
		Spec: v1alpha1.PlacementPolicySpec{
			Cluster:   v1alpha1.ClusterReference{Name: "c"},
			GroupRefs: refs,
		},
	}
}

func newTiKVGroupRequest(namespace, name string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func newTiKVGroupRequestQueue() workqueue.TypedRateLimitingInterface[reconcile.Request] {
	return workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedItemBasedRateLimiter[reconcile.Request]())
}

func drainTiKVGroupRequestQueue(queue workqueue.TypedRateLimitingInterface[reconcile.Request]) []reconcile.Request {
	requests := []reconcile.Request{}
	for queue.Len() > 0 {
		request, shutdown := queue.Get()
		if shutdown {
			break
		}
		requests = append(requests, request)
		queue.Done(request)
	}
	return requests
}
