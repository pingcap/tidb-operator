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

package placementpolicy

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
)

func TestBuildRulesUsesTiKVGroupLabelForAllTargets(t *testing.T) {
	normal := newTiKVGroup("g1", false)
	exclusive := newTiKVGroup("g2", true)
	r := &Reconciler{
		Client: client.NewFakeClient(normal, exclusive),
	}
	policy := newPlacementPolicy()

	rules, err := r.buildRules(context.Background(), policy)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	for _, rule := range rules {
		assert.Equal(t, "tidb-operator", rule.GroupID)
		assert.Equal(t, "p:r1-1-txn", rule.ID)
		assert.Equal(t, "7800000100000000fb", rule.StartKeyHex)
		assert.Equal(t, "7800000200000000fb", rule.EndKeyHex)
		assert.Equal(t, v1alpha1.PlacementPolicyRoleVoter, rule.Role)
		assert.Equal(t, int32(3), rule.Count)
		require.Len(t, rule.LabelConstraints, 2)
		assert.Equal(t, pdapi.PlacementLabelConstraint{
			Key:    v1alpha1.PlacementTiKVGroupLabelKey,
			Op:     "in",
			Values: []string{"ns.g1", "ns.g2"},
		}, rule.LabelConstraints[0])
		assert.Equal(t, pdapi.PlacementLabelConstraint{
			Key:    v1alpha1.PlacementTiKVGroupExclusiveLabelKey,
			Op:     "notIn",
			Values: []string{placementExclusiveConstraintNonexistentValue},
		}, rule.LabelConstraints[1])
	}
}

func TestReconcileDeleteRemovesFinalizerWhenClusterMissing(t *testing.T) {
	ctx := context.Background()
	policy := newDeletingPlacementPolicy()
	cli := client.NewFakeClient(policy)
	r := &Reconciler{
		Client:          cli,
		PDClientManager: fakePDClientManager{},
	}

	_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}})
	require.NoError(t, err)

	var actual v1alpha1.PlacementPolicy
	require.NoError(t, cli.Get(ctx, types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, &actual))
	assert.NotContains(t, actual.Finalizers, metav1alpha1.Finalizer)
}

func TestReconcileDeleteRemovesFinalizerWhenClusterDeleting(t *testing.T) {
	ctx := context.Background()
	policy := newDeletingPlacementPolicy()
	cluster := newDeletingCluster()
	cli := client.NewFakeClient(policy, cluster)
	r := &Reconciler{
		Client:          cli,
		PDClientManager: fakePDClientManager{},
	}

	_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}})
	require.NoError(t, err)

	var actual v1alpha1.PlacementPolicy
	require.NoError(t, cli.Get(ctx, types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, &actual))
	assert.NotContains(t, actual.Finalizers, metav1alpha1.Finalizer)
}

func TestReconcileSkipsSyncWhenClusterPausedOrSuspending(t *testing.T) {
	for _, tt := range []struct {
		name   string
		update func(*v1alpha1.Cluster)
	}{
		{
			name: "paused",
			update: func(cluster *v1alpha1.Cluster) {
				cluster.Spec.Paused = true
			},
		},
		{
			name: "suspending compute",
			update: func(cluster *v1alpha1.Cluster) {
				cluster.Spec.SuspendAction = &v1alpha1.SuspendAction{SuspendCompute: true}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			policy := newPlacementPolicy()
			cluster := newCluster()
			tt.update(cluster)
			cli := client.NewFakeClient(policy, cluster)
			r := &Reconciler{
				Client:          cli,
				PDClientManager: fakePDClientManager{},
			}

			_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}})
			require.NoError(t, err)

			var actual v1alpha1.PlacementPolicy
			require.NoError(t, cli.Get(ctx, types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, &actual))
			assert.Contains(t, actual.Finalizers, metav1alpha1.Finalizer)
			assert.Nil(t, apimeta.FindStatusCondition(actual.Status.Conditions, v1alpha1.CondSynced))
		})
	}
}

