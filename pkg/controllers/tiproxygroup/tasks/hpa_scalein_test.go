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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

func TestWithHPAScaleInAnnotation(t *testing.T) {
	t.Parallel()

	tiproxy := &v1alpha1.TiProxy{ObjectMeta: metav1.ObjectMeta{Name: "tiproxy-a", Namespace: "ns"}}
	original := client.RawPatch(types.MergePatchType, []byte(`{"metadata":{"resourceVersion":"1","annotations":{"core.pingcap.com/defer-delete":null}},"spec":{"offline":true}}`))

	next, changed, err := withHPAScaleInAnnotation(original, tiproxy, true)
	require.NoError(t, err)
	assert.True(t, changed)

	data, err := next.Data(tiproxy)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(data, &body))
	spec := body["spec"].(map[string]any)
	assert.Equal(t, true, spec["offline"])
	metadata := body["metadata"].(map[string]any)
	annotations := metadata["annotations"].(map[string]any)
	assert.Equal(t, v1alpha1.AnnoValTrue, annotations[v1alpha1.AnnoKeyTiProxyHPAScaleIn])
	assert.Nil(t, annotations[v1alpha1.AnnoKeyDeferDelete])

	cleared, changed, err := withHPAScaleInAnnotation(original, tiproxy, false)
	require.NoError(t, err)
	assert.True(t, changed)
	data, err = cleared.Data(tiproxy)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &body))
	annotations = body["metadata"].(map[string]any)["annotations"].(map[string]any)
	assert.Nil(t, annotations[v1alpha1.AnnoKeyTiProxyHPAScaleIn])
	assert.Nil(t, annotations[v1alpha1.AnnoKeyDeferDelete])

	unchanged, changed, err := withHPAScaleInAnnotation(
		client.RawPatch(types.MergePatchType, []byte(`{"metadata":{"annotations":{"core.pingcap.com/defer-delete":"true"}}}`)),
		tiproxy,
		true,
	)
	require.NoError(t, err)
	assert.False(t, changed)
	raw, err := unchanged.Data(tiproxy)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), v1alpha1.AnnoKeyTiProxyHPAScaleIn)

	invalid := client.RawPatch(types.JSONPatchType, []byte(`[]`))
	skipped, changed, err := withHPAScaleInAnnotation(invalid, tiproxy, true)
	require.Error(t, err)
	assert.False(t, changed)
	assert.Nil(t, skipped)
}

func TestTaskUpdaterAnnotatesOnlyHPAScaleIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		manager     string
		subresource string
		wantAnno    bool
		replicas    int32
		revision    string
		image       string
		wantOffline bool
	}{
		{
			name:        "HPA scale-in marks the offlined instance",
			manager:     runtime.HPAFieldManager,
			subresource: "scale",
			wantAnno:    true,
			replicas:    1,
			revision:    newRevision,
			wantOffline: true,
		},
		{
			name:        "manual scale-in does not mark the offlined instance",
			manager:     "kubectl-scale",
			subresource: "scale",
			replicas:    1,
			revision:    newRevision,
			wantOffline: true,
		},
		{
			name:        "kube-controller-manager update outside scale is not an HPA scale-in",
			manager:     runtime.HPAFieldManager,
			subresource: "status",
			replicas:    1,
			revision:    newRevision,
			wantOffline: true,
		},
		{
			name:        "HPA rolling update does not mark instances",
			manager:     runtime.HPAFieldManager,
			subresource: "scale",
			replicas:    2,
			revision:    oldRevision,
			image:       "pingcap/tiproxy:v8.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			proxyg := fake.FakeObj("aaa", func(obj *v1alpha1.TiProxyGroup) *v1alpha1.TiProxyGroup {
				obj.Spec.Replicas = ptr.To(tt.replicas)
				obj.Spec.Template.Annotations = map[string]string{
					v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "60",
				}
				if tt.image != "" {
					obj.Spec.Template.Spec.Image = ptr.To(tt.image)
				}
				obj.ManagedFields = []metav1.ManagedFieldsEntry{{
					Manager:     tt.manager,
					Operation:   metav1.ManagedFieldsOperationUpdate,
					Subresource: tt.subresource,
					FieldsV1:    &metav1.FieldsV1{Raw: []byte(`{"f:spec":{"f:replicas":{}}}`)},
				}}
				return obj
			})
			s := &state{
				proxyg:  proxyg,
				cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				proxies: []*v1alpha1.TiProxy{
					fakeAvailableTiProxy("aaa-xxx", proxyg, tt.revision),
					fakeAvailableTiProxy("aaa-yyy", proxyg, tt.revision),
				},
				updateRevision: newRevision,
			}
			s.IFeatureGates = stateutil.NewFeatureGates[scope.TiProxyGroup](s)

			fc := client.NewFakeClient(proxyg, s.cluster)
			for _, obj := range s.proxies {
				require.NoError(t, fc.Apply(ctx, obj.DeepCopy()))
				require.NoError(t, fc.Status().Update(ctx, obj.DeepCopy()))
			}

			res, done := task.RunTask(ctx, TaskUpdater(&ReconcileContext{State: s}, fc, tracker.New().AllocateFactory("tiproxy")))
			require.False(t, done)
			assert.NotEqual(t, task.SFail.String(), res.Status().String(), res.Message())

			list := v1alpha1.TiProxyList{}
			require.NoError(t, fc.List(ctx, &list))

			var offlined, annotated int
			for i := range list.Items {
				item := &list.Items[i]
				if item.Annotations[v1alpha1.AnnoKeyTiProxyHPAScaleIn] == v1alpha1.AnnoValTrue {
					annotated++
					assert.NotNil(t, item.Spec.Offline)
					assert.True(t, *item.Spec.Offline)
				}
				if item.Spec.Offline != nil && *item.Spec.Offline {
					offlined++
				}
			}
			if tt.wantAnno {
				assert.Equal(t, 1, annotated)
			} else {
				assert.Zero(t, annotated)
			}
			if tt.wantOffline {
				assert.Equal(t, 1, offlined)
			} else {
				assert.Zero(t, offlined)
			}
		})
	}
}

