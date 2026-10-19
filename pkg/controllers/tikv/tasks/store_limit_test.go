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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	pdapi "github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestAdjustStoreLimitsRetriesAndRestores(t *testing.T) {
	ctx := context.Background()
	kv := &v1alpha1.TiKV{ObjectMeta: metav1.ObjectMeta{Name: "kv", Namespace: "ns"}}
	c := client.NewFakeClient(kv)
	pd := pdapi.NewMockPDClient(gomock.NewController(t))
	limits := func(add, leader float64) map[string]pdapi.StoreLimits {
		return map[string]pdapi.StoreLimits{"1": {AddPeer: ptr.To(add), TransferLeaderIn: ptr.To(leader)}}
	}
	gomock.InOrder(
		pd.EXPECT().GetStoreLimits(ctx).Return(limits(15, 100), nil),
		pd.EXPECT().SetStoreLimit(ctx, "1", pdapi.StoreLimitAddPeer, incomingScaleInLimit).DoAndReturn(
			func(context.Context, string, pdapi.StoreLimitType, float64) error {
				stored := &v1alpha1.TiKV{}
				require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(kv), stored))
				require.Equal(t, "add-peer:15,transfer-leader-in:100", stored.Annotations[v1alpha1.AnnoKeyStoreLimits])
				return nil
			}),
		pd.EXPECT().SetStoreLimit(ctx, "1", pdapi.StoreLimitTransferLeaderIn, incomingScaleInLimit).Return(errors.New("PD write failed")),
		pd.EXPECT().GetStoreLimits(ctx).Return(limits(incomingScaleInLimit, 100), nil),
		pd.EXPECT().SetStoreLimit(ctx, "1", pdapi.StoreLimitTransferLeaderIn, incomingScaleInLimit).Return(nil),
		pd.EXPECT().GetStoreLimits(ctx).Return(limits(incomingScaleInLimit, incomingScaleInLimit), nil),
		pd.EXPECT().GetStoreLimits(ctx).Return(limits(incomingScaleInLimit, incomingScaleInLimit), nil),
		pd.EXPECT().SetStoreLimit(ctx, "1", pdapi.StoreLimitAddPeer, 15.0).Return(nil),
		pd.EXPECT().SetStoreLimit(ctx, "1", pdapi.StoreLimitTransferLeaderIn, 100.0).Return(nil),
	)
	require.Error(t, adjustStoreLimits(ctx, c, pd, kv, "1", true))
	// Reload the instance to simulate retry after a restart.
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(kv), kv))
	require.NoError(t, adjustStoreLimits(ctx, c, pd, kv, "1", true))
	require.NoError(t, adjustStoreLimits(ctx, c, pd, kv, "1", true), "already limited stores need no writes")
	require.NoError(t, adjustStoreLimits(ctx, c, pd, kv, "1", false))
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(kv), kv))
	require.Empty(t, kv.Annotations[v1alpha1.AnnoKeyStoreLimits])
}

func TestStoreLimitRestorationPreservesExternalChanges(t *testing.T) {
	ctx := context.Background()
	kv := &v1alpha1.TiKV{ObjectMeta: metav1.ObjectMeta{Name: "kv", Namespace: "ns"}}
	c := client.NewFakeClient(kv)
	require.NoError(t, saveOriginalStoreLimits(ctx, c, kv, &originalStoreLimits{AddPeer: 15, TransferLeaderIn: 100}))
	pd := pdapi.NewMockPDClient(gomock.NewController(t))
	pd.EXPECT().GetStoreLimits(ctx).Return(map[string]pdapi.StoreLimits{"1": {AddPeer: ptr.To(42.0), TransferLeaderIn: ptr.To(incomingScaleInLimit)}}, nil)
	pd.EXPECT().SetStoreLimit(ctx, "1", pdapi.StoreLimitTransferLeaderIn, 100.0).Return(nil)
	require.NoError(t, adjustStoreLimits(ctx, c, pd, kv, "1", false))
}

func TestStoreLimitAdjustmentStopsOnMissingCapabilityOrSaveFailure(t *testing.T) {
	for _, scenario := range []string{"missing transfer-leader-in", "annotation write failed"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			kv := &v1alpha1.TiKV{ObjectMeta: metav1.ObjectMeta{Name: "kv", Namespace: "ns"}}
			c := client.NewFakeClient(kv)
			limits := pdapi.StoreLimits{AddPeer: ptr.To(15.0)}
			if scenario == "annotation write failed" {
				limits.TransferLeaderIn = ptr.To(100.0)
				c.WithError("patch", "*", errors.New("patch failed"))
			}
			pd := pdapi.NewMockPDClient(gomock.NewController(t))
			pd.EXPECT().GetStoreLimits(ctx).Return(map[string]pdapi.StoreLimits{"1": limits}, nil)
			require.Error(t, adjustStoreLimits(ctx, c, pd, kv, "1", true))
		})
	}
}

