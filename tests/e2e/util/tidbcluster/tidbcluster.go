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
	"github.com/pingcap/tidb-operator/pkg/util/tidbcluster"
	"github.com/pingcap/tidb-operator/tests"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	ctrlCli "sigs.k8s.io/controller-runtime/pkg/client"
)

// IsTidbClusterReady returns true if a tidbcluster is ready; false otherwise.
func IsTidbClusterReady(tc *v1alpha1.TidbCluster) bool {
	condition := tidbcluster.GetTidbClusterReadyCondition(tc.Status)
	return condition != nil && condition.Status == v1.ConditionTrue
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
