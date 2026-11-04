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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/tiproxyapi/v1"
	httputil "github.com/pingcap/tidb-operator/v2/pkg/utils/http"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

// hpaScaleInLimit is the longest an HPA scale-in waits, measured from the
// graceful-shutdown begin time. A zero connection count still deletes the pod
// earlier. When the limit elapses the pod is deleted whether or not the count
// could be read.
const hpaScaleInLimit = 365 * 24 * time.Hour

type reviveAbandonPatch struct {
	Metadata reviveAbandonMetadata `json:"metadata"`
	Spec     *reviveAbandonSpec    `json:"spec,omitempty"`
}

type reviveAbandonMetadata struct {
	ResourceVersion string             `json:"resourceVersion"`
	Annotations     map[string]*string `json:"annotations,omitempty"`
}

type reviveAbandonSpec struct {
	Offline bool `json:"offline"`
}

func isHealthOverrideUnsupported(err error) bool {
	var httpErr httputil.Error
	if !errors.As(err, &httpErr) {
		return false
	}
	// When the health override API exists, DELETE always succeeds whether or not an
	// override is currently set and whether the instance is healthy. 404/405 therefore
	// mean the API is absent on this TiProxy build.
	return httpErr.Status == http.StatusNotFound ||
		httpErr.Status == http.StatusMethodNotAllowed
}

func abandonRevive(ctx context.Context, c client.Client, tiproxy *v1alpha1.TiProxy) error {
	p := reviveAbandonPatch{
		Metadata: reviveAbandonMetadata{
			ResourceVersion: tiproxy.ResourceVersion,
			Annotations: map[string]*string{
				v1alpha1.AnnoKeyTiProxyReviveAbandoned: ptr.To(v1alpha1.AnnoValTrue),
			},
		},
		Spec: &reviveAbandonSpec{
			Offline: true,
		},
	}

	data, err := json.Marshal(&p)
	if err != nil {
		return fmt.Errorf("invalid revive abandon patch: %w", err)
	}

	return c.Patch(ctx, tiproxy, client.RawPatch(types.MergePatchType, data))
}

func drainPodForGracefulShutdown(
	ctx context.Context,
	c client.Client,
	state State,
	pod *corev1.Pod,
	tpClient tiproxyapi.TiProxyClient,
) (time.Duration, error) {
	logger := logr.FromContextOrDiscard(ctx)
	tiproxy := state.Object()

	if !pod.GetDeletionTimestamp().IsZero() {
		return task.DefaultRequeueAfter, nil
	}

	waitForConnections := annotatedHPAScaleIn(tiproxy)
	seconds, ok, err := gracefulShutdownDeleteDelaySeconds(tiproxy)
	if err != nil {
		return 0, err
	}
	if !waitForConnections && (!ok || seconds <= 0) {
		return deleteTiProxyPod(ctx, c, pod)
	}

	// Persist begin-time before MarkUnhealthy so a scale-out after the external
	// side effect (or during MarkUnhealthy retries) can still revive via this signal.
	startAt, hasBegin := gracefulShutdownBeginTime(pod)
	if !hasBegin {
		startAt = time.Now()
		if err := markGracefulShutdownBeginTime(ctx, c, pod, startAt); err != nil {
			return 0, err
		}
	}

	// One limit for an HPA scale-in: a drained count deletes earlier, and a
	// positive count or an unreadable check both wait until this deadline.
	if waitForConnections && time.Until(startAt.Add(hpaScaleInLimit)) <= 0 {
		return deleteHPAScaleInAtLimit(ctx, c, logger, pod, tiproxy)
	}

	// Keep calling MarkUnhealthy even after begin-time is written.
	proceed, overrideUnsupported := ensureTiProxyMarkedUnhealthy(ctx, state, c, logger, tpClient)
	// 404/405 leave the instance healthy, so new connections keep arriving and
	// waiting for the count to reach zero would not finish. Use the delete delay.
	if overrideUnsupported {
		waitForConnections = false
	}
	if waitForConnections {
		if !proceed {
			return hpaScaleInRequeue(startAt), nil
		}
		logger.V(1).Info("HPA scale-in waits for connections instead of the configured graceful shutdown delay",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
		)
		drained, _ := tiProxyConnectionsDrained(ctx, state, c, logger, tpClient)
		if drained {
			return deleteTiProxyPod(ctx, c, pod)
		}
		return hpaScaleInRequeue(startAt), nil
	}
	if !proceed {
		return task.DefaultRequeueAfter, nil
	}

	return deleteAfterGracefulDelay(ctx, c, state, logger, tpClient, tiproxy, pod, startAt, seconds, ok)
}