func TestReconcileDeleteSkipsSyncWhenClusterPaused(t *testing.T) {
	ctx := context.Background()
	policy := newDeletingPlacementPolicy()
	cluster := newCluster()
	cluster.Spec.Paused = true
	cli := client.NewFakeClient(policy, cluster)
	r := &Reconciler{
		Client:          cli,
		PDClientManager: fakePDClientManager{},
	}

	_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}})
	require.NoError(t, err)

	var actual v1alpha1.PlacementPolicy
	require.NoError(t, cli.Get(ctx, types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, &actual))
	assert.Contains(t, actual.Finalizers, metav1alpha1.Finalizer)
}

func TestClusterEventHandlerEnqueuesByClusterField(t *testing.T) {
	ctx := context.Background()
	policy := newPlacementPolicy()
	otherPolicy := newPlacementPolicyForCluster("other", "other")
	r := &Reconciler{
		Client: client.NewFakeClient(policy, otherPolicy),
	}
	eventHandler := r.ClusterEventHandler()
	cluster := newCluster()

	createQueue := newRequestQueue()
	eventHandler.Create(ctx, event.TypedCreateEvent[client.Object]{Object: cluster}, createQueue)
	assert.ElementsMatch(t, []reconcile.Request{newRequest(policy)}, drainRequestQueue(createQueue))

	updateQueue := newRequestQueue()
	updatedCluster := cluster.DeepCopy()
	updatedCluster.Spec.Paused = true
	eventHandler.Update(ctx, event.TypedUpdateEvent[client.Object]{ObjectOld: cluster, ObjectNew: updatedCluster}, updateQueue)
	assert.ElementsMatch(t, []reconcile.Request{newRequest(policy)}, drainRequestQueue(updateQueue))

	deleteQueue := newRequestQueue()
	eventHandler.Delete(ctx, event.TypedDeleteEvent[client.Object]{Object: cluster}, deleteQueue)
	assert.ElementsMatch(t, []reconcile.Request{newRequest(policy)}, drainRequestQueue(deleteQueue))
}

func TestTiKVGroupEventHandlerEnqueuesOnCreateDeleteAndExclusiveChange(t *testing.T) {
	ctx := context.Background()
	policy := newPlacementPolicy()
	otherPolicy := newPlacementPolicyForCluster("other", "other")
	r := &Reconciler{
		Client: client.NewFakeClient(policy, otherPolicy),
	}
	eventHandler := r.TiKVGroupEventHandler()
	kvg := newTiKVGroup("g1", false)

	createQueue := newRequestQueue()
	eventHandler.Create(ctx, event.TypedCreateEvent[client.Object]{Object: kvg}, createQueue)
	assert.ElementsMatch(t, []reconcile.Request{newRequest(policy)}, drainRequestQueue(createQueue))

	unchangedQueue := newRequestQueue()
	unchanged := kvg.DeepCopy()
	unchanged.Labels = map[string]string{"k": "v"}
	eventHandler.Update(ctx, event.TypedUpdateEvent[client.Object]{ObjectOld: kvg, ObjectNew: unchanged}, unchangedQueue)
	assert.Empty(t, drainRequestQueue(unchangedQueue))

	nilToFalseQueue := newRequestQueue()
	nilPlacement := kvg.DeepCopy()
	nilPlacement.Spec.Template.Spec.Placement = nil
	eventHandler.Update(ctx, event.TypedUpdateEvent[client.Object]{ObjectOld: nilPlacement, ObjectNew: kvg}, nilToFalseQueue)
	assert.Empty(t, drainRequestQueue(nilToFalseQueue))

	exclusiveQueue := newRequestQueue()
	exclusive := newTiKVGroup("g1", true)
	eventHandler.Update(ctx, event.TypedUpdateEvent[client.Object]{ObjectOld: kvg, ObjectNew: exclusive}, exclusiveQueue)
	assert.ElementsMatch(t, []reconcile.Request{newRequest(policy)}, drainRequestQueue(exclusiveQueue))

	deleteQueue := newRequestQueue()
	eventHandler.Delete(ctx, event.TypedDeleteEvent[client.Object]{Object: kvg}, deleteQueue)
	assert.ElementsMatch(t, []reconcile.Request{newRequest(policy)}, drainRequestQueue(deleteQueue))
}

