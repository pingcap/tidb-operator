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
	"context"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/util/tidbcluster"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/e2e/framework"
	testutils "k8s.io/kubernetes/test/utils"
	ctrlCli "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	pollInterval = time.Second * 10
)

type TidbClusterConditionFn func(tc *v1alpha1.TidbCluster) (bool, error)

// WaitForTidbClusterCondition waits for a TidbCluster to be matched to the given condition.
func WaitForTidbClusterCondition(c versioned.Interface, ns, name string, timeout time.Duration, conditionFn TidbClusterConditionFn) error {
	return wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		tc, err := c.PingcapV1alpha1().TidbClusters(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			if testutils.IsRetryableAPIError(err) {
				return false, nil
			}
			return false, err
		}
		return conditionFn(tc)
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
	cond := tidbcluster.GetTidbClusterReadyCondition(tc.Status)
	if minReadyDuration <= 0 || !cond.LastTransitionTime.IsZero() && cond.LastTransitionTime.Add(minReadyDuration).Before(now) {
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

// GetSts returns the StatefulSet ns/name
// for test case simplicity, any error will panic
func GetSts(cli ctrlCli.Client, ns, name string) *appsv1.StatefulSet {
	objKey := ctrlCli.ObjectKey{Namespace: ns, Name: name}
	sts := appsv1.StatefulSet{}
	err := cli.Get(context.TODO(), objKey, &sts)
	framework.ExpectNoError(err, "failed to get StatefulSet %s/%s", ns, name)
	return &sts
}

// ListPods returns the PodList in ns with selector
func ListPods(cli ctrlCli.Client, ns string, labelSelector labels.Selector) *v1.PodList {
	options := ctrlCli.ListOptions{
		Namespace:     ns,
		LabelSelector: labelSelector,
	}
	pods := v1.PodList{}
	err := cli.List(context.TODO(), &pods, &options)
	framework.ExpectNoError(err, "failed to list Pod with options: %+v", options)
	return &pods
}
