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

package tiproxy

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

func TestLegacyInvalidInstanceClusterDeletion(t *testing.T) {
	now := metav1.Now()
	cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{
		Name: "cluster", Namespace: "test", DeletionTimestamp: &now, Finalizers: []string{metav1alpha1.Finalizer},
	}}
	instance := &v1alpha1.TiProxy{ObjectMeta: metav1.ObjectMeta{Name: "invalid", Namespace: "test", Finalizers: []string{metav1alpha1.Finalizer}}}
	instance.Spec.Cluster.Name = cluster.Name
	pod := fake.FakeObj("managed-pod", fake.InstanceOwner[scope.TiProxy, corev1.Pod](instance))
	pod.Namespace = instance.Namespace
	pod.Labels[v1alpha1.LabelKeyCluster] = cluster.Name
	c := client.NewFakeClient(cluster, instance, pod)
	r := &Reconciler{Client: c, Tracker: tracker.New().Tracker("tiproxy")}
	key := client.ObjectKeyFromObject(instance)
	result, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	require.Positive(t, result.RequeueAfter)
	var remaining v1alpha1.TiProxy
	require.NoError(t, c.Get(t.Context(), key, &remaining))
	require.Contains(t, remaining.Finalizers, metav1alpha1.Finalizer)
	require.True(t, errors.IsNotFound(c.Get(t.Context(), client.ObjectKeyFromObject(pod), &corev1.Pod{})))
	_, err = r.Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	var got v1alpha1.TiProxy
	require.NoError(t, c.Get(t.Context(), key, &got))
	require.NotContains(t, got.Finalizers, metav1alpha1.Finalizer)
}

func TestLegacyInvalidInstanceDeletionWithoutPod(t *testing.T) {
	now := metav1.Now()
	cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "test"}}
	cluster.Status.PD = "http://pd:2379"
	instance := &v1alpha1.TiProxy{ObjectMeta: metav1.ObjectMeta{
		Name: "invalid", Namespace: "test", DeletionTimestamp: &now, Finalizers: []string{metav1alpha1.Finalizer},
	}}
	instance.Spec.Cluster.Name = cluster.Name
	c := client.NewFakeClient(cluster, instance)
	r := &Reconciler{Client: c, Tracker: tracker.New().Tracker("tiproxy")}
	key := client.ObjectKeyFromObject(instance)
	_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	var got v1alpha1.TiProxy
	err = c.Get(t.Context(), key, &got)
	if !errors.IsNotFound(err) {
		require.NoError(t, err)
		require.NotContains(t, got.Finalizers, metav1alpha1.Finalizer)
	}
}

func TestLegacyInvalidInstanceDeletionPodSafety(t *testing.T) {
	for _, tc := range []struct {
		name           string
		foreignOwner   bool
		multiplePods   bool
		delay          string
		wantError      bool
		wantPodDeleted bool
	}{
		{name: "owned pod without delay", wantPodDeleted: true},
		{name: "owned pod with graceful delay", delay: "3600"},
		{name: "foreign owner", foreignOwner: true, wantError: true},
		{name: "multiple matching pods", multiplePods: true, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := metav1.Now()
			cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "test"}}
			cluster.Status.PD = "http://pd:2379"
			instance := &v1alpha1.TiProxy{ObjectMeta: metav1.ObjectMeta{
				Name: "invalid", Namespace: "test", UID: "instance-uid", DeletionTimestamp: &now,
				Finalizers:  []string{metav1alpha1.Finalizer},
				Annotations: map[string]string{v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: tc.delay},
			}}
			instance.Spec.Cluster.Name = cluster.Name
			pod := fake.FakeObj("adopted-pod", fake.InstanceOwner[scope.TiProxy, corev1.Pod](instance))
			pod.Namespace = instance.Namespace
			pod.Labels[v1alpha1.LabelKeyCluster] = cluster.Name
			if tc.foreignOwner {
				pod.OwnerReferences[0].UID = "another-instance"
			}
			objects := []client.Object{cluster, instance, pod}
			if tc.multiplePods {
				another := pod.DeepCopy()
				another.Name = "another-pod"
				objects = append(objects, another)
			}
			c := client.NewFakeClient(objects...)
			r := &Reconciler{Client: c, Tracker: tracker.New().Tracker("tiproxy")}
			key := client.ObjectKeyFromObject(instance)
			result, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
			if tc.wantError {
				require.ErrorContains(t, err, "cannot get pod")
			} else {
				require.NoError(t, err)
				require.Positive(t, result.RequeueAfter)
			}
			var remaining v1alpha1.TiProxy
			require.NoError(t, c.Get(t.Context(), key, &remaining))
			require.Contains(t, remaining.Finalizers, metav1alpha1.Finalizer)
			err = c.Get(t.Context(), client.ObjectKeyFromObject(pod), &corev1.Pod{})
			if tc.wantPodDeleted {
				require.True(t, errors.IsNotFound(err))
				_, err = r.Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
				require.NoError(t, err)
				require.NoError(t, c.Get(t.Context(), key, &remaining))
				require.NotContains(t, remaining.Finalizers, metav1alpha1.Finalizer)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInvalidInstanceNormalReconcile(t *testing.T) {
	cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "test"}}
	cluster.Status.PD = "http://pd:2379"
	instance := &v1alpha1.TiProxy{ObjectMeta: metav1.ObjectMeta{Name: "invalid", Namespace: "test"}}
	instance.Spec.Cluster.Name = cluster.Name
	instance.Spec.Volumes = []v1alpha1.Volume{{Name: "data"}}
	c := client.NewFakeClient(cluster, instance)
	r := &Reconciler{Client: c, Tracker: tracker.New().Tracker("tiproxy")}
	_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(instance)})
	require.ErrorContains(t, err, "invalid instance")
}
