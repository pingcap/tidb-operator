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
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
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

func (rdc *realDMClusterControl) UpdateDMCluster(dc *v1alpha1.DMCluster, newStatus *v1alpha1.DMClusterStatus, oldStatus *v1alpha1.DMClusterStatus) (*v1alpha1.DMCluster, error) {
	ns := dc.GetNamespace()
	dcName := dc.GetName()

	status := dc.Status.DeepCopy()
	var updateDC *v1alpha1.DMCluster

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		updateDC, updateErr = rdc.cli.PingcapV1alpha1().DMClusters(ns).Update(dc)
		if updateErr == nil {
			klog.Infof("DMCluster: [%s/%s] updated successfully", ns, dcName)
			return nil
		}
		klog.V(4).Infof("failed to update DMCluster: [%s/%s], error: %v", ns, dcName, updateErr)

		if updated, err := rdc.dcLister.DMClusters(ns).Get(dcName); err == nil {
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
