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
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"k8s.io/kubernetes/test/e2e/framework"
)

// MustWaitForComponentPhase wait a component to be in a specific phase
func MustWaitForComponentPhase(c versioned.Interface, tc *v1alpha1.TidbCluster, comp v1alpha1.MemberType, phase v1alpha1.MemberPhase,
	timeout, pollInterval time.Duration) {
	lastPhase, err := WaitForComponentPhase(c, tc, comp, phase, timeout, pollInterval)
	framework.ExpectNoError(err, "failed to wait for .Status.%s.Phase of tc %s/%s to be %s, last phase is %s",
		comp, tc.Namespace, tc.Name, phase, lastPhase)
}

func WaitForComponentPhase(c versioned.Interface, tc *v1alpha1.TidbCluster, comp v1alpha1.MemberType, phase v1alpha1.MemberPhase,
	timeout, pollInterval time.Duration) (v1alpha1.MemberPhase, error) {
	var lastPhase v1alpha1.MemberPhase
	cond := func(tc *v1alpha1.TidbCluster) (bool, error) {
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

		return lastPhase == phase, nil
	}

	return lastPhase, WaitForTCCondition(c, tc.Namespace, tc.Name, timeout, pollInterval, cond)
}
