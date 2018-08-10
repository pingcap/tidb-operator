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

package tidbcluster

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	v1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap.com/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
)

// StatusUpdaterInterface is an interface used to update the TidbClusterStatus associated with a TidbCluster.
// For any use other than testing, clients should create an instance using NewRealTidbClusterStatusUpdater.
type StatusUpdaterInterface interface {
	// UpdateTidbClusterStatus sets the tidbCluster's Status to status. Implementations are required to retry on conflicts,
	// but fail on other errors. If the returned error is nil tidbCluster's Status has been successfully set to status.
	UpdateTidbClusterStatus(*v1alpha1.TidbCluster, *v1alpha1.TidbClusterStatus) error
}

// NewRealTidbClusterStatusUpdater returns a StatusUpdaterInterface that updates the Status of a TidbCluster,
// using the supplied client and setLister.
func NewRealTidbClusterStatusUpdater(
	cli versioned.Interface,
	tcLister v1listers.TidbClusterLister) StatusUpdaterInterface {
	return &realTidbClusterStatusUpdater{cli, tcLister}
}

type realTidbClusterStatusUpdater struct {
	cli      versioned.Interface
	tcLister v1listers.TidbClusterLister
}

func (tcs *realTidbClusterStatusUpdater) UpdateTidbClusterStatus(
	tc *v1alpha1.TidbCluster,
	status *v1alpha1.TidbClusterStatus) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	// don't wait due to limited number of clients, but backoff after the default number of steps
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		tc.Status = *status
		_, updateErr := tcs.cli.PingcapV1alpha1().TidbClusters(ns).Update(tc)
		if updateErr == nil {
			return nil
		}
		if updated, err := tcs.tcLister.TidbClusters(ns).Get(tcName); err == nil {
			// make a copy so we don't mutate the shared cache
			tc = updated.DeepCopy()
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated TidbCluster %s/%s from lister: %v", ns, tcName, err))
		}

		return updateErr
	})
}

var _ StatusUpdaterInterface = &realTidbClusterStatusUpdater{}
