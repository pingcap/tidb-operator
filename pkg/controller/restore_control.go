// Copyright 2022 PingCAP, Inc.
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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	tcinformers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

// RestoreControlInterface manages Restores
type RestoreControlInterface interface {
	UpdateRestore(*v1alpha1.Restore) (*v1alpha1.Restore, error)
}

type realRestoreControl struct {
	cli      versioned.Interface
	rsLister listers.RestoreLister
	recorder record.EventRecorder
}

// NewRealRestoreControl creates a new RestoreControlInterface
func NewRealRestoreControl(
	cli versioned.Interface,
	rsLister listers.RestoreLister,
	recorder record.EventRecorder,
) RestoreControlInterface {
	return &realRestoreControl{
		cli:      cli,
		rsLister: rsLister,
		recorder: recorder,
	}
}

func (c *realRestoreControl) UpdateRestore(rs *v1alpha1.Restore) (*v1alpha1.Restore, error) {
	ns := rs.GetNamespace()
	rsName := rs.GetName()

	var updateRs *v1alpha1.Restore

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		updateRs, updateErr = c.cli.PingcapV1alpha1().Restores(ns).Update(context.TODO(), rs, metav1.UpdateOptions{})
		if updateErr == nil {
			klog.Infof("Restore: [%s/%s] updated successfully", ns, rsName)
			return nil
		}
		klog.V(4).Infof("failed to update Restore: [%s/%s], error: %v", ns, rsName, updateErr)

		if updated, err := c.rsLister.Restores(ns).Get(rsName); err == nil {
			rs.ResourceVersion = updated.ResourceVersion
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated Restore %s/%s from lister: %v", ns, rsName, err))
		}

		return updateErr
	})
	if err != nil {
		klog.Errorf("failed to update Restore: [%s/%s], error: %v", ns, rsName, err)
	}
	return updateRs, err
}

// FakeRestoreControl is a fake RestoreControlInterface
type FakeRestoreControl struct {
	RestoreLister        listers.RestoreLister
	RestoreIndexer       cache.Indexer
	updateRestoreTracker RequestTracker
	createRestoreTracker RequestTracker
}

// NewFakeRestoreControl returns a FakeRestoreControl
func NewFakeRestoreControl(rsInformer tcinformers.RestoreInformer) *FakeRestoreControl {
	return &FakeRestoreControl{
		rsInformer.Lister(),
		rsInformer.Informer().GetIndexer(),
		RequestTracker{},
		RequestTracker{},
	}
}

// UpdateRestore updates the Restore
func (c *FakeRestoreControl) UpdateRestore(rs *v1alpha1.Restore) (*v1alpha1.Restore, error) {
	defer c.updateRestoreTracker.Inc()
	if c.updateRestoreTracker.ErrorReady() {
		defer c.updateRestoreTracker.Reset()
		return rs, c.updateRestoreTracker.GetError()
	}

	return rs, c.RestoreIndexer.Update(rs)
}
