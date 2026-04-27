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
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type deleteOptionRecorderClient struct {
	client.Client
	lastDeleteOptions ctrlclient.DeleteOptions
}

func (c *deleteOptionRecorderClient) Delete(ctx context.Context, obj ctrlclient.Object, opts ...ctrlclient.DeleteOption) error {
	c.lastDeleteOptions = ctrlclient.DeleteOptions{}
	c.lastDeleteOptions.ApplyOptions(opts)
	return c.Client.Delete(ctx, obj, opts...)
}

type testTiProxyHealthServer struct {
	server *httptest.Server

	mu                  sync.Mutex
	healthStatus        int
	markUnhealthyStatus int
	healthCalls         int
	markUnhealthyCalls  int
}

func newTestTiProxyHealthServer(t *testing.T, healthStatus, markUnhealthyStatus int) *testTiProxyHealthServer {
	t.Helper()

	s := &testTiProxyHealthServer{
		healthStatus:        healthStatus,
		markUnhealthyStatus: markUnhealthyStatus,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/debug/health", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.healthCalls++
		w.WriteHeader(s.healthStatus)
		_, _ = w.Write([]byte(`{"config_checksum":1}`))
	})
	mux.HandleFunc("/api/debug/health/unhealthy", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.markUnhealthyCalls++
		w.WriteHeader(s.markUnhealthyStatus)
		if s.markUnhealthyStatus == http.StatusOK {
			s.healthStatus = http.StatusBadGateway
		}
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	s.server = server
	return s
}

func (s *testTiProxyHealthServer) port(t *testing.T) int32 {
	t.Helper()
	u, err := url.Parse(s.server.URL)
	require.NoError(t, err)
	port := u.Port()
	value, err := strconv.ParseInt(port, 10, 32)
	require.NoError(t, err)
	return int32(value)
}

func (s *testTiProxyHealthServer) counts() (health, markUnhealthy int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.healthCalls, s.markUnhealthyCalls
}

func runDrainPodForDeleteTask(t *testing.T, ctx context.Context, s State, c client.Client) (task.Result, bool) {
	t.Helper()

	contextRes, contextDone := task.RunTask(ctx, common.TaskContextPod[scope.TiProxy](s, c))
	require.NotNil(t, contextRes)
	require.False(t, contextDone)
	require.NotEqual(t, task.SFail.String(), contextRes.Status().String(), contextRes.Message())

	return task.RunTask(ctx, TaskDrainPodForDelete(s, c, nil))
}

func TestTaskDrainPodForDeleteDeletePodImmediatelyByDefault(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tiproxy := deletingTiProxy("")
	pod := fakePod(localTestCluster(), tiproxy)

	fc := client.NewFakeClient(tiproxy, pod)
	res, done := runDrainPodForDeleteTask(t, ctx, &state{tiproxy: tiproxy}, fc)

	assert.Equal(t, task.SRetry.String(), res.Status().String())
	assert.False(t, done)

	err := fc.Get(ctx, client.ObjectKeyFromObject(pod), &corev1.Pod{})
	assert.True(t, apierrors.IsNotFound(err))
}

func TestTaskDrainPodForDeleteMarkPodUnhealthyBeforeDeleting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := newTestTiProxyHealthServer(t, http.StatusOK, http.StatusOK)
	cluster := localTestCluster()
	tiproxy := deletingTiProxyWithAPIAddress("3600", server.port(t))
	pod := fakePod(cluster, tiproxy)

	fc := client.NewFakeClient(cluster, tiproxy, pod)
	res, done := runDrainPodForDeleteTask(t, ctx, &state{cluster: cluster, tiproxy: tiproxy}, fc)

	assert.Equal(t, task.SRetry.String(), res.Status().String())
	assert.False(t, done)

	healthCalls, markUnhealthyCalls := server.counts()
	assert.Equal(t, 2, healthCalls)
	assert.Equal(t, 1, markUnhealthyCalls)

	actual := &corev1.Pod{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(pod), actual))
	assert.NotEmpty(t, actual.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime])
}

