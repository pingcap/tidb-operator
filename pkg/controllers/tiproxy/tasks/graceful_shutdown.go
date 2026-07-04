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

	var seconds int32
	var ok bool
	if tiproxy != nil {
		raw := tiproxy.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds]
		if raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 32)
			if err != nil {
				return 0, err
			}
			seconds = int32(parsed)
			ok = true
		}
	}
	if !ok || seconds <= 0 {
		return deleteTiProxyPod(ctx, c, pod)
	}

	var startAt time.Time
	if raw := pod.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime]; raw != "" {
		startAt, _ = time.Parse(time.RFC3339Nano, raw)
	}
	if startAt.IsZero() {
		tpClient, err := newTiProxyAPIClient(ctx, state, c)
		if err != nil {
			logger.Info(
				"failed to build TiProxy API client before graceful shutdown, continue retrying",
				"namespace", tiproxy.Namespace,
				"name", tiproxy.Name,
				"error", err,
			)
			return task.DefaultRequeueAfter, nil
		}

		healthy, err := tpClient.IsHealthy(ctx)
		if err != nil {
			logger.Info(
				"failed to query TiProxy health before graceful shutdown, continue retrying",
				"namespace", tiproxy.Namespace,
				"name", tiproxy.Name,
				"error", err,
			)
			return task.DefaultRequeueAfter, nil
		}
		if healthy {
			if err := tpClient.MarkUnhealthy(ctx); err != nil {
				logger.Info(
					"failed to mark TiProxy unhealthy before graceful shutdown, continue retrying",
					"namespace", tiproxy.Namespace,
					"name", tiproxy.Name,
					"error", err,
				)
				return task.DefaultRequeueAfter, nil
			}

			healthy, err = tpClient.IsHealthy(ctx)
			if err != nil {
				logger.Info(
					"failed to re-check TiProxy health after graceful shutdown action, continue retrying",
					"namespace", tiproxy.Namespace,
					"name", tiproxy.Name,
					"error", err,
				)
				return task.DefaultRequeueAfter, nil
			}
			if healthy {
				logger.Info(
					"TiProxy health is still healthy after graceful shutdown action, continue retrying",
					"namespace", tiproxy.Namespace,
					"name", tiproxy.Name,
				)
				return task.DefaultRequeueAfter, nil
			}
		}

		startAt = time.Now()
		newPod := pod.DeepCopy()
		if newPod.Annotations == nil {
			newPod.Annotations = map[string]string{}
		}
		newPod.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime] = startAt.Format(time.RFC3339Nano)
		if err := c.Update(ctx, newPod); err != nil {
			return 0, err
		}
		*pod = *newPod
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

	tpClient, err := newTiProxyAPIClient(ctx, state, c)
	if err != nil {
		logger.Info(
			"failed to build TiProxy API client before checking connections, continue waiting",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
	} else {
		connectionCount, err := tpClient.ConnectionCount(ctx)
		if err != nil {
			logger.Info(
				"failed to query TiProxy connections before graceful delete, continue waiting",
				"namespace", tiproxy.Namespace,
				"name", tiproxy.Name,
				"error", err,
			)
		} else if connectionCount == 0 {
			logger.Info(
				"TiProxy has no active connections, delete pod without waiting for the remaining graceful delay",
				"namespace", tiproxy.Namespace,
				"name", tiproxy.Name,
			)
			if forOfflineScaleIn {
				if err := guardOfflineScaleInBeforePodDelete(ctx, c, tiproxy, forOfflineScaleIn); err != nil {
					return 0, err
				}
				if !coreutil.IsOffline[scope.TiProxy](tiproxy) {
					return task.DefaultRequeueAfter, nil
				}
			}
			return deleteTiProxyPod(ctx, c, pod)
		} else {
			logger.Info(
				"TiProxy still has active connections, continue waiting",
				"namespace", tiproxy.Namespace,
				"name", tiproxy.Name,
				"connectionCount", connectionCount,
			)
		}
	}

	if remaining > task.DefaultRequeueAfter {
		remaining = task.DefaultRequeueAfter
	}

	return remaining, nil
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

func deleteTiProxyPod(ctx context.Context, c client.Client, pod *corev1.Pod) (time.Duration, error) {
	if err := c.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
		return 0, err
	}
	return task.DefaultRequeueAfter, nil
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
