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

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"

	v1alpha1br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	t "github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type ReconcileContext struct {
	namespacedName types.NamespacedName
	ctx            context.Context
	cli            client.Client

	tibrgc    *v1alpha1br.TiBRGC
	cluster   *v1alpha1.Cluster
	t2Cronjob *batchv1.CronJob
	t3Cronjob *batchv1.CronJob

	// tibrgcList4SameCluster is a list of tibrgc instances in the same cluster
	// this filed only used to prevent another tibrgc instance scheduled in the same cluster
	tibrgcList4SameCluster *v1alpha1br.TiBRGCList
}

func NewReconcileContext(ctx context.Context,
	namespacedName types.NamespacedName,
	cli client.Client,
) *ReconcileContext {
	return &ReconcileContext{
		namespacedName: namespacedName,
		ctx:            ctx,
		cli:            cli,
	}
}

func (c *ReconcileContext) NamespacedName() types.NamespacedName {
	return c.namespacedName
}

func (c *ReconcileContext) Client() client.Client {
	return c.cli
}

func (c *ReconcileContext) TiBRGC() *v1alpha1br.TiBRGC {
	return c.tibrgc
}

func (c *ReconcileContext) TiBRGCList4SameCluster() *v1alpha1br.TiBRGCList {
	return c.tibrgcList4SameCluster
}

func (c *ReconcileContext) T2Cronjob() *batchv1.CronJob {
	return c.t2Cronjob
}

func (c *ReconcileContext) T3Cronjob() *batchv1.CronJob {
	return c.t3Cronjob
}

func (c *ReconcileContext) Cluster() *v1alpha1.Cluster {
	return c.cluster
}

func (c *ReconcileContext) TLSEnabled() bool {
	return c.Cluster().Spec.TLSCluster != nil && c.Cluster().Spec.TLSCluster.Enabled
}

func (c *ReconcileContext) RefreshTiBRGC() error {
	tibrgc := &v1alpha1br.TiBRGC{}
	if err := c.cli.Get(c.ctx, c.namespacedName, tibrgc); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		c.tibrgc = nil
		return nil // Not found is not an error
	}
	c.tibrgc = tibrgc
	return nil
}

func (c *ReconcileContext) RefreshTiBRGCList4SameCluster() error {
	if c.tibrgc == nil {
		return errors.New("TiBRGC is nil")
	}

	var brgclist v1alpha1br.TiBRGCList
	if err := c.cli.List(c.ctx, &brgclist, &client.ListOptions{
		Namespace: c.tibrgc.Namespace,
		FieldSelector: fields.SelectorFromSet(map[string]string{
			ClusterNameField: c.tibrgc.Spec.Cluster.Name,
		}),
	}); err != nil {
		return err
	}
	c.tibrgcList4SameCluster = &brgclist
	return nil
}

func (c *ReconcileContext) RefreshCluster() error {
	if c.tibrgc == nil {
		return errors.New("TiBRGC is nil")
	}
	clusterName := types.NamespacedName{
		Namespace: c.tibrgc.Namespace,
		Name:      c.tibrgc.Spec.Cluster.Name,
	}
	cluster := &v1alpha1.Cluster{}
	if err := c.cli.Get(c.ctx, clusterName, cluster); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		c.cluster = nil
		return nil // Not found is not an error
	}
	c.cluster = cluster
	return nil
}

func (c *ReconcileContext) RefreshT2CronJob() error {
	if c.tibrgc == nil {
		return errors.New("TiBRGC is nil")
	}
	t2CronjobName := types.NamespacedName{
		Namespace: c.tibrgc.Namespace,
		Name:      T2CronjobName(c.tibrgc),
	}
	t2Cronjob := &batchv1.CronJob{}
	if err := c.cli.Get(c.ctx, t2CronjobName, t2Cronjob); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		c.t2Cronjob = nil
		return nil // Not found is not an error
	}
	c.t2Cronjob = t2Cronjob
	return nil
}

func (c *ReconcileContext) RefreshT3CronJob() error {
	if c.tibrgc == nil {
		return errors.New("TiBRGC is nil")
	}
	t3CronjobName := types.NamespacedName{
		Namespace: c.tibrgc.Namespace,
		Name:      T3CronjobName(c.tibrgc),
	}
	t3Cronjob := &batchv1.CronJob{}
	if err := c.cli.Get(c.ctx, t3CronjobName, t3Cronjob); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		c.t3Cronjob = nil
		return nil // Not found is not an error
	}
	c.t3Cronjob = t3Cronjob
	return nil
}

func TaskContextRefreshTiBRGC(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("ContextRefreshTiBRGC", func(ctx context.Context) t.Result {
		err := rtx.RefreshTiBRGC()
		if err != nil {
			return t.Fail().With("can't get TiBRGC %s: %v", rtx.namespacedName, err)
		}
		if rtx.TiBRGC() == nil {
			return t.Complete().With("tibrgc is refreshed, nil")
		}
		return t.Complete().With("tibrgc is refreshed, not nil")
	})
}

func TaskContextRefreshTiBRGCList4SameCluster(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("ContextRefreshTiBRGCList4SameCluster", func(ctx context.Context) t.Result {
		err := rtx.RefreshTiBRGCList4SameCluster()
		if err != nil {
			return t.Fail().With("can't get TiBRGCList4SameCluster %s: %v", rtx.namespacedName, err)
		}
		return t.Complete().With("TiBRGCList4SameCluster is refreshed")
	})
}

func TaskContextRefreshCluster(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("ContextRefreshCluster", func(ctx context.Context) t.Result {
		err := rtx.RefreshCluster()
		if err != nil {
			return t.Fail().With("can't get Cluster for TiBRGC %s: %v", rtx.namespacedName, err)
		}
		if rtx.Cluster() == nil {
			return t.Complete().With("cluster is refreshed, nil")
		}
		return t.Complete().With("cluster is refreshed, not nil")
	})
}

func TaskContextRefreshT2CronJob(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("ContextRefreshT2CronJob", func(ctx context.Context) t.Result {
		err := rtx.RefreshT2CronJob()
		if err != nil {
			return t.Fail().With("can't get T2CronJob for TiBRGC %s: %v", rtx.namespacedName, err)
		}
		if rtx.T2Cronjob() == nil {
			return t.Complete().With("t2 cronjob is refreshed, nil")
		}
		return t.Complete().With("t2 cronjob is refreshed, not nil")
	})
}

func TaskContextRefreshT3CronJob(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("ContextRefreshT3CronJob", func(ctx context.Context) t.Result {
		err := rtx.RefreshT3CronJob()
		if err != nil {
			return t.Fail().With("can't get T3CronJob for TiBRGC %s: %v", rtx.namespacedName, err)
		}
		if rtx.T3Cronjob() == nil {
			return t.Complete().With("t3 cronjob is refreshed, nil")
		}
		return t.Complete().With("t3 cronjob is refreshed, not nil")
	})
}

func TaskSkipReconcile(rtx *ReconcileContext, reason string) t.Task {
	return t.NameTaskFunc("TaskSkipReconcile", func(ctx context.Context) t.Result {
		return t.Complete().With("skip reconcile, reason: %s", reason)
	})
}