func TestTaskDrainPodForDeleteDeletePodAfterDrainDelay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := newTestTiProxyHealthServer(t, http.StatusBadGateway, http.StatusOK)
	cluster := localTestCluster()
	tiproxy := deletingTiProxyWithAPIAddress("60", server.port(t))
	pod := fakePod(cluster, tiproxy)
	pod.Annotations = map[string]string{
		v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime: time.Now().Add(-2 * time.Minute).Format(time.RFC3339Nano),
	}

	fc := client.NewFakeClient(cluster, tiproxy, pod)
	res, done := runDrainPodForDeleteTask(t, ctx, &state{cluster: cluster, tiproxy: tiproxy}, fc)

	assert.Equal(t, task.SRetry.String(), res.Status().String())
	assert.False(t, done)

	healthCalls, markUnhealthyCalls := server.counts()
	assert.Equal(t, 0, healthCalls)
	assert.Equal(t, 0, markUnhealthyCalls)

	err := fc.Get(ctx, client.ObjectKeyFromObject(pod), &corev1.Pod{})
	assert.True(t, apierrors.IsNotFound(err))
}

func TestTaskDrainPodForDeleteContinueDeleteDelayWhenMarkUnhealthyFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := newTestTiProxyHealthServer(t, http.StatusOK, http.StatusInternalServerError)
	cluster := localTestCluster()
	tiproxy := deletingTiProxyWithAPIAddress("3600", server.port(t))
	pod := fakePod(cluster, tiproxy)

	fc := client.NewFakeClient(cluster, tiproxy, pod)
	res, done := runDrainPodForDeleteTask(t, ctx, &state{cluster: cluster, tiproxy: tiproxy}, fc)

	assert.Equal(t, task.SRetry.String(), res.Status().String())
	assert.False(t, done)

	healthCalls, markUnhealthyCalls := server.counts()
	assert.Equal(t, 1, healthCalls)
	assert.Equal(t, 1, markUnhealthyCalls)

	actual := &corev1.Pod{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(pod), actual))
	assert.Empty(t, actual.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime])
}

func TestTaskDrainPodForDeleteDoNotStartDeleteDelayWhenUnhealthyAPIIsUnsupportedButFallbackCannotRun(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := newTestTiProxyHealthServer(t, http.StatusOK, http.StatusNotFound)
	cluster := localTestCluster()
	tiproxy := deletingTiProxyWithAPIAddress("3600", server.port(t))
	pod := fakePod(cluster, tiproxy)
	s := &state{cluster: cluster, tiproxy: tiproxy}

	fc := client.NewFakeClient(cluster, tiproxy, pod)
	res, done := runDrainPodForDeleteTask(t, ctx, s, fc)

	assert.Equal(t, task.SRetry.String(), res.Status().String())
	assert.False(t, done)

	healthCalls, markUnhealthyCalls := server.counts()
	assert.Equal(t, 1, healthCalls)
	assert.Equal(t, 1, markUnhealthyCalls)

	actual := &corev1.Pod{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(pod), actual))
	assert.Empty(t, actual.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime])
}

func TestTaskDrainPodForDeleteWaitForDeleteDelayAfterGracefulShutdownStarted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := newTestTiProxyHealthServer(t, http.StatusOK, http.StatusOK)
	cluster := localTestCluster()
	tiproxy := deletingTiProxyWithAPIAddress("3600", server.port(t))
	pod := fakePod(cluster, tiproxy)
	pod.Annotations = map[string]string{
		v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime: time.Now().Add(-10 * time.Second).Format(time.RFC3339Nano),
	}

	fc := client.NewFakeClient(cluster, tiproxy, pod)
	res, done := runDrainPodForDeleteTask(t, ctx, &state{cluster: cluster, tiproxy: tiproxy}, fc)

	assert.Equal(t, task.SRetry.String(), res.Status().String())
	assert.False(t, done)

	healthCalls, markUnhealthyCalls := server.counts()
	assert.Equal(t, 0, healthCalls)
	assert.Equal(t, 0, markUnhealthyCalls)

	actual := &corev1.Pod{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(pod), actual))
	assert.True(t, actual.GetDeletionTimestamp().IsZero())
}

