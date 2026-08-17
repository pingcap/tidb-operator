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
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	tiproxyapi "github.com/pingcap/tidb-operator/v2/pkg/tiproxyapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type reviveTestHealthServer struct {
	server *httptest.Server

	mu                sync.Mutex
	healthStatus      int
	clearHealthStatus int
	clearHealthCalls  int
	overrideCleared   bool
}

func newReviveTestHealthServer(t *testing.T) *reviveTestHealthServer {
	t.Helper()

	s := &reviveTestHealthServer{
		healthStatus:      http.StatusBadGateway,
		clearHealthStatus: http.StatusOK,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/debug/health", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		switch r.Method {
		case http.MethodDelete:
			s.clearHealthCalls++
			w.WriteHeader(s.clearHealthStatus)
			if s.clearHealthStatus == http.StatusOK {
				s.overrideCleared = true
				s.healthStatus = http.StatusOK
			}
		default:
			w.WriteHeader(s.healthStatus)
			_, _ = w.Write([]byte(`{"config_checksum":1}`))
		}
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	s.server = server
	return s
}

func (s *reviveTestHealthServer) port(t *testing.T) int32 {
	t.Helper()

	u, err := url.Parse(s.server.URL)
	require.NoError(t, err)

	_, portStr, err := net.SplitHostPort(u.Host)
	require.NoError(t, err)

	port, err := strconv.ParseInt(portStr, 10, 32)
	require.NoError(t, err)
	return int32(port)
}

func revivableTiProxyWithAPI(apiPort int32) *v1alpha1.TiProxy {
	return fake.FakeObj("aaa-proxy-0", func(obj *v1alpha1.TiProxy) *v1alpha1.TiProxy {
		obj.Namespace = corev1.NamespaceDefault
		obj.Spec.Cluster.Name = "aaa"
		obj.Spec.Version = fakeVersion
		obj.Spec.Subdomain = "tiproxy-peer"
		obj.Spec.Server.Ports.API = &v1alpha1.Port{Port: apiPort}
		obj.Spec.Offline = ptr.To(false)
		obj.Annotations = map[string]string{
			v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
		}
		return obj
	})
}

func revivableTiProxyPod(cluster *v1alpha1.Cluster, tiproxy *v1alpha1.TiProxy, beginTime string) *corev1.Pod {
	pod := fakePod(cluster, tiproxy)
	pod.Annotations = map[string]string{
		v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime: beginTime,
	}
	return pod
}

func reviveTestReconcileContext(t *testing.T, s *state, healthServer *reviveTestHealthServer) *ReconcileContext {
	t.Helper()

	rc := &ReconcileContext{State: s}
	if healthServer != nil {
		rc.TiProxyClient = tiproxyapi.NewTiProxyClient(
			fmt.Sprintf("127.0.0.1:%d", healthServer.port(t)),
			tiproxyRequestTimeout,
			nil,
		)
	}
	return rc
}

func TestTaskReviveFromScaleInClearsPodDrainState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	healthServer := newReviveTestHealthServer(t)
	cluster := localTestCluster()
	tiproxy := revivableTiProxyWithAPI(healthServer.port(t))
	beginTime := time.Now().Format(time.RFC3339Nano)
	pod := revivableTiProxyPod(cluster, tiproxy, beginTime)

	s := &state{cluster: cluster, tiproxy: tiproxy, pod: pod}
	rc := reviveTestReconcileContext(t, s, healthServer)
	fc := client.NewFakeClient(cluster, tiproxy, pod)

	res, done := task.RunTask(ctx, TaskReviveFromScaleIn(rc, fc))
	require.False(t, done)
	assert.Equal(t, task.SComplete.String(), res.Status().String())

	actualPod := &corev1.Pod{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(pod), actualPod))
	assert.Empty(t, actualPod.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime])
	assert.True(t, s.IsHealthy())
	assert.Equal(t, 1, healthServer.clearHealthCalls)
}

func TestTaskReviveFromScaleInAbandonOnUnsupportedHealthOverride(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	healthServer := newReviveTestHealthServer(t)
	healthServer.clearHealthStatus = http.StatusNotFound

	cluster := localTestCluster()
	tiproxy := revivableTiProxyWithAPI(healthServer.port(t))
	beginTime := time.Now().Format(time.RFC3339Nano)
	pod := revivableTiProxyPod(cluster, tiproxy, beginTime)

	s := &state{cluster: cluster, tiproxy: tiproxy, pod: pod}
	rc := reviveTestReconcileContext(t, s, healthServer)
	fc := client.NewFakeClient(cluster, tiproxy, pod)

	res, done := task.RunTask(ctx, TaskReviveFromScaleIn(rc, fc))
	require.False(t, done)
	assert.Equal(t, task.SComplete.String(), res.Status().String())
	assert.Contains(t, res.Message(), "abandon revive")

	actual := &v1alpha1.TiProxy{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(tiproxy), actual))
	require.NotNil(t, actual.Spec.Offline)
	assert.True(t, *actual.Spec.Offline)
	assert.Equal(t, v1alpha1.AnnoValTrue, actual.Annotations[v1alpha1.AnnoKeyTiProxyReviveAbandoned])

	actualPod := &corev1.Pod{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(pod), actualPod))
	assert.Equal(t, beginTime, actualPod.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime])
	assert.False(t, s.IsHealthy())
	assert.Equal(t, 1, healthServer.clearHealthCalls)
}

