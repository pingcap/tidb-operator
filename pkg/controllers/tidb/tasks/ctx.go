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
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/pkg/tidbapi/v1"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
	tlsutil "github.com/pingcap/tidb-operator/pkg/utils/tls"
)

const (
	pdRequestTimeout   = 10 * time.Second
	tidbRequestTimeout = 10 * time.Second
)

type ReconcileContext struct {
	context.Context

	Key types.NamespacedName

	TiDBClient tidbapi.TiDBClient
	PDClient   pdapi.PDClient

	Healthy   bool
	Suspended bool

	Cluster *v1alpha1.Cluster
	TiDB    *v1alpha1.TiDB
	Pod     *corev1.Pod

	GracefulWaitTimeInSeconds int64

	// ConfigHash stores the hash of **user-specified** config (i.e.`.Spec.Config`),
	// which will be used to determine whether the config has changed.
	// This ensures that our config overlay logic will not restart the tidb cluster unexpectedly.
	ConfigHash string

	// Pod cannot be updated when call DELETE API, so we have to set this field to indicate
	// the underlay pod has been deleting
	PodIsTerminating bool
}

func (ctx *ReconcileContext) Self() *ReconcileContext {
	return ctx
}

func TaskContextTiDB(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("ContextTiDB", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		var tidb v1alpha1.TiDB
		if err := c.Get(ctx, rtx.Key, &tidb); err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("can't get tidb instance: %w", err)
			}
			return task.Complete().With("tidb instance has been deleted")
		}
		rtx.TiDB = &tidb
		return task.Complete().With("tidb is set")
	})
}

func TaskContextCluster(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("ContextCluster", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		var cluster v1alpha1.Cluster
		if err := c.Get(ctx, client.ObjectKey{
			Name:      rtx.TiDB.Spec.Cluster.Name,
			Namespace: rtx.TiDB.Namespace,
		}, &cluster); err != nil {
			return task.Fail().With("cannot find cluster %s: %w", rtx.TiDB.Spec.Cluster.Name, err)
		}
		rtx.Cluster = &cluster
		return task.Complete().With("cluster is set")
	})
}

func TaskContextPod(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("ContextPod", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		var pod corev1.Pod
		if err := c.Get(ctx, client.ObjectKey{
			Name:      rtx.TiDB.PodName(),
			Namespace: rtx.TiDB.Namespace,
		}, &pod); err != nil {
			if errors.IsNotFound(err) {
				return task.Complete().With("pod is not created")
			}
			return task.Fail().With("failed to get pod of tidb: %w", err)
		}

		rtx.Pod = &pod
		if !rtx.Pod.GetDeletionTimestamp().IsZero() {
			rtx.PodIsTerminating = true
		}
		return task.Complete().With("pod is set")
	})
}

func TaskContextInfoFromPDAndTiDB(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("ContextInfoFromPDAndTiDB", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		if rtx.Cluster.Status.PD == "" {
			return task.Fail().With("pd url is not initialized")
		}
		var (
			scheme    = "http"
			tlsConfig *tls.Config
		)
		if rtx.Cluster.IsTLSClusterEnabled() {
			scheme = "https"
			var err error
			tlsConfig, err = tlsutil.GetTLSConfigFromSecret(ctx, c,
				rtx.Cluster.Namespace, v1alpha1.TLSClusterClientSecretName(rtx.Cluster.Name))
			if err != nil {
				return task.Fail().With("cannot get tls config from secret: %w", err)
			}
		}
		rtx.TiDBClient = tidbapi.NewTiDBClient(TiDBServiceURL(rtx.TiDB, scheme), tidbRequestTimeout, tlsConfig)
		health, err := rtx.TiDBClient.GetHealth(ctx)
		if err != nil {
			return task.Complete().With(
				fmt.Sprintf("context without health info is completed, tidb can't be reached: %v", err))
		}
		rtx.Healthy = health
		rtx.PDClient = pdapi.NewPDClient(rtx.Cluster.Status.PD, pdRequestTimeout, tlsConfig)

		return task.Complete().With("get info from tidb")
	})
}

func CondTiDBHasBeenDeleted() task.Condition[ReconcileContext] {
	return task.CondFunc[ReconcileContext](func(ctx task.Context[ReconcileContext]) bool {
		return ctx.Self().TiDB == nil
	})
}

func CondTiDBIsDeleting() task.Condition[ReconcileContext] {
	return task.CondFunc[ReconcileContext](func(ctx task.Context[ReconcileContext]) bool {
		return !ctx.Self().TiDB.GetDeletionTimestamp().IsZero()
	})
}

func CondClusterIsPaused() task.Condition[ReconcileContext] {
	return task.CondFunc[ReconcileContext](func(ctx task.Context[ReconcileContext]) bool {
		return ctx.Self().Cluster.ShouldPauseReconcile()
	})
}

func CondClusterIsSuspending() task.Condition[ReconcileContext] {
	return task.CondFunc[ReconcileContext](func(ctx task.Context[ReconcileContext]) bool {
		return ctx.Self().Cluster.ShouldSuspendCompute()
	})
}

func CondPDIsNotInitialized() task.Condition[ReconcileContext] {
	return task.CondFunc[ReconcileContext](func(ctx task.Context[ReconcileContext]) bool {
		return ctx.Self().Cluster.Status.PD == ""
	})
}
