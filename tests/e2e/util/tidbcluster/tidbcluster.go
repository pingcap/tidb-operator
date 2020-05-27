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

package tidbcluster

import (
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/util/tidbcluster"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	testutils "k8s.io/kubernetes/test/utils"
)

var (
	pollInterval = time.Second * 10
)

type TidbClusterCondition func(tc *v1alpha1.TidbCluster) (bool, error)

// WaitForTidbClusterCondition waits for a TidbCluster to be matched to the given condition.
func WaitForTidbClusterCondition(c versioned.Interface, ns, name string, timeout time.Duration, condition TidbClusterCondition) error {
	return wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		tc, err := c.PingcapV1alpha1().TidbClusters(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			if testutils.IsRetryableAPIError(err) {
				return false, nil
			}
			return false, err
		}
		return condition(tc)
	})
}

// IsTidbClusterReady returns true if a tidbcluster is ready; false otherwise.
func IsTidbClusterReady(tc *v1alpha1.TidbCluster) bool {
	condition := tidbcluster.GetTidbClusterReadyCondition(tc.Status)
	return condition != nil && condition.Status == v1.ConditionTrue
}

// IsTidbClusterAvaiable returns true if a tidbcluster is ready for at least minReadyDuration duration; false otherwise.
func IsTidbClusterAvaiable(tc *v1alpha1.TidbCluster, minReadyDuration time.Duration, now time.Time) bool {
	if !IsTidbClusterReady(tc) {
		return false
	}
	c := tidbcluster.GetTidbClusterReadyCondition(tc.Status)
	if minReadyDuration <= 0 || !c.LastTransitionTime.IsZero() && c.LastTransitionTime.Add(minReadyDuration).Before(now) {
		return true
	}
	return false
}

// WaitForTidbClusterReady waits for a TidbClusterCondition to be ready for at least minReadyDuration duration.
func WaitForTidbClusterReady(c versioned.Interface, ns, name string, timeout time.Duration, minReadyDuration time.Duration) error {
	return WaitForTidbClusterCondition(c, ns, name, timeout, func(tc *v1alpha1.TidbCluster) (bool, error) {
		return IsTidbClusterAvaiable(tc, minReadyDuration, time.Now()), nil
	})
}