func TestTaskReviveFromScaleInAbandonOnMethodNotAllowedHealthOverride(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	healthServer := newReviveTestHealthServer(t)
	healthServer.clearHealthStatus = http.StatusMethodNotAllowed

	cluster := localTestCluster()
	tiproxy := revivableTiProxyWithAPI(healthServer.port(t))
	beginTime := time.Now().Format(time.RFC3339Nano)
	pod := revivableTiProxyPod(cluster, tiproxy, beginTime)

	s := &state{cluster: cluster, tiproxy: tiproxy, pod: pod}
	rc := reviveTestReconcileContext(t, s, healthServer)
	fc := client.NewFakeClient(cluster, tiproxy, pod)

	res, done := task.RunTask(ctx, TaskReviveFromScaleIn(rc, fc))
	require.False(t, done)
	assert.Equal(t, task.SComplete.String(), res.Status().String())
	assert.Contains(t, res.Message(), "abandon revive")

	actual := &v1alpha1.TiProxy{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(tiproxy), actual))
	require.NotNil(t, actual.Spec.Offline)
	assert.True(t, *actual.Spec.Offline)
	assert.Equal(t, v1alpha1.AnnoValTrue, actual.Annotations[v1alpha1.AnnoKeyTiProxyReviveAbandoned])
	assert.Equal(t, 1, healthServer.clearHealthCalls)
}

func TestTaskReviveFromScaleInAbandonRetriesOnConflict(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	healthServer := newReviveTestHealthServer(t)
	healthServer.clearHealthStatus = http.StatusNotFound

	cluster := localTestCluster()
	tiproxy := revivableTiProxyWithAPI(healthServer.port(t))
	beginTime := time.Now().Format(time.RFC3339Nano)
	pod := revivableTiProxyPod(cluster, tiproxy, beginTime)
	fc := client.NewFakeClient(cluster, tiproxy, pod)
	fc.WithError("patch", "*", apierrors.NewConflict(
		schema.GroupResource{Group: "core.pingcap.com", Resource: "tiproxies"},
		tiproxy.Name,
		errors.New("resource version conflict"),
	))

	s := &state{cluster: cluster, tiproxy: tiproxy, pod: pod}
	rc := reviveTestReconcileContext(t, s, healthServer)

	res, done := task.RunTask(ctx, TaskReviveFromScaleIn(rc, fc))
	require.False(t, done)
	assert.Equal(t, task.SRetry.String(), res.Status().String())
	assert.Contains(t, res.Message(), "abandon revive")

	actual := &v1alpha1.TiProxy{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(tiproxy), actual))
	assert.Empty(t, actual.Annotations[v1alpha1.AnnoKeyTiProxyReviveAbandoned])
	assert.False(t, coreutil.IsOffline[scope.TiProxy](actual))
}

func TestTaskReviveFromScaleInRetriesHealthClearBeforePodAnnotationCleanup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	healthServer := newReviveTestHealthServer(t)
	healthServer.clearHealthStatus = http.StatusInternalServerError

	cluster := localTestCluster()
	tiproxy := revivableTiProxyWithAPI(healthServer.port(t))
	beginTime := time.Now().Format(time.RFC3339Nano)
	pod := revivableTiProxyPod(cluster, tiproxy, beginTime)

	s := &state{cluster: cluster, tiproxy: tiproxy, pod: pod}
	rc := reviveTestReconcileContext(t, s, healthServer)
	fc := client.NewFakeClient(cluster, tiproxy, pod)

	res, done := task.RunTask(ctx, TaskReviveFromScaleIn(rc, fc))
	require.False(t, done)
	assert.Equal(t, task.SRetry.String(), res.Status().String())

	actualPod := &corev1.Pod{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(pod), actualPod))
	assert.Equal(t, beginTime, actualPod.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime])
	assert.False(t, s.IsHealthy())
}

