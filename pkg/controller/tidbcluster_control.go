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
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	tcinformers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// TidbClusterControlInterface manages TidbClusters
type TidbClusterControlInterface interface {
	UpdateTidbCluster(*v1alpha1.TidbCluster, *v1alpha1.TidbClusterStatus, *v1alpha1.TidbClusterStatus) (*v1alpha1.TidbCluster, error)
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

func (rtc *realTidbClusterControl) UpdateTidbCluster(tc *v1alpha1.TidbCluster, newStatus *v1alpha1.TidbClusterStatus, oldStatus *v1alpha1.TidbClusterStatus) (*v1alpha1.TidbCluster, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	status := tc.Status.DeepCopy()
	var updateTC *v1alpha1.TidbCluster

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		updateTC, updateErr = rtc.cli.PingcapV1alpha1().TidbClusters(ns).Update(tc)
		if updateErr == nil {
			klog.Infof("TidbCluster: [%s/%s] updated successfully", ns, tcName)
			return nil
		}
		klog.Errorf("failed to update TidbCluster: [%s/%s], error: %v", ns, tcName, updateErr)

		if updated, err := rtc.tcLister.TidbClusters(ns).Get(tcName); err == nil {
			// make a copy so we don't mutate the shared cache
			tc = updated.DeepCopy()
			tc.Status = *status
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated TidbCluster %s/%s from lister: %v", ns, tcName, err))
		}

		return updateErr
	})
	return updateTC, err
}

func (rtc *realTidbClusterControl) recordTidbClusterEvent(verb string, tc *v1alpha1.TidbCluster, err error) {
	tcName := tc.GetName()
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s TidbCluster %s successful",
			strings.ToLower(verb), tcName)
		rtc.recorder.Event(tc, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s TidbCluster %s failed error: %s",
			strings.ToLower(verb), tcName, err)
		rtc.recorder.Event(tc, corev1.EventTypeWarning, reason, msg)
	}
}

func deepEqualExceptHeartbeatTime(newStatus *v1alpha1.TidbClusterStatus, oldStatus *v1alpha1.TidbClusterStatus) bool {
	sweepHeartbeatTime(newStatus.TiKV.Stores)
	sweepHeartbeatTime(newStatus.TiKV.TombstoneStores)
	sweepHeartbeatTime(oldStatus.TiKV.Stores)
	sweepHeartbeatTime(oldStatus.TiKV.TombstoneStores)

	return apiequality.Semantic.DeepEqual(newStatus, oldStatus)
}

func sweepHeartbeatTime(stores map[string]v1alpha1.TiKVStore) {
	for id, store := range stores {
		store.LastHeartbeatTime = metav1.Time{}
		stores[id] = store
	}
}

// FakeTidbClusterControl is a fake TidbClusterControlInterface
type FakeTidbClusterControl struct {
	TcLister                 listers.TidbClusterLister
	TcIndexer                cache.Indexer
	updateTidbClusterTracker RequestTracker
}

// NewFakeTidbClusterControl returns a FakeTidbClusterControl
func NewFakeTidbClusterControl(tcInformer tcinformers.TidbClusterInformer) *FakeTidbClusterControl {
	return &FakeTidbClusterControl{
		tcInformer.Lister(),
		tcInformer.Informer().GetIndexer(),
		RequestTracker{},
	}
}

// SetUpdateTidbClusterError sets the error attributes of updateTidbClusterTracker
func (ssc *FakeTidbClusterControl) SetUpdateTidbClusterError(err error, after int) {
	ssc.updateTidbClusterTracker.SetError(err).SetAfter(after)
}

// UpdateTidbCluster updates the TidbCluster
func (ssc *FakeTidbClusterControl) UpdateTidbCluster(tc *v1alpha1.TidbCluster, _ *v1alpha1.TidbClusterStatus, _ *v1alpha1.TidbClusterStatus) (*v1alpha1.TidbCluster, error) {
	defer ssc.updateTidbClusterTracker.Inc()
	if ssc.updateTidbClusterTracker.ErrorReady() {
		defer ssc.updateTidbClusterTracker.Reset()
		return tc, ssc.updateTidbClusterTracker.GetError()
	}

	return tc, ssc.TcIndexer.Update(tc)
}