// deleteAfterGracefulDelay deletes once the configured delay elapses, or earlier
// when the instance has no SQL connections. A missing or non-positive delay
// deletes immediately.
func deleteAfterGracefulDelay(
	ctx context.Context,
	c client.Client,
	state State,
	logger logr.Logger,
	tpClient tiproxyapi.TiProxyClient,
	tiproxy *v1alpha1.TiProxy,
	pod *corev1.Pod,
	startAt time.Time,
	seconds int32,
	ok bool,
) (time.Duration, error) {
	if !ok || seconds <= 0 {
		return deleteTiProxyPod(ctx, c, pod)
	}

	delay := time.Duration(seconds) * time.Second
	remaining := time.Until(hpaDeleteDelayStart(tiproxy, startAt, delay).Add(delay))
	if remaining <= 0 {
		return deleteTiProxyPod(ctx, c, pod)
	}

	if drained, err := tiProxyConnectionsDrained(ctx, state, c, logger, tpClient); err == nil && drained {
		return deleteTiProxyPod(ctx, c, pod)
	}

	if remaining > task.DefaultRequeueAfter {
		remaining = task.DefaultRequeueAfter
	}

	return remaining, nil
}

func annotatedHPAScaleIn(tiproxy *v1alpha1.TiProxy) bool {
	// An explicit delete uses the configured delay. The one-year HPA limit must
	// not replace that delay after the object is deleted.
	if tiproxy == nil || !tiproxy.GetDeletionTimestamp().IsZero() {
		return false
	}
	return tiproxy.Annotations[v1alpha1.AnnoKeyTiProxyHPAScaleIn] == v1alpha1.AnnoValTrue
}

func deleteHPAScaleInAtLimit(
	ctx context.Context,
	c client.Client,
	logger logr.Logger,
	pod *corev1.Pod,
	tiproxy *v1alpha1.TiProxy,
) (time.Duration, error) {
	logger.Info("HPA scale-in reached the 1y limit, delete pod",
		"namespace", tiproxy.Namespace,
		"name", tiproxy.Name,
	)
	return deleteTiProxyPod(ctx, c, pod)
}

func hpaScaleInRequeue(startAt time.Time) time.Duration {
	remaining := time.Until(startAt.Add(hpaScaleInLimit))
	if remaining <= 0 || remaining > task.DefaultRequeueAfter {
		return task.DefaultRequeueAfter
	}
	return remaining
}

// hpaDeleteDelayStart is the start of the configured delete delay.
// HPA scale-in writes begin-time when the connection wait starts, and that wait
// can run longer than the delay. Deleting the object then must not treat the
// old begin-time as the delay clock: remaining would already be <= 0 and the
// pod would be deleted at once. A delay that is still running keeps its clock.
func hpaDeleteDelayStart(tiproxy *v1alpha1.TiProxy, begin time.Time, delay time.Duration) time.Time {
	if tiproxy == nil || tiproxy.GetDeletionTimestamp().IsZero() ||
		tiproxy.Annotations[v1alpha1.AnnoKeyTiProxyHPAScaleIn] != v1alpha1.AnnoValTrue {
		return begin
	}
	deletedAt := tiproxy.GetDeletionTimestamp().Time
	// delay is positive here. A begin time at or after deletion is still inside
	// the window, so the comparison does not need a separate begin check.
	if begin.Add(delay).After(deletedAt) {
		return begin
	}
	return deletedAt
}

func TaskClearHPAScaleInAnnotation(state State, c client.Client) task.Task {
	return task.NameTaskFunc("ClearHPAScaleInAnnotation", func(ctx context.Context) task.Result {
		tiproxy := state.Object()
		if tiproxy.Annotations[v1alpha1.AnnoKeyTiProxyHPAScaleIn] != v1alpha1.AnnoValTrue {
			return task.Complete().With("tiproxy is not marked as an HPA scale-in")
		}
		// This task runs only after CondObjectIsDeleting breaks, so the object is not deleting.
		if coreutil.IsOffline[scope.TiProxy](tiproxy) {
			return task.Complete().With("tiproxy HPA scale-in annotation stays until it is back online")
		}

		if err := clearHPAScaleInAnnotation(ctx, c, tiproxy); err != nil {
			if apierrors.IsConflict(err) {
				return task.Retry(task.DefaultRequeueAfter).With("tiproxy changed before clearing HPA scale-in annotation, retry")
			}
			return task.Fail().With("cannot clear tiproxy HPA scale-in annotation: %v", err)
		}
		// Patch decodes the response onto this object. If the removed key was the
		// only annotation, the response omits metadata.annotations and the local map stays.
		delete(tiproxy.Annotations, v1alpha1.AnnoKeyTiProxyHPAScaleIn)
		return task.Complete().With("cleared tiproxy HPA scale-in annotation")
	})
}

