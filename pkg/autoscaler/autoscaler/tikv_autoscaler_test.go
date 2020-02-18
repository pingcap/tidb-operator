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

package autoscaler

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/label"
)

func TestSyncTiKVAfterCalculated(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name                     string
		currentReplicas          int32
		recommendedReplicas      int32
		currentScaleInCount      int32
		currentScaleOutCount     int32
		expectedScaleOutAnnValue string
		expectedScaleInAnnValue  string
		autoScalingPermitted     bool
	}
	testFn := func(test *testcase) {
		t.Log(test.name)

		tac := newTidbClusterAutoScaler()
		tc := newTidbCluster()
		tc.Spec.TiKV.Replicas = test.currentReplicas
		tac.Annotations[label.AnnTiKVConsecutiveScaleInCount] = fmt.Sprintf("%d", test.currentScaleInCount)
		tac.Annotations[label.AnnTiKVConsecutiveScaleOutCount] = fmt.Sprintf("%d", test.currentScaleOutCount)
		tac.Spec.TiDB = nil

		err := syncTiKVAfterCalculated(tc, tac, test.currentReplicas, test.recommendedReplicas)
		g.Expect(err).ShouldNot(HaveOccurred())

		_, existed := tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp]
		g.Expect(existed).Should(Equal(test.autoScalingPermitted))
		if test.autoScalingPermitted {
			g.Expect(tc.Spec.TiKV.Replicas).Should(Equal(test.recommendedReplicas))
		} else {
			g.Expect(tc.Spec.TiKV.Replicas).Should(Equal(test.currentReplicas))
		}
	}

	tests := []*testcase{
		{
			name:                     "tikv scale-out,permitted, no failure Instance",
			currentReplicas:          3,
			recommendedReplicas:      4,
			currentScaleInCount:      0,
			currentScaleOutCount:     1,
			expectedScaleOutAnnValue: "0",
			expectedScaleInAnnValue:  "0",
			autoScalingPermitted:     true,
		},
		{
			name:                     "tikv scale-out, rejected, no failure instance",
			currentReplicas:          3,
			recommendedReplicas:      4,
			currentScaleInCount:      1,
			currentScaleOutCount:     0,
			expectedScaleInAnnValue:  "0",
			expectedScaleOutAnnValue: "1",
			autoScalingPermitted:     false,
		},
		{
			name:                     "tikv scale-in, permitted, no failure instance",
			currentReplicas:          4,
			recommendedReplicas:      3,
			currentScaleInCount:      1,
			currentScaleOutCount:     0,
			expectedScaleInAnnValue:  "0",
			expectedScaleOutAnnValue: "0",
			autoScalingPermitted:     true,
		},
		{
			name:                     "tikv scale-in, rejected, no failure instace",
			currentReplicas:          4,
			recommendedReplicas:      3,
			currentScaleInCount:      0,
			currentScaleOutCount:     0,
			expectedScaleInAnnValue:  "1",
			expectedScaleOutAnnValue: "0",
			autoScalingPermitted:     false,
		},
		{
			name:                     "tikv no-scaling, no failure",
			currentReplicas:          3,
			recommendedReplicas:      3,
			currentScaleInCount:      1,
			currentScaleOutCount:     0,
			expectedScaleInAnnValue:  "0",
			expectedScaleOutAnnValue: "0",
			autoScalingPermitted:     false,
		},
	}
	for _, test := range tests {
		testFn(test)
	}
}
