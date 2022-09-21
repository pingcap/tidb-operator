// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	tcinformers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
)

// TidbClusterControlInterface manages TidbClusters
type TidbClusterControlInterface interface {
	UpdateTidbCluster(*v1alpha1.TidbCluster, *v1alpha1.TidbClusterStatus, *v1alpha1.TidbClusterStatus) (*v1alpha1.TidbCluster, error)
	Create(*v1alpha1.TidbCluster) error
	Patch(tc *v1alpha1.TidbCluster, data []byte, subresources ...string) (result *v1alpha1.TidbCluster, err error)
}

type realTidbClusterControl struct {
	cli      versioned.Interface
	tcLister listers.TidbClusterLister
	recorder record.EventRecorder
}

// NewRealTidbClusterControl creates a new TidbClusterControlInterface
func NewRealTidbClusterControl(cli versioned.Interface,
	tcLister listers.TidbClusterLister,
	recorder record.EventRecorder) TidbClusterControlInterface {
	return &realTidbClusterControl{
		cli,
		tcLister,
		recorder,
	}
}

func (c *realTidbClusterControl) UpdateTidbCluster(tc *v1alpha1.TidbCluster, newStatus *v1alpha1.TidbClusterStatus, oldStatus *v1alpha1.TidbClusterStatus) (*v1alpha1.TidbCluster, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	status := tc.Status.DeepCopy()
	var updateTC *v1alpha1.TidbCluster

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		updateTC, updateErr = c.cli.PingcapV1alpha1().TidbClusters(ns).Update(context.TODO(), tc, metav1.UpdateOptions{})
		if updateErr == nil {
			klog.Infof("TidbCluster: [%s/%s] updated successfully", ns, tcName)
			return nil
		}
		klog.V(4).Infof("failed to update TidbCluster: [%s/%s], error: %v", ns, tcName, updateErr)

		if updated, err := c.tcLister.TidbClusters(ns).Get(tcName); err == nil {
			// make a copy so we don't mutate the shared cache
			tc = updated.DeepCopy()
			// TiKV.EvictLeader is controlled by pod leader evictor in pkg/controller/tidbcluster/pod_control.go
			// So don't overwrite it
			status.TiKV.EvictLeader = tc.Status.TiKV.EvictLeader
			tc.Status = *status
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated TidbCluster %s/%s from lister: %v", ns, tcName, err))
		}

		return updateErr
	})
	if err != nil {
		klog.Errorf("failed to update TidbCluster: [%s/%s], error: %v", ns, tcName, err)
	}
	return updateTC, err
}

func (c *realTidbClusterControl) Create(*v1alpha1.TidbCluster) error {
	return nil
}

func (c *realTidbClusterControl) Patch(tc *v1alpha1.TidbCluster, data []byte, subresources ...string) (result *v1alpha1.TidbCluster, err error) {
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var patchErr error
		_, patchErr = c.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Patch(context.TODO(), tc.Name, types.MergePatchType, data, metav1.PatchOptions{})
		return patchErr
	})
	if err != nil {
		klog.Errorf("failed to patch TidbCluster: [%s/%s], error: %v", tc.Namespace, tc.Name, err)
	}
	return tc, err
}

// FakeTidbClusterControl is a fake TidbClusterControlInterface
type FakeTidbClusterControl struct {
	TcLister                 listers.TidbClusterLister
	TcIndexer                cache.Indexer
	updateTidbClusterTracker RequestTracker
	createTidbClusterTracker RequestTracker
}

// NewFakeTidbClusterControl returns a FakeTidbClusterControl
func NewFakeTidbClusterControl(tcInformer tcinformers.TidbClusterInformer) *FakeTidbClusterControl {
	return &FakeTidbClusterControl{
		tcInformer.Lister(),
		tcInformer.Informer().GetIndexer(),
		RequestTracker{},
		RequestTracker{},
	}
}

// SetUpdateTidbClusterError sets the error attributes of updateTidbClusterTracker
func (c *FakeTidbClusterControl) SetUpdateTidbClusterError(err error, after int) {
	c.updateTidbClusterTracker.SetError(err).SetAfter(after)
}

// UpdateTidbCluster updates the TidbCluster
func (c *FakeTidbClusterControl) UpdateTidbCluster(tc *v1alpha1.TidbCluster, _ *v1alpha1.TidbClusterStatus, _ *v1alpha1.TidbClusterStatus) (*v1alpha1.TidbCluster, error) {
	defer c.updateTidbClusterTracker.Inc()
	if c.updateTidbClusterTracker.ErrorReady() {
		defer c.updateTidbClusterTracker.Reset()
		return tc, c.updateTidbClusterTracker.GetError()
	}

	return tc, c.TcIndexer.Update(tc)
}

func (c *FakeTidbClusterControl) Create(tc *v1alpha1.TidbCluster) error {
	defer func() {
		c.createTidbClusterTracker.Inc()
	}()

	if c.createTidbClusterTracker.ErrorReady() {
		defer c.createTidbClusterTracker.Reset()
		return c.createTidbClusterTracker.GetError()
	}
	return c.TcIndexer.Add(tc)
}

func (c *FakeTidbClusterControl) Patch(tc *v1alpha1.TidbCluster, data []byte, subresources ...string) (result *v1alpha1.TidbCluster, err error) {
	return nil, nil
}