func TestSyncPolicyRulesPostsRuleGroup(t *testing.T) {
	ctx := context.Background()
	locationLabels := []string{"zone", "rack"}
	rules := []pdapi.PlacementRule{
		{
			GroupID: "tidb-operator",
			ID:      "p:r1-1-txn",
			Role:    v1alpha1.PlacementPolicyRoleVoter,
			Count:   3,
		},
	}
	expectedRules := []pdapi.PlacementRule{
		{
			GroupID:        "tidb-operator",
			ID:             "p:r1-1-raw",
			Role:           v1alpha1.PlacementPolicyRoleVoter,
			Count:          3,
			LocationLabels: locationLabels,
		},
	}
	pdc := pdapi.NewMockPDClient(gomock.NewController(t))
	gomock.InOrder(
		pdc.EXPECT().GetPlacementRuleBundle(gomock.Any(), "pd").Return(&pdapi.PlacementRuleGroupBundle{
			ID: "pd",
			Rules: []pdapi.PlacementRule{
				{
					GroupID:        "pd",
					ID:             "default",
					LocationLabels: locationLabels,
				},
			},
		}, nil),
		pdc.EXPECT().SetPlacementRuleGroup(gomock.Any(), &pdapi.PlacementRuleGroup{
			ID:       "tidb-operator",
			Index:    pdPlacementRuleGroupIndex,
			Override: true,
		}).Return(nil),
		pdc.EXPECT().SetPlacementRuleGroupRulesByIDPrefix(gomock.Any(), "tidb-operator", "p:", expectedRules).Return(nil),
	)

	err := syncPolicyRules(ctx, pdc, "p:", rules)
	require.NoError(t, err)
}

func TestSyncPolicyRulesUsesEmptyLocationLabelsWhenDefaultRuleMissing(t *testing.T) {
	ctx := context.Background()
	rules := []pdapi.PlacementRule{
		{
			GroupID: "tidb-operator",
			ID:      "p:r1-1-raw",
			Role:    v1alpha1.PlacementPolicyRoleVoter,
			Count:   3,
		},
	}
	expectedRules := []pdapi.PlacementRule{
		{
			GroupID: "tidb-operator",
			ID:      "p:r1-1-raw",
			Role:    v1alpha1.PlacementPolicyRoleVoter,
			Count:   3,
		},
	}
	pdc := pdapi.NewMockPDClient(gomock.NewController(t))
	pdc.EXPECT().GetPlacementRuleBundle(gomock.Any(), "pd").Return(&pdapi.PlacementRuleGroupBundle{
		ID: "pd",
		Rules: []pdapi.PlacementRule{
			{
				GroupID: "pd",
				ID:      "other",
			},
		},
	}, nil)
	gomock.InOrder(
		pdc.EXPECT().SetPlacementRuleGroup(gomock.Any(), &pdapi.PlacementRuleGroup{
			ID:       "tidb-operator",
			Index:    pdPlacementRuleGroupIndex,
			Override: true,
		}).Return(nil),
		pdc.EXPECT().SetPlacementRuleGroupRulesByIDPrefix(gomock.Any(), "tidb-operator", "p:", expectedRules).Return(nil),
	)

	err := syncPolicyRules(ctx, pdc, "p:", rules)
	require.NoError(t, err)
}

func TestSyncPolicyRulesNormalizesEmptyDefaultLocationLabels(t *testing.T) {
	ctx := context.Background()
	rules := []pdapi.PlacementRule{
		{
			GroupID: "tidb-operator",
			ID:      "p:r1-1-raw",
			Role:    v1alpha1.PlacementPolicyRoleVoter,
			Count:   3,
		},
	}
	expectedRules := []pdapi.PlacementRule{
		{
			GroupID: "tidb-operator",
			ID:      "p:r1-1-raw",
			Role:    v1alpha1.PlacementPolicyRoleVoter,
			Count:   3,
		},
	}
	pdc := pdapi.NewMockPDClient(gomock.NewController(t))
	gomock.InOrder(
		pdc.EXPECT().GetPlacementRuleBundle(gomock.Any(), "pd").Return(&pdapi.PlacementRuleGroupBundle{
			ID: "pd",
			Rules: []pdapi.PlacementRule{
				{
					GroupID:        "pd",
					ID:             "default",
					LocationLabels: []string{},
				},
			},
		}, nil),
		pdc.EXPECT().SetPlacementRuleGroup(gomock.Any(), &pdapi.PlacementRuleGroup{
			ID:       "tidb-operator",
			Index:    pdPlacementRuleGroupIndex,
			Override: true,
		}).Return(nil),
		pdc.EXPECT().SetPlacementRuleGroupRulesByIDPrefix(gomock.Any(), "tidb-operator", "p:", expectedRules).Return(nil),
	)

	err := syncPolicyRules(ctx, pdc, "p:", rules)
	require.NoError(t, err)
}