func TestTaskDrainPodForDeleteDeletePodWithoutExtendingGracePeriodAfterDelay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := newTestTiProxyHealthServer(t, http.StatusBadGateway, http.StatusOK)
	cluster := localTestCluster()
	tiproxy := deletingTiProxyWithAPIAddress("60", server.port(t))
	pod := fakePod(cluster, tiproxy)
	pod.Annotations = map[string]string{
		v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime: time.Now().Add(-2 * time.Minute).Format(time.RFC3339Nano),
	}

	recorder := &deleteOptionRecorderClient{Client: client.NewFakeClient(cluster, tiproxy, pod)}
	res, done := runDrainPodForDeleteTask(t, ctx, &state{cluster: cluster, tiproxy: tiproxy}, recorder)

	assert.Equal(t, task.SRetry.String(), res.Status().String())
	assert.False(t, done)
	require.Nil(t, recorder.lastDeleteOptions.GracePeriodSeconds)
}

func TestDrainPodForDeleteBlocksCommonFinalizerUntilPodIsGone(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := newTestTiProxyHealthServer(t, http.StatusOK, http.StatusOK)
	cluster := localTestCluster()
	tiproxy := deletingTiProxyWithAPIAddress("3600", server.port(t))
	pod := fakePod(cluster, tiproxy)
	s := &state{cluster: cluster, tiproxy: tiproxy}

	fc := client.NewFakeClient(cluster, tiproxy, pod)
	res, done := task.RunTask(ctx, task.Block(
		common.TaskContextPod[scope.TiProxy](s, fc),
		TaskDrainPodForDelete(s, fc, nil),
		task.If(task.CondFunc(func() bool { return s.Pod() == nil }),
			common.TaskInstanceFinalizerDel[scope.TiProxy](s, fc, common.DefaultInstanceSubresourceLister),
		),
	))

	assert.Equal(t, task.SRetry.String(), res.Status().String())
	assert.False(t, done)

	actual := &v1alpha1.TiProxy{}
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(tiproxy), actual))
	assert.NotEmpty(t, actual.Finalizers)
}

func localTestCluster() *v1alpha1.Cluster {
	return fake.FakeObj[v1alpha1.Cluster]("aaa", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
		obj.Namespace = corev1.NamespaceDefault
		obj.Spec.DNS = &v1alpha1.DNS{
			Mode:          v1alpha1.DNSModeFQDN,
			ClusterDomain: "localhost",
		}
		return obj
	})
}

func deletingTiProxy(deleteDelaySeconds string) *v1alpha1.TiProxy {
	now := metav1.Now()
	return fake.FakeObj("aaa-xxx",
		fake.DeleteTimestamp[v1alpha1.TiProxy](&now),
		fake.AddFinalizer[v1alpha1.TiProxy](),
		func(obj *v1alpha1.TiProxy) *v1alpha1.TiProxy {
			obj.Namespace = corev1.NamespaceDefault
			obj.Spec.Cluster.Name = "aaa"
			obj.Spec.Version = fakeVersion
			obj.Spec.Subdomain = "tiproxy-peer"
			if deleteDelaySeconds != "" {
				seconds, err := strconv.ParseInt(deleteDelaySeconds, 10, 32)
				if err != nil {
					panic(err)
				}
				obj.Spec.GracefulShutdownDeleteDelaySeconds = ptrTo[int32](int32(seconds))
			}
			return obj
		},
	)
}

func deletingTiProxyWithAPIAddress(deleteDelaySeconds string, apiPort int32) *v1alpha1.TiProxy {
	tiproxy := deletingTiProxy(deleteDelaySeconds)
	tiproxy.Spec.Server.Ports.API = &v1alpha1.Port{Port: apiPort}
	return tiproxy
}

func ptrTo[T any](v T) *T {
	return &v
}
