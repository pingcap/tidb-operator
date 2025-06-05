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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

	tibr        *v1alpha1br.TiBR
	cluster     *v1alpha1.Cluster
	configmap   *corev1.ConfigMap
	sts         *appsv1.StatefulSet
	headlessSvc *corev1.Service

	unSyncedReasons []string
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

func (c *ReconcileContext) TiBR() *v1alpha1br.TiBR {
	return c.tibr
}

func (c *ReconcileContext) Cluster() *v1alpha1.Cluster {
	return c.cluster
}

func (c *ReconcileContext) TLSEnabled() bool {
	return c.Cluster().Spec.TLSCluster != nil && c.Cluster().Spec.TLSCluster.Enabled
}

func (c *ReconcileContext) ConfigMap() *corev1.ConfigMap {
	return c.configmap
}

func (c *ReconcileContext) StatefulSet() *appsv1.StatefulSet {
	return c.sts
}

func (c *ReconcileContext) HeadlessSvc() *corev1.Service {
	return c.headlessSvc
}

func (c *ReconcileContext) RefreshTiBR() error {
	tibr := &v1alpha1br.TiBR{}
	if err := c.cli.Get(c.ctx, c.namespacedName, tibr); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		c.tibr = nil
		return nil // Not found is not an error
	}
	c.tibr = tibr
	return nil
}

func (c *ReconcileContext) RefreshCluster() error {
	if c.tibr == nil {
		return errors.New("TiBR is nil")
	}
	clusterName := types.NamespacedName{
		Namespace: c.tibr.Namespace,
		Name:      c.tibr.Spec.Cluster.Name,
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

func (c *ReconcileContext) RefreshConfigMap() error {
	if c.tibr == nil {
		return errors.New("TiBR is nil")
	}
	cmName := types.NamespacedName{
		Namespace: c.tibr.Namespace,
		Name:      ConfigMapName(c.tibr),
	}
	configMap := &corev1.ConfigMap{}
	if err := c.cli.Get(c.ctx, cmName, configMap); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		c.configmap = nil
		return nil // Not found is not an error
	}
	c.configmap = configMap
	return nil
}

func (c *ReconcileContext) RefreshStatefulSet() error {
	if c.tibr == nil {
		return errors.New("TiBR is nil")
	}
	stsName := types.NamespacedName{
		Namespace: c.tibr.Namespace,
		Name:      StatefulSetName(c.tibr),
	}
	sts := &appsv1.StatefulSet{}
	if err := c.cli.Get(c.ctx, stsName, sts); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		c.sts = nil
		return nil // Not found is not an error
	}
	c.sts = sts
	return nil
}

func (c *ReconcileContext) RefreshHeadlessSvc() error {
	if c.tibr == nil {
		return errors.New("TiBR is nil")
	}
	svcName := types.NamespacedName{
		Namespace: c.tibr.Namespace,
		Name:      HeadlessSvcName(c.tibr),
	}
	headlessSvc := &corev1.Service{}
	if err := c.cli.Get(c.ctx, svcName, headlessSvc); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		c.headlessSvc = nil
		return nil // Not found is not an error
	}
	c.headlessSvc = headlessSvc
	return nil
}

func TaskContextRefreshTiBR(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("ContextRefreshTiBR", func(ctx context.Context) t.Result {
		err := rtx.RefreshTiBR()
		if err != nil {
			return t.Fail().With("can't get TiBR %s: %v", rtx.namespacedName, err)
		}
		if rtx.TiBR() == nil {
			return t.Complete().With("tibr is refreshed, nil")
		}
		return t.Complete().With("tibr is refreshed, not nil")
	})
}

func TaskContextRefreshCluster(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("ContextRefreshCluster", func(ctx context.Context) t.Result {
		err := rtx.RefreshCluster()
		if err != nil {
			return t.Fail().With("can't get Cluster for TiBR %s: %v", rtx.namespacedName, err)
		}
		if rtx.Cluster() == nil {
			return t.Complete().With("cluster is refreshed, nil")
		}
		return t.Complete().With("cluster is refreshed, not nil")
	})
}

func TaskContextRefreshConfigMap(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("ContextRefreshConfigMap", func(ctx context.Context) t.Result {
		err := rtx.RefreshConfigMap()
		if err != nil {
			return t.Fail().With("can't get configmap for TiBR %s: %v", rtx.namespacedName, err)
		}
		if rtx.ConfigMap() == nil {
			return t.Complete().With("configmap is refreshed, nil")
		}
		return t.Complete().With("configmap is refreshed, not nil")
	})
}

func TaskContextRefreshStatefulSet(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("ContextRefreshStatefulSet", func(ctx context.Context) t.Result {
		err := rtx.RefreshStatefulSet()
		if err != nil {
			return t.Fail().With("can't get statefulset for TiBR %s: %v", rtx.namespacedName, err)
		}
		if rtx.StatefulSet() == nil {
			return t.Complete().With("statefulset is refreshed, nil")
		}
		return t.Complete().With("statefulset is refreshed, not nil")
	})
}

func TaskContextRefreshHeadlessSvc(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("ContextRefreshHeadlessSvc", func(ctx context.Context) t.Result {
		err := rtx.RefreshHeadlessSvc()
		if err != nil {
			return t.Fail().With("can't get headless svc for TiBR %s: %v", rtx.namespacedName, err)
		}
		if rtx.HeadlessSvc() == nil {
			return t.Complete().With("headless service is refreshed, nil")
		}
		return t.Complete().With("headless service is refreshed, not nil")
	})
}