func TestSyncPolicyRulesReturnsDefaultRuleLookupError(t *testing.T) {
	ctx := context.Background()
	rules := []pdapi.PlacementRule{
		{
			GroupID: "tidb-operator",
			ID:      "p:r1-1-raw",
			Role:    v1alpha1.PlacementPolicyRoleVoter,
			Count:   3,
		},
	}
	pdc := pdapi.NewMockPDClient(gomock.NewController(t))
	pdc.EXPECT().GetPlacementRuleBundle(gomock.Any(), "pd").Return(nil, errors.New("pd unavailable"))

	err := syncPolicyRules(ctx, pdc, "p:", rules)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get default placement rule bundle")
}

func TestSyncPolicyRulesSkipsDefaultRuleLookupWhenNoRules(t *testing.T) {
	ctx := context.Background()
	pdc := pdapi.NewMockPDClient(gomock.NewController(t))
	gomock.InOrder(
		pdc.EXPECT().SetPlacementRuleGroup(gomock.Any(), &pdapi.PlacementRuleGroup{
			ID:       "tidb-operator",
			Index:    pdPlacementRuleGroupIndex,
			Override: true,
		}).Return(nil),
		pdc.EXPECT().SetPlacementRuleGroupRulesByIDPrefix(
			gomock.Any(),
			"tidb-operator",
			"p:",
			[]pdapi.PlacementRule(nil),
		).Return(nil),
	)

	err := syncPolicyRules(ctx, pdc, "p:", nil)
	require.NoError(t, err)
}

func TestDeletePolicyRulesDeletesPolicyPrefixOnly(t *testing.T) {
	ctx := context.Background()
	policy := newPlacementPolicy()
	pdc := pdapi.NewMockPDClient(gomock.NewController(t))
	pdc.EXPECT().DeletePlacementRuleGroupRulesByIDPrefix(gomock.Any(), "tidb-operator", "p:").Return(nil)

	require.NoError(t, deletePolicyRules(ctx, pdc, policy))
}

func TestUpdateStatusReturnsOriginalError(t *testing.T) {
	ctx := context.Background()
	policy := newPlacementPolicy()
	originalErr := errors.New("sync failed")
	cli := client.NewFakeClient(policy)
	r := &Reconciler{
		Client: cli,
	}

	err := r.updateStatus(ctx, policy, originalErr)

	require.ErrorIs(t, err, originalErr)

	var actual v1alpha1.PlacementPolicy
	require.NoError(t, cli.Get(ctx, types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, &actual))
	cond := apimeta.FindStatusCondition(actual.Status.Conditions, v1alpha1.CondSynced)
	require.NotNil(t, cond)
	assert.Equal(t, metav1.ConditionFalse, cond.Status)
	assert.Equal(t, reasonSyncFailed, cond.Reason)
	assert.Equal(t, originalErr.Error(), cond.Message)
}

func TestUpdateStatusGeneratesSyncedCondition(t *testing.T) {
	ctx := context.Background()
	policy := newPlacementPolicy()
	cli := client.NewFakeClient(policy)
	r := &Reconciler{
		Client: cli,
	}

	err := r.updateStatus(ctx, policy, nil)

	require.NoError(t, err)

	var actual v1alpha1.PlacementPolicy
	require.NoError(t, cli.Get(ctx, types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, &actual))
	cond := apimeta.FindStatusCondition(actual.Status.Conditions, v1alpha1.CondSynced)
	require.NotNil(t, cond)
	assert.Equal(t, metav1.ConditionTrue, cond.Status)
	assert.Equal(t, reasonSynced, cond.Reason)
	assert.Equal(t, "Placement policy synced", cond.Message)
}