func TestStoreLimitTaskSkipsUnneededPDCalls(t *testing.T) {
	for _, scenario := range []string{"gate disabled", "online without saved limits"} {
		t.Run(scenario, func(t *testing.T) {
			kv := &v1alpha1.TiKV{Spec: v1alpha1.TiKVSpec{}}
			kv.Spec.Features = []meta.Feature{meta.TiKVSequentialScaleIn}
			kv.Spec.Offline = ptr.To(scenario == "gate disabled")
			st := &state{tikv: kv, storeState: pdv1.NodeStateServing}
			if scenario == "gate disabled" {
				kv.Spec.Features = nil
			}
			st.cluster = &v1alpha1.Cluster{}
			for _, feature := range kv.Spec.Features {
				st.cluster.Spec.FeatureGates = append(st.cluster.Spec.FeatureGates, meta.FeatureGate{Name: feature})
			}
			st.IFeatureGates = stateutil.NewFeatureGates[scope.TiKV](st)
			rtx := &ReconcileContext{State: st, PDSynced: true, Store: &pdv1.Store{ID: "1"}}
			result, _ := task.RunTask(context.Background(), TaskStoreLimitAdjust(rtx, nil, nil))
			require.Equal(t, task.SComplete, result.Status(), result.Message())
		})
	}
}

func TestStoreLimitTaskRestoresDuringCancellation(t *testing.T) {
	ctx := context.Background()
	kv := &v1alpha1.TiKV{ObjectMeta: metav1.ObjectMeta{
		Name: "kv", Namespace: "ns",
		Annotations: map[string]string{v1alpha1.AnnoKeyStoreLimits: "add-peer:15,transfer-leader-in:100"},
	}}
	kv.Spec.Offline = ptr.To(false)
	c := client.NewFakeClient(kv)
	ctrl := gomock.NewController(t)
	pd := pdapi.NewMockPDClient(ctrl)
	pc := pdm.NewMockPDClient(ctrl)
	pc.EXPECT().Underlay().Return(pd)
	pd.EXPECT().GetStoreLimits(ctx).Return(map[string]pdapi.StoreLimits{
		"1": {AddPeer: ptr.To(incomingScaleInLimit), TransferLeaderIn: ptr.To(incomingScaleInLimit)},
	}, nil)
	pd.EXPECT().SetStoreLimit(ctx, "1", pdapi.StoreLimitAddPeer, 15.0).Return(nil)
	pd.EXPECT().SetStoreLimit(ctx, "1", pdapi.StoreLimitTransferLeaderIn, 100.0).Return(nil)
	st := &state{tikv: kv, storeState: pdv1.NodeStateRemoving}
	st.IPDClient = &stubPDClientState{client: pc}
	rtx := &ReconcileContext{State: st, PDSynced: true, Store: &pdv1.Store{ID: "1"}}
	result, _ := task.RunTask(ctx, TaskStoreLimitAdjust(rtx, c, nil))
	require.Equal(t, task.SComplete, result.Status(), result.Message())
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(kv), kv))
	require.Empty(t, kv.Annotations[v1alpha1.AnnoKeyStoreLimits])
}

func TestOfflineStoreFeatureSwitchAndSelection(t *testing.T) {
	for _, stale := range []bool{false, true} {
		for _, scenario := range []string{"legacy eviction", "selected self", "selected peer"} {
			name := scenario
			if stale {
				name += " with stale instance features"
			}
			t.Run(name, func(t *testing.T) {
				kv := &v1alpha1.TiKV{ObjectMeta: metav1.ObjectMeta{Name: "b", UID: "b"}}
				kv.Spec.Offline = ptr.To(true)
				if scenario == "selected self" {
					kv.Annotations = map[string]string{v1alpha1.AnnoKeySelectedToDelete: v1alpha1.AnnoValTrue}
				}
				if scenario != "legacy eviction" {
					kv.Spec.Features = []meta.Feature{meta.TiKVSequentialScaleIn}
				}
				st := &state{tikv: kv, storeState: pdv1.NodeStateServing}
				st.cluster = &v1alpha1.Cluster{}
				for _, feature := range kv.Spec.Features {
					st.cluster.Spec.FeatureGates = append(st.cluster.Spec.FeatureGates, meta.FeatureGate{Name: feature})
				}
				if stale {
					kv.Spec.Features = []meta.Feature{meta.FeatureModification}
					if scenario == "legacy eviction" {
						kv.Spec.Features = append(kv.Spec.Features, meta.TiKVSequentialScaleIn)
					}
				}
				st.IFeatureGates = stateutil.NewFeatureGates[scope.TiKV](st)
				ctrl := gomock.NewController(t)
				pc := pdm.NewMockPDClient(ctrl)
				st.IPDClient = &stubPDClientState{client: pc}
				if scenario != "legacy eviction" {
					pd := pdapi.NewMockPDClient(ctrl)
					pc.EXPECT().Underlay().Return(pd)
					if scenario == "selected self" {
						pd.EXPECT().DeleteStore(gomock.Any(), "1").Return(nil)
					}
					cache := pdm.NewMockStoreCache[pdv1.Store, *pdv1.Store](ctrl)
					pc.EXPECT().Stores().Return(cache)
					cache.EXPECT().Refresh()
				}
				rtx := &ReconcileContext{State: st, PDSynced: true, Store: &pdv1.Store{ID: "1"}}
				result, _ := task.RunTask(context.Background(), TaskOfflineStore(rtx, nil, client.NewFakeClient(kv)))
				expected := task.SWait
				if scenario == "legacy eviction" {
					expected = task.SRetry
				}
				require.Equal(t, expected, result.Status(), result.Message())
				require.Equal(t, scenario == "legacy eviction", rtx.ShouldEvictLeader)
			})
		}
	}
}

