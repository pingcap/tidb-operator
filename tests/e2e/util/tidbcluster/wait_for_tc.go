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
	testutils "github.com/pingcap/tidb-operator/tests/e2e/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

type TCCondition func(tc *v1alpha1.TidbCluster) (bool, error)

// WaitForTCConditionReady waits for a TidbClusterCondition to be ready for at least minReadyDuration duration.
func WaitForTCConditionReady(c versioned.Interface, ns, name string, timeout time.Duration, minReadyDuration time.Duration) error {
	condition := func(tc *v1alpha1.TidbCluster) (bool, error) {
		return IsTidbClusterAvailable(tc, minReadyDuration, time.Now()), nil
	}

	return WaitForTCCondition(c, ns, name, timeout, time.Second*10, condition)
}

// WaitForTCCondition waits for a TidbCluster to be matched to the given condition.
//
// If tc is not found, break the poll and return error
func WaitForTCCondition(c versioned.Interface, tcNS string, tcName string, timeout, pollInterval time.Duration, condition TCCondition) error {
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

// WatchWaitForTCCondition waits for a TidbCluster to match the given condition
// using the watch API, so every status update is evaluated immediately without
// a polling delay. Use this (started before triggering a change) when the
// target state may be short-lived.
func WatchWaitForTCCondition(c versioned.Interface, tcNS string, tcName string, timeout time.Duration, condition TCCondition) error {
	fieldSelector := fields.OneTermEqualSelector("metadata.name", tcName).String()
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return c.PingcapV1alpha1().TidbClusters(tcNS).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return c.PingcapV1alpha1().TidbClusters(tcNS).Watch(context.TODO(), opts)
		},
	}

	ctx, cancel := watchtools.ContextWithOptionalTimeout(context.Background(), timeout)
	defer cancel()

	_, err := watchtools.UntilWithSync(ctx, lw, &v1alpha1.TidbCluster{}, nil,
		func(event watch.Event) (bool, error) {
			switch event.Type {
			case watch.Deleted:
				return false, nil
			case watch.Added, watch.Modified:
				tc, ok := event.Object.(*v1alpha1.TidbCluster)
				if !ok {
					return false, nil
				}
				return condition(tc)
			}
			return false, nil
		},
	)
	return err
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