func TestUpdateStatusReturnsStatusUpdateError(t *testing.T) {
	ctx := context.Background()
	policy := newPlacementPolicy()
	updateErr := errors.New("update failed")
	cli := client.NewFakeClient(policy)
	cli.WithError("update", "placementpolicies", updateErr)
	r := &Reconciler{
		Client: cli,
	}

	err := r.updateStatus(ctx, policy, nil)

	require.ErrorIs(t, err, updateErr)
	require.Contains(t, err.Error(), "failed to update placement policy status")
}

func TestUpdateStatusJoinsOriginalAndStatusUpdateErrors(t *testing.T) {
	ctx := context.Background()
	policy := newPlacementPolicy()
	originalErr := errors.New("sync failed")
	updateErr := errors.New("update failed")
	cli := client.NewFakeClient(policy)
	cli.WithError("update", "placementpolicies", updateErr)
	r := &Reconciler{
		Client: cli,
	}

	err := r.updateStatus(ctx, policy, originalErr)

	require.ErrorIs(t, err, originalErr)
	require.ErrorIs(t, err, updateErr)
	require.Contains(t, err.Error(), "failed to update placement policy status")
}

func newPlacementPolicy() *v1alpha1.PlacementPolicy {
	return newPlacementPolicyForCluster("p", "c")
}

func newPlacementPolicyForCluster(name, clusterName string) *v1alpha1.PlacementPolicy {
	return &v1alpha1.PlacementPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      name,
		},
		Spec: v1alpha1.PlacementPolicySpec{
			Cluster: v1alpha1.ClusterReference{Name: clusterName},
			GroupRefs: []v1alpha1.PlacementPolicyGroupRef{
				{
					Group: v1alpha1.GroupName,
					Kind:  "TiKVGroup",
					Name:  "g1",
				},
				{
					Group: v1alpha1.GroupName,
					Kind:  "TiKVGroup",
					Name:  "g2",
				},
			},
			Rules: []v1alpha1.PlacementPolicyRule{
				{
					Name:  "r1",
					Role:  v1alpha1.PlacementPolicyRoleVoter,
					Count: 3,
					Selector: v1alpha1.PlacementPolicySelector{
						Keyspace: &v1alpha1.PlacementPolicyKeyspaceSelector{
							IDs: []string{"1"},
						},
					},
				},
			},
		},
	}
}

func newCluster() *v1alpha1.Cluster {
	return &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "c",
		},
	}
}

func newDeletingPlacementPolicy() *v1alpha1.PlacementPolicy {
	policy := newPlacementPolicy()
	now := metav1.Now()
	policy.DeletionTimestamp = &now
	policy.Finalizers = []string{metav1alpha1.Finalizer}
	return policy
}

func newDeletingCluster() *v1alpha1.Cluster {
	now := metav1.Now()
	cluster := newCluster()
	cluster.DeletionTimestamp = &now
	cluster.Finalizers = []string{metav1alpha1.Finalizer}
	return cluster
}

func newTiKVGroup(name string, exclusive bool) *v1alpha1.TiKVGroup {
	return &v1alpha1.TiKVGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      name,
		},
		Spec: v1alpha1.TiKVGroupSpec{
			Cluster: v1alpha1.ClusterReference{Name: "c"},
			Template: v1alpha1.TiKVTemplate{
				Spec: v1alpha1.TiKVTemplateSpec{
					Placement: &v1alpha1.TiKVStorePlacement{
						Exclusive: ptr.To(exclusive),
					},
				},
			},
		},
	}
}

type fakePDClientManager struct{}

func newRequest(policy *v1alpha1.PlacementPolicy) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: policy.Namespace,
			Name:      policy.Name,
		},
	}
}

func newRequestQueue() workqueue.TypedRateLimitingInterface[reconcile.Request] {
	return workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedItemBasedRateLimiter[reconcile.Request]())
}

func drainRequestQueue(queue workqueue.TypedRateLimitingInterface[reconcile.Request]) []reconcile.Request {
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

func (fakePDClientManager) Register(*v1alpha1.Cluster) error {
	return nil
}

func (fakePDClientManager) Deregister(string) {}

func (fakePDClientManager) Get(string) (pdm.PDClient, bool) {
	return nil, false
}

func (fakePDClientManager) Source(runtime.Object, handler.EventHandler) source.Source {
	return nil
}

func (fakePDClientManager) Start(context.Context) {}