func TestParseOriginalStoreLimits(t *testing.T) {
	for _, value := range []string{"add-peer:10,transfer-leader-in:10", "transfer-leader-in:10,add-peer:10"} {
		limits, err := parseOriginalStoreLimits(value)
		require.NoError(t, err)
		require.Equal(t, originalStoreLimits{AddPeer: 10, TransferLeaderIn: 10}, limits)
	}
	limits, err := parseOriginalStoreLimits("add-peer:0.25,transfer-leader-in:1e-6")
	require.NoError(t, err)
	require.Equal(t, originalStoreLimits{AddPeer: 0.25, TransferLeaderIn: 0.000001}, limits)
	for _, value := range []string{"add-peer:10", "add-peer:bad,transfer-leader-in:10", "add-peer=10,transfer-leader-in:10", "add-peer:10,other:10"} {
		_, err := parseOriginalStoreLimits(value)
		require.Error(t, err, value)
	}
}

func TestStoreLimitUsesClusterGate(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		kv := &v1alpha1.TiKV{}
		kv.Spec.Offline = ptr.To(true)
		kv.Spec.Features = []meta.Feature{meta.FeatureModification}
		if !enabled {
			kv.Spec.Features = append(kv.Spec.Features, meta.TiKVSequentialScaleIn)
		}
		st := &state{tikv: kv, cluster: &v1alpha1.Cluster{}}
		if enabled {
			st.cluster.Spec.FeatureGates = []meta.FeatureGate{{Name: meta.TiKVSequentialScaleIn}}
		}
		st.IFeatureGates = stateutil.NewFeatureGates[scope.TiKV](st)
		result, _ := task.RunTask(context.Background(), TaskStoreLimitAdjust(&ReconcileContext{State: st}, nil, nil))
		expected := task.SComplete
		if enabled {
			expected = task.SWait
		}
		require.Equal(t, expected, result.Status())
	}
}

func TestOfflineCancellationRetainsSelection(t *testing.T) {
	for _, scenario := range []string{"cancel pending", "cancel failed", "confirmed", "removed", "patch failed"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			kv := &v1alpha1.TiKV{ObjectMeta: metav1.ObjectMeta{Name: "a", ResourceVersion: "1", Annotations: map[string]string{
				v1alpha1.AnnoKeySelectedToDelete: v1alpha1.AnnoValTrue,
			}}}
			kv.Spec.Offline = ptr.To(false)
			st := &state{tikv: kv, cluster: &v1alpha1.Cluster{}, storeState: pdv1.NodeStateServing}
			if scenario == "cancel pending" || scenario == "cancel failed" {
				st.storeState = pdv1.NodeStateRemoving
			}
			if scenario == "removed" {
				st.storeState = pdv1.NodeStateRemoved
			}
			ctrl := gomock.NewController(t)
			pc := pdm.NewMockPDClient(ctrl)
			pd := pdapi.NewMockPDClient(ctrl)
			st.IPDClient = &stubPDClientState{client: pc}
			pc.EXPECT().Underlay().Return(pd)
			if st.storeState == pdv1.NodeStateRemoving {
				var err error
				if scenario == "cancel failed" {
					err = errors.New("PD unavailable")
				}
				pd.EXPECT().CancelDeleteStore(ctx, "1").Return(err)
				cache := pdm.NewMockStoreCache[pdv1.Store, *pdv1.Store](ctrl)
				pc.EXPECT().Stores().Return(cache)
				cache.EXPECT().Refresh()
			}
			fc := client.NewFakeClient(kv)
			if scenario == "patch failed" {
				fc.WithError("patch", "*", errors.New("patch failed"))
			}
			rtx := &ReconcileContext{State: st, PDSynced: true, Store: &pdv1.Store{ID: "1"}}
			result, _ := task.RunTask(ctx, TaskOfflineStore(rtx, nil, fc))
			expected := task.SComplete
			if scenario == "cancel pending" {
				expected = task.SWait
			}
			if scenario == "cancel failed" || scenario == "patch failed" {
				expected = task.SFail
			}
			require.Equal(t, expected, result.Status(), result.Message())
			saved := &v1alpha1.TiKV{}
			require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(kv), saved))
			released := scenario == "confirmed" || scenario == "removed"
			require.Equal(t, !released, saved.Annotations[v1alpha1.AnnoKeySelectedToDelete] == v1alpha1.AnnoValTrue)
		})
	}
}