func TestTaskUpdaterManualScaleInClearsStaleHPAAnnotation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	proxyg := fake.FakeObj("aaa", func(obj *v1alpha1.TiProxyGroup) *v1alpha1.TiProxyGroup {
		obj.Spec.Replicas = ptr.To[int32](1)
		obj.Spec.Template.Annotations = map[string]string{
			v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "60",
		}
		obj.ManagedFields = []metav1.ManagedFieldsEntry{{
			Manager:     "kubectl-scale",
			Operation:   metav1.ManagedFieldsOperationUpdate,
			Subresource: "scale",
			FieldsV1:    &metav1.FieldsV1{Raw: []byte(`{"f:spec":{"f:replicas":{}}}`)},
		}}
		return obj
	})
	proxies := []*v1alpha1.TiProxy{
		fakeAvailableTiProxy("aaa-xxx", proxyg, newRevision),
		fakeAvailableTiProxy("aaa-yyy", proxyg, newRevision),
	}
	for _, proxy := range proxies {
		proxy.Annotations[v1alpha1.AnnoKeyTiProxyHPAScaleIn] = v1alpha1.AnnoValTrue
	}
	s := &state{
		proxyg:         proxyg,
		cluster:        fake.FakeObj[v1alpha1.Cluster]("cluster"),
		proxies:        proxies,
		updateRevision: newRevision,
	}
	s.IFeatureGates = stateutil.NewFeatureGates[scope.TiProxyGroup](s)

	fc := client.NewFakeClient(proxyg, s.cluster)
	for _, obj := range s.proxies {
		require.NoError(t, fc.Apply(ctx, obj.DeepCopy()))
		require.NoError(t, fc.Status().Update(ctx, obj.DeepCopy()))
	}

	res, done := task.RunTask(ctx, TaskUpdater(&ReconcileContext{State: s}, fc, tracker.New().AllocateFactory("tiproxy")))
	require.False(t, done)
	assert.NotEqual(t, task.SFail.String(), res.Status().String(), res.Message())

	list := v1alpha1.TiProxyList{}
	require.NoError(t, fc.List(ctx, &list))

	var offlined int
	for i := range list.Items {
		item := &list.Items[i]
		if item.Spec.Offline == nil || !*item.Spec.Offline {
			continue
		}
		offlined++
		assert.Empty(t, item.Annotations[v1alpha1.AnnoKeyTiProxyHPAScaleIn])
	}
	assert.Equal(t, 1, offlined)
}
