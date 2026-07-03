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

func drainPodForGracefulShutdown(
	ctx context.Context,
	c client.Client,
	state State,
	pod *corev1.Pod,
	// forOfflineScaleIn distinguishes graceful scale-in drain from TiProxy CR deletion.
	// Both paths delete the pod as soon as tiproxy_server_connections reaches zero (#6936).
	//
	// Scale-in additionally guards against racing TiProxyGroup scale-out revival by
	// re-reading spec.offline before deleting the pod; skip delete if scale-out already cleared offline.
	forOfflineScaleIn bool,
) (time.Duration, error) {
	logger := logr.FromContextOrDiscard(ctx)
	tiproxy := state.Object()

	if !pod.GetDeletionTimestamp().IsZero() {
		return task.DefaultRequeueAfter, nil
	}

	seconds, ok, err := coreutil.GracefulShutdownDeleteDelaySeconds(tiproxy)
	if err != nil {
		return 0, err
	}
	if !ok || seconds <= 0 {
		return deleteTiProxyPod(ctx, c, pod)
	}

	// pod is guaranteed non-nil here (dereferenced above).
	startAt := coreutil.GracefulShutdownBeginTimeFromSources(tiproxy.Annotations, pod.Annotations)
	if startAt.IsZero() {
		if !ensureTiProxyMarkedUnhealthy(ctx, state, c, logger) {
			return task.DefaultRequeueAfter, nil
		}
		startAt = time.Now()
		if err := markGracefulShutdownBeginTime(ctx, c, tiproxy, pod, startAt); err != nil {
			return 0, err
		}
	}

	remaining := time.Until(startAt.Add(time.Duration(seconds) * time.Second))
	if remaining <= 0 {
		if err := guardOfflineScaleInBeforePodDelete(ctx, c, tiproxy, forOfflineScaleIn); err != nil {
			return 0, err
		}
		if forOfflineScaleIn && !coreutil.IsOffline[scope.TiProxy](tiproxy) {
			return task.DefaultRequeueAfter, nil
		}
		return deleteTiProxyPod(ctx, c, pod)
	}

	if tiProxyConnectionsDrained(ctx, state, c, logger) {
		if forOfflineScaleIn {
			if err := guardOfflineScaleInBeforePodDelete(ctx, c, tiproxy, forOfflineScaleIn); err != nil {
				return 0, err
			}
			if !coreutil.IsOffline[scope.TiProxy](tiproxy) {
				return task.DefaultRequeueAfter, nil
			}
		}
		return deleteTiProxyPod(ctx, c, pod)
	}

	if remaining > task.DefaultRequeueAfter {
		remaining = task.DefaultRequeueAfter
	}

	return remaining, nil
}

func tiProxyConnectionsDrained(ctx context.Context, state State, c client.Client, logger logr.Logger) bool {
	tiproxy := state.Object()

	tpClient, err := newTiProxyAPIClient(ctx, state, c)
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

func guardOfflineScaleInBeforePodDelete(
	ctx context.Context,
	c client.Client,
	tiproxy *v1alpha1.TiProxy,
	forOfflineScaleIn bool,
) error {
	if !forOfflineScaleIn {
		return nil
	}

	fresh := &v1alpha1.TiProxy{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(tiproxy), fresh); err != nil {
		return err
	}
	*tiproxy = *fresh
	return nil
}

func markGracefulShutdownBeginTime(
	ctx context.Context,
	c client.Client,
	tiproxy *v1alpha1.TiProxy,
	pod *corev1.Pod,
	startAt time.Time,
) error {
	formatted := startAt.Format(time.RFC3339Nano)

	newTiProxy := tiproxy.DeepCopy()
	if newTiProxy.Annotations == nil {
		newTiProxy.Annotations = map[string]string{}
	}
	newTiProxy.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime] = formatted
	if err := c.Update(ctx, newTiProxy); err != nil {
		return err
	}
	*tiproxy = *newTiProxy

	newPod := pod.DeepCopy()
	if newPod.Annotations == nil {
		newPod.Annotations = map[string]string{}
	}
	newPod.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime] = formatted
	return c.Update(ctx, newPod)
}

func deleteTiProxyPod(ctx context.Context, c client.Client, pod *corev1.Pod) (time.Duration, error) {
	if err := c.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
		return 0, err
	}
	return task.DefaultRequeueAfter, nil
}

func ensureTiProxyMarkedUnhealthy(ctx context.Context, state State, c client.Client, logger logr.Logger) bool {
	tiproxy := state.Object()

	tpClient, err := newTiProxyAPIClient(ctx, state, c)
	if err != nil {
		logger.Info(
			"failed to build TiProxy API client before graceful shutdown, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false
	}

	healthy, err := tpClient.IsHealthy(ctx)
	if err != nil {
		logger.Info(
			"failed to query TiProxy health before graceful shutdown, continue retrying",
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
			"failed to mark TiProxy unhealthy before graceful shutdown, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false
	}

	healthy, err = tpClient.IsHealthy(ctx)
	if err != nil {
		logger.Info(
			"failed to re-check TiProxy health after graceful shutdown action, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false
	}
	if healthy {
		logger.Info(
			"TiProxy health is still healthy after graceful shutdown action, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
		)
		return false
	}
	return true
}

func ensureTiProxyHealthOverrideCleared(ctx context.Context, state State, c client.Client, logger logr.Logger) bool {
	tpClient, err := newTiProxyAPIClient(ctx, state, c)
	if err != nil {
		logger.Info("failed to build TiProxy API client before revive, continue retrying", "error", err)
		return false
	}

	if err := tpClient.ClearHealthOverride(ctx); err != nil {
		logger.Info("failed to clear TiProxy health override before revive, continue retrying", "error", err)
		return false
	}

	healthy, err := tpClient.IsHealthy(ctx)
	if err != nil {
		logger.Info("failed to query TiProxy health after revive, continue retrying", "error", err)
		return false
	}
	return healthy
}

func offlineScaleInDrainComplete(tiproxy *v1alpha1.TiProxy, pod *corev1.Pod) (bool, error) {
	annotationSources := []map[string]string{tiproxy.Annotations}
	if pod != nil {
		annotationSources = append(annotationSources, pod.Annotations)
	}
	if !gracefulShutdownStarted(annotationSources...) {
		return false, nil
	}

	return pod == nil || !pod.GetDeletionTimestamp().IsZero(), nil
}

func gracefulShutdownStarted(sources ...map[string]string) bool {
	return !coreutil.GracefulShutdownBeginTimeFromSources(sources...).IsZero()
}

func needsScaleInRevive(tiproxy *v1alpha1.TiProxy, pod *corev1.Pod) bool {
	if tiproxy == nil || coreutil.IsOffline[scope.TiProxy](tiproxy) {
		return false
	}
	if coreutil.HasGracefulDrainState(tiproxy.Annotations) {
		return true
	}
	if pod != nil && coreutil.HasGracefulDrainState(pod.Annotations) {
		return true
	}
	return false
}

func clearGracefulDrainAnnotationsOnCR(tiproxy *v1alpha1.TiProxy) bool {
	if tiproxy.Annotations == nil {
		return false
	}
	changed := false
	if tiproxy.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime] != "" {
		delete(tiproxy.Annotations, v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime)
		changed = true
	}
	return changed
}

func newTiProxyAPIClient(ctx context.Context, state State, c client.Client) (tiproxyapi.TiProxyClient, error) {
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