func TestTaskReviveFromScaleInUsesPodDrainState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	healthServer := newReviveTestHealthServer(t)
	cluster := localTestCluster()
	tiproxy := revivableTiProxyWithAPI(healthServer.port(t))
	pod := revivableTiProxyPod(cluster, tiproxy, time.Now().Format(time.RFC3339Nano))

	s := &state{cluster: cluster, tiproxy: tiproxy, pod: pod}
	rc := reviveTestReconcileContext(t, s, healthServer)
	fc := client.NewFakeClient(cluster, tiproxy, pod)

	res, done := task.RunTask(ctx, TaskReviveFromScaleIn(rc, fc))
	require.False(t, done)
	assert.Equal(t, task.SComplete.String(), res.Status().String())
	assert.True(t, s.IsHealthy())
}

func TestTaskReviveFromScaleInPodGoneSkipsHealthClear(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	healthServer := newReviveTestHealthServer(t)
	cluster := localTestCluster()
	tiproxy := revivableTiProxyWithAPI(healthServer.port(t))

	s := &state{cluster: cluster, tiproxy: tiproxy, pod: nil}
	rc := reviveTestReconcileContext(t, s, nil)
	fc := client.NewFakeClient(cluster, tiproxy)

	res, done := task.RunTask(ctx, TaskReviveFromScaleIn(rc, fc))
	require.False(t, done)
	assert.Equal(t, task.SComplete.String(), res.Status().String())
	assert.Contains(t, res.Message(), "does not need scale-in revive")
	assert.Equal(t, 0, healthServer.clearHealthCalls)
	assert.False(t, s.IsHealthy())
}

func TestCondTiProxyNeedsScaleInRevive(t *testing.T) {
	t.Parallel()

	cluster := localTestCluster()
	tiproxy := revivableTiProxyWithAPI(1234)
	beginTime := time.Now().Format(time.RFC3339Nano)
	pod := revivableTiProxyPod(cluster, tiproxy, beginTime)

	cases := []struct {
		desc     string
		state    *state
		expected bool
	}{
		{
			desc:     "pod has graceful shutdown begin time",
			state:    &state{tiproxy: tiproxy, pod: pod},
			expected: true,
		},
		{
			desc:     "pod missing begin time annotation",
			state:    &state{tiproxy: tiproxy, pod: fakePod(cluster, tiproxy)},
			expected: false,
		},
		{
			desc:     "pod is nil",
			state:    &state{tiproxy: tiproxy, pod: nil},
			expected: false,
		},
		{
			desc: "tiproxy is offline",
			state: func() *state {
				offline := tiproxy.DeepCopy()
				offline.Spec.Offline = ptr.To(true)
				return &state{tiproxy: offline, pod: pod}
			}(),
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, CondTiProxyNeedsScaleInRevive(tc.state).Satisfy())
		})
	}
}

func TestReviveFromScaleInRetryStopsFollowingTasks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	healthServer := newReviveTestHealthServer(t)
	healthServer.clearHealthStatus = http.StatusInternalServerError

	cluster := localTestCluster()
	tiproxy := revivableTiProxyWithAPI(healthServer.port(t))
	beginTime := time.Now().Format(time.RFC3339Nano)
	pod := revivableTiProxyPod(cluster, tiproxy, beginTime)

	s := &state{cluster: cluster, tiproxy: tiproxy, pod: pod}
	rc := reviveTestReconcileContext(t, s, healthServer)
	fc := client.NewFakeClient(cluster, tiproxy, pod)

	var followingTaskRan bool
	pipeline := task.Block(
		task.IfBreak(CondTiProxyNeedsScaleInRevive(s),
			TaskReviveFromScaleIn(rc, fc),
		),
		task.NameTaskFunc("Following", func(context.Context) task.Result {
			followingTaskRan = true
			return task.Complete().With("following task ran")
		}),
	)

	res, done := task.RunTask(ctx, pipeline)
	require.True(t, done)
	assert.Equal(t, task.SRetry.String(), res.Status().String())
	assert.False(t, followingTaskRan)
}

func offlinedTiProxyForDelete() *v1alpha1.TiProxy {
	return fake.FakeObj("aaa-proxy-0", func(obj *v1alpha1.TiProxy) *v1alpha1.TiProxy {
		obj.Namespace = corev1.NamespaceDefault
		obj.Spec.Cluster.Name = "aaa"
		obj.Spec.Version = fakeVersion
		obj.Spec.Offline = ptr.To(true)
		obj.Annotations = map[string]string{
			v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
		}
		return obj
	})
}

