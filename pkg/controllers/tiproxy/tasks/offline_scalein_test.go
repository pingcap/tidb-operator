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
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
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
	host := s.server.URL
	host = host[len("http://"):]
	colon := len(host) - 1
	for colon >= 0 && host[colon] != ':' {
		colon--
	}
	value, err := strconv.ParseInt(host[colon+1:], 10, 32)
	require.NoError(t, err)
	return int32(value)
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

func TestTaskReviveFromScaleInClearsPodDrainState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	healthServer := newReviveTestHealthServer(t)
	cluster := localTestCluster()
	tiproxy := revivableTiProxyWithAPI(healthServer.port(t))
	beginTime := time.Now().Format(time.RFC3339Nano)
	pod := revivableTiProxyPod(cluster, tiproxy, beginTime)

	s := &state{cluster: cluster, tiproxy: tiproxy, pod: pod}
	fc := client.NewFakeClient(cluster, tiproxy, pod)

	res, done := task.RunTask(ctx, TaskReviveFromScaleIn(s, fc))
	require.False(t, done)
	assert.Equal(t, task.SComplete.String(), res.Status().String())

	actualPod := &corev1.Pod{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(pod), actualPod))
	assert.Empty(t, actualPod.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime])
	assert.True(t, s.IsHealthy())
	assert.Equal(t, 1, healthServer.clearHealthCalls)
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
	fc := client.NewFakeClient(cluster, tiproxy, pod)

	res, done := task.RunTask(ctx, TaskReviveFromScaleIn(s, fc))
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
	fc := client.NewFakeClient(cluster, tiproxy, pod)

	res, done := task.RunTask(ctx, TaskReviveFromScaleIn(s, fc))
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
	fc := client.NewFakeClient(cluster, tiproxy)

	res, done := task.RunTask(ctx, TaskReviveFromScaleIn(s, fc))
	require.False(t, done)
	assert.Equal(t, task.SComplete.String(), res.Status().String())
	assert.Contains(t, res.Message(), "does not need scale-in revive")
	assert.Equal(t, 0, healthServer.clearHealthCalls)
	assert.False(t, s.IsHealthy())
}

func TestNeedsScaleInRevive(t *testing.T) {
	t.Parallel()

	now := time.Now().Format(time.RFC3339Nano)
	offlineProxy := fake.FakeObj("offline", func(obj *v1alpha1.TiProxy) *v1alpha1.TiProxy {
		obj.Spec.Offline = ptr.To(true)
		return obj
	})
	podWithDrain := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime: now,
			},
		},
	}

	assert.False(t, needsScaleInRevive(offlineProxy, nil))
	assert.True(t, needsScaleInRevive(&v1alpha1.TiProxy{}, podWithDrain))
	assert.False(t, needsScaleInRevive(&v1alpha1.TiProxy{}, &corev1.Pod{}))
}

func TestOfflineScaleInDrainComplete(t *testing.T) {
	t.Parallel()

	now := time.Now()
	startAt := now.Add(-time.Minute).Format(time.RFC3339Nano)

	tiproxy := &v1alpha1.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
			},
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime: startAt,
			},
		},
	}

	complete, err := offlineScaleInDrainComplete(tiproxy, pod)
	require.NoError(t, err)
	assert.False(t, complete)

	deleting := now
	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime: startAt,
			},
			DeletionTimestamp: &metav1.Time{Time: deleting},
		},
	}
	complete, err = offlineScaleInDrainComplete(tiproxy, pod)
	require.NoError(t, err)
	assert.True(t, complete)

	complete, err = offlineScaleInDrainComplete(tiproxy, nil)
	require.NoError(t, err)
	assert.True(t, complete)
}