func clearHPAScaleInAnnotation(ctx context.Context, c client.Client, tiproxy *v1alpha1.TiProxy) error {
	p := map[string]any{
		"metadata": map[string]any{
			"resourceVersion": tiproxy.ResourceVersion,
			"annotations": map[string]any{
				v1alpha1.AnnoKeyTiProxyHPAScaleIn: nil,
			},
		},
	}
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("invalid HPA scale-in annotation patch: %w", err)
	}
	return c.Patch(ctx, tiproxy, client.RawPatch(types.MergePatchType, data))
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

// tiProxyConnectionsDrained reports whether the instance has no SQL connections.
// A non-nil error means the count could not be read; callers decide how long to wait.
func tiProxyConnectionsDrained(
	ctx context.Context,
	state State,
	c client.Client,
	logger logr.Logger,
	tpClient tiproxyapi.TiProxyClient,
) (bool, error) {
	tiproxy := state.Object()

	apiClient, err := resolveTiProxyAPIClient(ctx, state, c, tpClient)
	if err != nil {
		logger.Info(
			"failed to build TiProxy API client before checking connections",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false, err
	}

	connectionCount, err := apiClient.ConnectionCount(ctx)
	if err != nil {
		logger.Info(
			"failed to query TiProxy connections before graceful delete",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false, err
	}
	if connectionCount > 0 {
		logger.Info(
			"TiProxy still has active connections, continue waiting",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"connectionCount", connectionCount,
		)
		return false, nil
	}

	logger.Info(
		"TiProxy has no active connections, delete pod without waiting for the remaining graceful delay",
		"namespace", tiproxy.Namespace,
		"name", tiproxy.Name,
	)
	return true, nil
}

// ensureTiProxyMarkedUnhealthy reports whether shutdown may proceed.
// overrideUnsupported is set when GET or PUT /api/debug/health returns 404 or 405.
// The instance is still healthy then, so a caller waiting only for connections
// must use the delete delay instead.
func ensureTiProxyMarkedUnhealthy(
	ctx context.Context,
	state State,
	c client.Client,
	logger logr.Logger,
	tpClient tiproxyapi.TiProxyClient,
) (proceed, overrideUnsupported bool) {
	tiproxy := state.Object()

	apiClient, err := resolveTiProxyAPIClient(ctx, state, c, tpClient)
	if err != nil {
		logger.Info(
			"failed to build TiProxy API client before graceful delete, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false, false
	}

	healthy, err := apiClient.IsHealthy(ctx)
	if err != nil {
		// GET 404/405 is the same missing API as PUT 404/405. Treating it as
		// "already unhealthy" would wait for connections on an instance that
		// is still accepting them.
		if isHealthOverrideUnsupported(err) {
			logger.Info(
				"health override API unsupported, skip MarkUnhealthy and continue delete delay",
				"namespace", tiproxy.Namespace,
				"name", tiproxy.Name,
				"error", err,
			)
			return true, true
		}
		logger.Info(
			"failed to query TiProxy health before graceful delete retry, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false, false
	}
	if !healthy {
		return true, false
	}

	if err := apiClient.MarkUnhealthy(ctx); err != nil {
		// When the health override API exists, PUT succeeds. 404/405 mean the API
		// is absent. Proceed so the delete delay can bound the wait; the caller
		// must not use the unbounded connection wait while the instance stays healthy.
		if isHealthOverrideUnsupported(err) {
			logger.Info(
				"health override API unsupported, skip MarkUnhealthy and continue delete delay",
				"namespace", tiproxy.Namespace,
				"name", tiproxy.Name,
				"error", err,
			)
			return true, true
		}
		logger.Info(
			"failed to mark TiProxy unhealthy before graceful delete, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false, false
	}

	healthy, err = apiClient.IsHealthy(ctx)
	if err != nil {
		logger.Info(
			"failed to re-check TiProxy health after graceful delete action, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
			"error", err,
		)
		return false, false
	}
	if healthy {
		logger.Info(
			"TiProxy health is still healthy after graceful delete action, continue retrying",
			"namespace", tiproxy.Namespace,
			"name", tiproxy.Name,
		)
		return false, false
	}
	return true, false
}

func resolveTiProxyAPIClient(
	ctx context.Context,
	state State,
	c client.Client,
	tpClient tiproxyapi.TiProxyClient,
) (tiproxyapi.TiProxyClient, error) {
	if tpClient != nil {
		return tpClient, nil
	}
	return newTiProxyAPIClient(ctx, state, c)
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