func TestTaskOfflineScaleInDrainPersistsBeginTimeBeforeMarkUnhealthy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := newTestTiProxyHealthServer(t, http.StatusOK, http.StatusOK)
	cluster := localTestCluster()
	tiproxy := revivableTiProxyWithAPI(server.port(t))
	tiproxy.Spec.Offline = ptr.To(true)
	pod := fakePod(cluster, tiproxy)

	s := &state{cluster: cluster, tiproxy: tiproxy, pod: pod}
	rc := &ReconcileContext{
		State: s,
		TiProxyClient: tiproxyapi.NewTiProxyClient(
			fmt.Sprintf("127.0.0.1:%d", server.port(t)),
			tiproxyRequestTimeout,
			nil,
		),
	}
	fc := client.NewFakeClient(cluster, tiproxy, pod)

	res, done := task.RunTask(ctx, TaskOfflineScaleInDrain(rc, fc))
	require.False(t, done)
	assert.Equal(t, task.SRetry.String(), res.Status().String())

	actualPod := &corev1.Pod{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(pod), actualPod))
	assert.NotEmpty(t, actualPod.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime])

	healthCalls, markUnhealthyCalls := server.counts()
	assert.Equal(t, 2, healthCalls)
	assert.Equal(t, 1, markUnhealthyCalls)
}

func TestTaskOfflineScaleInDrainSkipsMarkUnhealthyWhenBeginTimeWriteFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := newTestTiProxyHealthServer(t, http.StatusOK, http.StatusOK)
	cluster := localTestCluster()
	tiproxy := revivableTiProxyWithAPI(server.port(t))
	tiproxy.Spec.Offline = ptr.To(true)
	pod := fakePod(cluster, tiproxy)

	s := &state{cluster: cluster, tiproxy: tiproxy, pod: pod}
	rc := &ReconcileContext{
		State: s,
		TiProxyClient: tiproxyapi.NewTiProxyClient(
			fmt.Sprintf("127.0.0.1:%d", server.port(t)),
			tiproxyRequestTimeout,
			nil,
		),
	}
	fc := client.NewFakeClient(cluster, tiproxy, pod)
	fc.WithError("update", "*", apierrors.NewConflict(
		schema.GroupResource{Resource: "pods"},
		pod.Name,
		errors.New("resource version conflict"),
	))

	res, done := task.RunTask(ctx, TaskOfflineScaleInDrain(rc, fc))
	require.False(t, done)
	assert.Equal(t, task.SFail.String(), res.Status().String())

	actualPod := &corev1.Pod{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(pod), actualPod))
	assert.Empty(t, actualPod.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime])

	_, markUnhealthyCalls := server.counts()
	assert.Equal(t, 0, markUnhealthyCalls)
}

func TestTaskDeleteOfflinedTiProxyDeletesWhenOffline(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cluster := localTestCluster()
	tiproxy := offlinedTiProxyForDelete()

	s := &state{cluster: cluster, tiproxy: tiproxy}
	fc := client.NewFakeClient(cluster, tiproxy)

	res, done := task.RunTask(ctx, TaskDeleteOfflinedTiProxy(s, fc))
	require.False(t, done)
	assert.Equal(t, task.SWait.String(), res.Status().String())

	actual := &v1alpha1.TiProxy{}
	err := fc.Get(ctx, client.ObjectKeyFromObject(tiproxy), actual)
	if err == nil {
		assert.False(t, actual.GetDeletionTimestamp().IsZero())
	}
}

func TestTaskDeleteOfflinedTiProxySkipsWhenNoLongerOffline(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cluster := localTestCluster()
	tiproxy := offlinedTiProxyForDelete()
	tiproxy.Spec.Offline = ptr.To(false)

	s := &state{cluster: cluster, tiproxy: tiproxy}
	fc := client.NewFakeClient(cluster, tiproxy)

	res, done := task.RunTask(ctx, TaskDeleteOfflinedTiProxy(s, fc))
	require.False(t, done)
	assert.Equal(t, task.SComplete.String(), res.Status().String())
	assert.Contains(t, res.Message(), "skip delete")

	actual := &v1alpha1.TiProxy{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(tiproxy), actual))
	assert.True(t, actual.GetDeletionTimestamp().IsZero())
}

func TestTaskDeleteOfflinedTiProxyRetriesOnConflict(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cluster := localTestCluster()
	tiproxy := offlinedTiProxyForDelete()
	fc := client.NewFakeClient(cluster, tiproxy)
	fc.WithError("delete", "*", apierrors.NewConflict(
		schema.GroupResource{Group: "core.pingcap.com", Resource: "tiproxies"},
		tiproxy.Name,
		errors.New("resource version conflict"),
	))

	s := &state{cluster: cluster, tiproxy: tiproxy}
	res, done := task.RunTask(ctx, TaskDeleteOfflinedTiProxy(s, fc))
	require.False(t, done)
	assert.Equal(t, task.SRetry.String(), res.Status().String())
	assert.Contains(t, res.Message(), "retry")

	actual := &v1alpha1.TiProxy{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(tiproxy), actual))
	assert.True(t, actual.GetDeletionTimestamp().IsZero())
}
