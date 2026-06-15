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
	"crypto/tls"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/tiproxyapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TaskDrainPodForDelete(state State, c client.Client) task.Task {
	return task.NameTaskFunc("DrainPodForDelete", func(ctx context.Context) task.Result {
		pod := state.Pod()
		if pod == nil {
			return task.Complete().With("pod doesn't exist")
		}

		retryAfter, err := drainOrDeletePod(ctx, c, state, pod)
		if err != nil {
			return task.Fail().With("cannot delete pod of tiproxy: %v", err)
		}
		if retryAfter > 0 {
			return task.Retry(retryAfter).With("wait for tiproxy pod to be deleted")
		}
		return task.Complete().With("pod is deleted")
	})
}

func drainOrDeletePod(ctx context.Context, c client.Client, state State, pod *corev1.Pod) (time.Duration, error) {
	logger := logr.FromContextOrDiscard(ctx)
	tiproxy := state.Object()

	if !pod.GetDeletionTimestamp().IsZero() {
		return task.DefaultRequeueAfter, nil
	}

	seconds, ok, err := gracefulShutdownDeleteDelaySeconds(tiproxy)
	if err != nil {
		return 0, err
	}
	if !ok || seconds <= 0 {
		return deleteTiProxyPod(ctx, c, pod)
	}

	startAt, ok := gracefulShutdownBeginTime(pod)
	if !ok {
		if !ensureTiProxyMarkedUnhealthy(ctx, state, c, logger) {
			return task.DefaultRequeueAfter, nil
		}
		startAt = time.Now()
		if err := markGracefulShutdownBeginTime(ctx, c, pod, startAt); err != nil {
			return 0, err
		}
	}

	remaining := time.Until(startAt.Add(time.Duration(seconds) * time.Second))
	if remaining <= 0 {
		return deleteTiProxyPod(ctx, c, pod)
	}

	if tiProxyConnectionsDrained(ctx, state, c, logger) {
		return deleteTiProxyPod(ctx, c, pod)
	}

	if remaining > task.DefaultRequeueAfter {
		remaining = task.DefaultRequeueAfter
	}

	return remaining, nil
}

func gracefulShutdownDeleteDelaySeconds(tiproxy *v1alpha1.TiProxy) (seconds int32, ok bool, err error) {
	raw := tiproxy.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds]
	if raw == "" {
		return 0, false, nil
	}

	parsed, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, false, err
	}
	return int32(parsed), true, nil
}

func gracefulShutdownBeginTime(pod *corev1.Pod) (time.Time, bool) {
	raw := pod.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime]
	if raw == "" {
		return time.Time{}, false
	}

	startAt, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, false
	}
	return startAt, true
}

func markGracefulShutdownBeginTime(ctx context.Context, c client.Client, pod *corev1.Pod, startAt time.Time) error {
	newPod := pod.DeepCopy()
	if newPod.Annotations == nil {
		newPod.Annotations = map[string]string{}
	}
	newPod.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime] = startAt.Format(time.RFC3339Nano)
	return c.Update(ctx, newPod)
}

func deleteTiProxyPod(ctx context.Context, c client.Client, pod *corev1.Pod) (time.Duration, error) {
	if err := c.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
		return 0, err
	}
	return task.DefaultRequeueAfter, nil
}

func tiProxyConnectionsDrained(ctx context.Context, state State, c client.Client, logger logr.Logger) bool {
	tiproxy := state.Object()

	tpClient, err := newTiProxyDeleteClient(ctx, state, c)
	if err != nil {
		logger.Info(
			"failed to build TiProxy API client before checking connections, continue waiting",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false
	}

	connectionCount, err := tpClient.ConnectionCount(ctx)
	if err != nil {
		logger.Info(
			"failed to query TiProxy connections before graceful delete, continue waiting",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false
	}
	if connectionCount > 0 {
		logger.Info(
			"TiProxy still has active connections, continue waiting",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"connectionCount", connectionCount,
		)
		return false
	}

	logger.Info(
		"TiProxy has no active connections, delete pod without waiting for the remaining graceful delay",
		"namespace", tiproxy.Namespace,
		"name", tiproxy.Name,
	)
	return true
}

func ensureTiProxyMarkedUnhealthy(ctx context.Context, state State, c client.Client, logger logr.Logger) bool {
	tiproxy := state.Object()

	tpClient, err := newTiProxyDeleteClient(ctx, state, c)
	if err != nil {
		logger.Info(
			"failed to build TiProxy API client before graceful delete, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false
	}

	healthy, err := tpClient.IsHealthy(ctx)
	if err != nil {
		logger.Info(
			"failed to query TiProxy health before graceful delete retry, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false
	}
	if !healthy {
		return true
	}

	if err := tpClient.MarkUnhealthy(ctx); err != nil {
		logger.Info(
			"failed to mark TiProxy unhealthy before graceful delete, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false
	}

	healthy, err = tpClient.IsHealthy(ctx)
	if err != nil {
		logger.Info(
			"failed to re-check TiProxy health after graceful delete action, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false
	}
	if healthy {
		logger.Info(
			"TiProxy health is still healthy after graceful delete action, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
		)
		return false
	}
	return true
}

func newTiProxyDeleteClient(ctx context.Context, state State, c client.Client) (tiproxyapi.TiProxyClient, error) {
	ck := state.Cluster()

	var tlsConfig *tls.Config
	if coreutil.IsTiProxyHTTPServerTLSEnabled(ck, state.Object()) {
		var err error
		tlsConfig, err = apicall.GetClientTLSConfig(ctx, c, ck)
		if err != nil {
			return nil, err
		}
	}

	tiproxy := state.TiProxy()
	addr := coreutil.InstanceAdvertiseAddress[scope.TiProxy](ck, tiproxy, coreutil.TiProxyAPIPort(tiproxy))
	return tiproxyapi.NewTiProxyClient(addr, tiproxyRequestTimeout, tlsConfig), nil
}
