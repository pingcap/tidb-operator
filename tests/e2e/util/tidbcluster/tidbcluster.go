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
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/util/tidbcluster"
	"github.com/pingcap/tidb-operator/tests"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/e2e/framework"
	testutils "k8s.io/kubernetes/test/utils"
	ctrlCli "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	pollInterval = time.Second * 10
)

type TidbClusterCondition func(tc *v1alpha1.TidbCluster) (bool, error)

// WaitForTidbClusterCondition waits for a TidbCluster to be matched to the given condition.
func WaitForTidbClusterCondition(c versioned.Interface, ns, name string, timeout time.Duration, condition TidbClusterCondition) error {
	return wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		tc, err := c.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), name, metav1.GetOptions{})
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

// WaitForTidbClusterConditionReady waits for a TidbClusterCondition to be ready for at least minReadyDuration duration.
func WaitForTidbClusterConditionReady(c versioned.Interface, ns, name string, timeout time.Duration, minReadyDuration time.Duration) error {
	return WaitForTidbClusterCondition(c, ns, name, timeout, func(tc *v1alpha1.TidbCluster) (bool, error) {
		return IsTidbClusterAvailable(tc, minReadyDuration, time.Now()), nil
	})
}

// MustWaitForComponentPhase wait a component to be in a specific phase
func MustWaitForComponentPhase(c versioned.Interface, tc *v1alpha1.TidbCluster, comp v1alpha1.MemberType, phase v1alpha1.MemberPhase, timeout, pollInterval time.Duration) {
	var lastPhase v1alpha1.MemberPhase
	check := func(tc *v1alpha1.TidbCluster) bool {
		switch comp {
		case v1alpha1.PDMemberType:
			lastPhase = tc.Status.PD.Phase
		case v1alpha1.TiKVMemberType:
			lastPhase = tc.Status.TiKV.Phase
		case v1alpha1.TiDBMemberType:
			lastPhase = tc.Status.TiDB.Phase
		case v1alpha1.TiFlashMemberType:
			lastPhase = tc.Status.TiFlash.Phase
		case v1alpha1.TiCDCMemberType:
			lastPhase = tc.Status.TiCDC.Phase
		case v1alpha1.PumpMemberType:
			lastPhase = tc.Status.Pump.Phase
		}

		return lastPhase == phase
	}

	err := WaitForTC(c, tc, check, timeout, pollInterval)
	framework.ExpectNoError(err, "failed to wait for .Status.%s.Phase of tc %s/%s to be %s, last phase is %s",
		comp, tc.Namespace, tc.Name, phase, lastPhase)
}

// MustCreateTCWithComponentsReady create TidbCluster and wait for components ready
func MustCreateTCWithComponentsReady(cli ctrlCli.Client, oa *tests.OperatorActions, tc *v1alpha1.TidbCluster, timeout, pollInterval time.Duration) {
	err := cli.Create(context.TODO(), tc)
	framework.ExpectNoError(err, "failed to create TidbCluster %s/%s", tc.Namespace, tc.Name)
	err = oa.WaitForTidbClusterReady(tc, timeout, pollInterval)
	framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", tc.Namespace, tc.Name)
}

func MustPDHasScheduler(pdSchedulers []string, scheduler string) bool {
	for _, s := range pdSchedulers {
		if strings.Contains(s, scheduler) {
			return true
		}
	}
	return false
}

func WaitForTC(c versioned.Interface, tc *v1alpha1.TidbCluster, check func(tc *v1alpha1.TidbCluster) bool, timeout, pollInterval time.Duration) error {
	err := wait.Poll(pollInterval, timeout, func() (bool, error) {
		tc, err := c.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(context.TODO(), tc.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		done := check(tc)
		return done, nil
	})

	return err
}
