// Copyright 2020 PingCAP, Inc.
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
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	tcinformers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// DMClusterControlInterface manages DMClusters
type DMClusterControlInterface interface {
	UpdateDMCluster(*v1alpha1.DMCluster, *v1alpha1.DMClusterStatus, *v1alpha1.DMClusterStatus) (*v1alpha1.DMCluster, error)
}

type realDMClusterControl struct {
	cli      versioned.Interface
	dcLister listers.DMClusterLister
	recorder record.EventRecorder
}

// NewRealDMClusterControl creates a new DMClusterControlInterface
func NewRealDMClusterControl(cli versioned.Interface,
	dcLister listers.DMClusterLister,
	recorder record.EventRecorder) DMClusterControlInterface {
	return &realDMClusterControl{
		cli,
		dcLister,
		recorder,
	}
}

func (c *realDMClusterControl) UpdateDMCluster(dc *v1alpha1.DMCluster, newStatus *v1alpha1.DMClusterStatus, oldStatus *v1alpha1.DMClusterStatus) (*v1alpha1.DMCluster, error) {
	ns := dc.GetNamespace()
	dcName := dc.GetName()

	status := dc.Status.DeepCopy()
	var updateDC *v1alpha1.DMCluster

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		updateDC, updateErr = c.cli.PingcapV1alpha1().DMClusters(ns).Update(dc)
		if updateErr == nil {
			klog.Infof("DMCluster: [%s/%s] updated successfully", ns, dcName)
			return nil
		}
		klog.V(4).Infof("failed to update DMCluster: [%s/%s], error: %v", ns, dcName, updateErr)

		if updated, err := c.dcLister.DMClusters(ns).Get(dcName); err == nil {
			// make a copy so we don't mutate the shared cache
			dc = updated.DeepCopy()
			dc.Status = *status
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated DMCluster %s/%s from lister: %v", ns, dcName, err))
		}

		return updateErr
	})
	if err != nil {
		klog.Errorf("failed to update DMCluster: [%s/%s], error: %v", ns, dcName, err)
	}
	return updateDC, err
}

// FakeDMClusterControl is a fake DMClusterControlInterface
type FakeDMClusterControl struct {
	DcLister               listers.DMClusterLister
	DcIndexer              cache.Indexer
	updateDMClusterTracker RequestTracker
}

// NewFakeDMClusterControl returns a FakeDMClusterControl
func NewFakeDMClusterControl(dcInformer tcinformers.DMClusterInformer) *FakeDMClusterControl {
	return &FakeDMClusterControl{
		dcInformer.Lister(),
		dcInformer.Informer().GetIndexer(),
		RequestTracker{},
	}
}

// SetUpdateDMClusterError sets the error attributes of updateDMClusterTracker
func (c *FakeDMClusterControl) SetUpdateDMClusterError(err error, after int) {
	c.updateDMClusterTracker.SetError(err).SetAfter(after)
}

// UpdateDMCluster updates the DMCluster
func (c *FakeDMClusterControl) UpdateDMCluster(dc *v1alpha1.DMCluster, _ *v1alpha1.DMClusterStatus, _ *v1alpha1.DMClusterStatus) (*v1alpha1.DMCluster, error) {
	defer c.updateDMClusterTracker.Inc()
	if c.updateDMClusterTracker.ErrorReady() {
		defer c.updateDMClusterTracker.Reset()
		return dc, c.updateDMClusterTracker.GetError()
	}

	return dc, c.DcIndexer.Update(dc)
}
