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
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/tiproxyapi/v1"
	httputil "github.com/pingcap/tidb-operator/v2/pkg/utils/http"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

var FinalizerSubresourceLister = common.NewSubresourceLister(
	common.NewSubresource[corev1.ConfigMapList](),
	common.NewSubresource[corev1.PersistentVolumeClaimList](),
)

func TaskDrainPodForDelete(state State, c client.Client, restConfig *rest.Config) task.Task {
	return task.NameTaskFunc("DrainPodForDelete", func(ctx context.Context) task.Result {
		pod := state.Pod()
		if pod == nil {
			return task.Complete().With("pod doesn't exist")
		}

		retryAfter, err := drainOrDeletePod(ctx, c, state, pod, restConfig)
		if err != nil {
			return task.Fail().With("cannot delete pod of tiproxy: %v", err)
		}
		if retryAfter > 0 {
			return task.Retry(retryAfter).With("wait for tiproxy pod to be deleted")
		}
		return task.Complete().With("pod is deleted")
	})
}

func drainOrDeletePod(ctx context.Context, c client.Client, state State, pod *corev1.Pod, restConfig *rest.Config) (time.Duration, error) {
	logger := logr.FromContextOrDiscard(ctx)
	tiproxy := state.Object()

	if !pod.GetDeletionTimestamp().IsZero() {
		return task.DefaultRequeueAfter, nil
	}

	seconds := tiproxy.Spec.GracefulShutdownDeleteDelaySeconds
	if seconds == nil || *seconds <= 0 {
		return deleteTiProxyPod(ctx, c, pod)
	}

	startAt, ok := gracefulShutdownBeginTime(pod)
	if !ok {
		if !ensureTiProxyMarkedUnhealthy(ctx, state, c, logger, restConfig) {
			return task.DefaultRequeueAfter, nil
		}
		startAt = time.Now()
		if err := markGracefulShutdownBeginTime(ctx, c, pod, startAt); err != nil {
			return 0, err
		}
	}

	remaining := time.Until(startAt.Add(time.Duration(*seconds) * time.Second))
	if remaining > task.DefaultRequeueAfter {
		remaining = task.DefaultRequeueAfter
	}
	if remaining > 0 {
		return remaining, nil
	}

	return deleteTiProxyPod(ctx, c, pod)
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

func ensureTiProxyMarkedUnhealthy(ctx context.Context, state State, c client.Client, logger logr.Logger, restConfig *rest.Config) bool {
	tiproxy := state.Object()

	tpClient, err := newTiProxyDeleteClient(ctx, state, c)
	if err != nil {
		logger.Info("failed to build TiProxy API client before graceful delete, continue retrying", "namespace", tiproxy.Namespace, "name", tiproxy.Name, "error", err)
		return false
	}

	healthy, err := tpClient.IsHealthy(ctx)
	if err != nil {
		logger.Info("failed to query TiProxy health before graceful delete retry, continue retrying", "namespace", tiproxy.Namespace, "name", tiproxy.Name, "error", err)
		return false
	}
	if !healthy {
		return true
	}

	if err := tpClient.MarkUnhealthy(ctx); err != nil {
		if httputil.IsNotFound(err) {
			gracefulWait, cfgErr := tpClient.GetGracefulWaitBeforeShutdown(ctx)
			if cfgErr != nil {
				logger.Info("failed to get TiProxy config for SIGTERM fallback after unhealthy API is not found, continue retrying", "namespace", tiproxy.Namespace, "name", tiproxy.Name, "error", cfgErr)
				return false
			}
			if tiproxy.Spec.GracefulShutdownDeleteDelaySeconds == nil || gracefulWait < int(*tiproxy.Spec.GracefulShutdownDeleteDelaySeconds) {
				logger.Info("TiProxy unhealthy API is not found and graceful-wait-before-shutdown is shorter than delete delay, continue retrying", "namespace", tiproxy.Namespace, "name", tiproxy.Name, "gracefulWaitBeforeShutdown", gracefulWait, "gracefulShutdownDeleteDelaySeconds", tiproxy.Spec.GracefulShutdownDeleteDelaySeconds)
				return false
			}
			if restConfig == nil {
				logger.Info("failed to fallback to SIGTERM after unhealthy API is not found, rest config is not available, continue retrying", "namespace", tiproxy.Namespace, "name", tiproxy.Name)
				return false
			}
			if err := sendSIGTERMToTiProxyPod(ctx, restConfig, state.Pod()); err != nil {
				logger.Info("failed to fallback to SIGTERM after unhealthy API is not found, continue retrying", "namespace", tiproxy.Namespace, "name", tiproxy.Name, "error", err)
				return false
			}
			logger.Info("TiProxy unhealthy API is not found, fallback to SIGTERM using graceful-wait-before-shutdown", "namespace", tiproxy.Namespace, "name", tiproxy.Name, "gracefulWaitBeforeShutdown", gracefulWait)
		} else {
			logger.Info("failed to mark TiProxy unhealthy before graceful delete, continue retrying", "namespace", tiproxy.Namespace, "name", tiproxy.Name, "error", err)
			return false
		}
	}

	healthy, err = tpClient.IsHealthy(ctx)
	if err != nil {
		logger.Info("failed to re-check TiProxy health after graceful delete action, continue retrying", "namespace", tiproxy.Namespace, "name", tiproxy.Name, "error", err)
		return false
	}
	if healthy {
		logger.Info("TiProxy health is still healthy after graceful delete action, continue retrying", "namespace", tiproxy.Namespace, "name", tiproxy.Name)
		return false
	}
	return true
}

func sendSIGTERMToTiProxyPod(ctx context.Context, restConfig *rest.Config, pod *corev1.Pod) error {
	cfg := rest.CopyConfig(restConfig)
	cfg.APIPath = "/api"
	cfg.GroupVersion = &corev1.SchemeGroupVersion
	cfg.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	restClient, err := rest.RESTClientFor(cfg)
	if err != nil {
		return err
	}

	req := restClient.Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: v1alpha1.ContainerNameTiProxy,
			Command:   []string{"/bin/sh", "-c", "kill -TERM 1"},
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return err
	}

	var stderr bytes.Buffer
	if err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{Stderr: &stderr}); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, stderr.String())
		}
		return err
	}

	return nil
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
