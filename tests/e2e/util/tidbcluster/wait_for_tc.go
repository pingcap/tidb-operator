// Copyright 2021 PingCAP, Inc.
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
	"context"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/util/tidbcluster"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	testutils "k8s.io/kubernetes/test/utils"
)

var (
	pollInterval = time.Second * 10
)

type TCCondition func(tc *v1alpha1.TidbCluster) (bool, error)

// WaitForTCConditionReady waits for a TidbClusterCondition to be ready for at least minReadyDuration duration.
func WaitForTCConditionReady(c versioned.Interface, ns, name string, timeout time.Duration, minReadyDuration time.Duration) error {
	condition := func(tc *v1alpha1.TidbCluster) (bool, error) {
		return IsTidbClusterAvailable(tc, minReadyDuration, time.Now()), nil
	}

	return WaitForTCCondition(c, ns, name, timeout, condition)
}

// WaitForTCCondition waits for a TidbCluster to be matched to the given condition.
//
// If tc is not found, break the poll and return error
func WaitForTCCondition(c versioned.Interface, tcNS string, tcName string, timeout time.Duration, condition TCCondition) error {
	return wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		tc, err := c.PingcapV1alpha1().TidbClusters(tcNS).Get(context.TODO(), tcName, metav1.GetOptions{})
		if err != nil {
			if testutils.IsRetryableAPIError(err) {
				return false, nil
			}
			return false, err
		}
		return condition(tc)
	})
}

// IsTidbClusterAvailable returns true if a tidbcluster is ready for at least minReadyDuration duration; false otherwise.
func IsTidbClusterAvailable(tc *v1alpha1.TidbCluster, minReadyDuration time.Duration, now time.Time) bool {
	if !IsTidbClusterReady(tc) {
		return false
	}
	c := tidbcluster.GetTidbClusterReadyCondition(tc.Status)
	if minReadyDuration <= 0 || !c.LastTransitionTime.IsZero() && c.LastTransitionTime.Add(minReadyDuration).Before(now) {
		return true
	}
	return false
}
