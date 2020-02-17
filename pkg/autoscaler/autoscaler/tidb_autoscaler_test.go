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

func TestSyncTiDBAfterCalculated(t *testing.T) {
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
		tc.Spec.TiDB.Replicas = test.currentReplicas
		tac.Annotations[label.AnnTiDBConsecutiveScaleInCount] = fmt.Sprintf("%d", test.currentScaleInCount)
		tac.Annotations[label.AnnTiDBConsecutiveScaleOutCount] = fmt.Sprintf("%d", test.currentScaleOutCount)

		err := syncTiDBAfterCalculated(tc, tac, test.currentReplicas, test.recommendedReplicas)
		g.Expect(err).ShouldNot(HaveOccurred())

		_, existed := tac.Annotations[label.AnnTiDBLastAutoScalingTimestamp]
		g.Expect(existed).Should(Equal(test.autoScalingPermitted))
		if test.autoScalingPermitted {
			g.Expect(tc.Spec.TiDB.Replicas).Should(Equal(test.recommendedReplicas))
		} else {
			g.Expect(tc.Spec.TiDB.Replicas).Should(Equal(test.currentReplicas))
		}

	}
	tests := []testcase{
		{
			name:                     "scale out, permitted",
			currentReplicas:          2,
			recommendedReplicas:      3,
			currentScaleOutCount:     1,
			currentScaleInCount:      0,
			expectedScaleOutAnnValue: "0",
			expectedScaleInAnnValue:  "0",
			autoScalingPermitted:     true,
		},
		{
			name:                     "scale out, rejected",
			currentReplicas:          2,
			recommendedReplicas:      3,
			currentScaleOutCount:     0,
			currentScaleInCount:      1,
			expectedScaleOutAnnValue: "1",
			expectedScaleInAnnValue:  "0",
			autoScalingPermitted:     false,
		},
		{
			name:                     "scale in, permitted",
			currentReplicas:          3,
			recommendedReplicas:      2,
			currentScaleOutCount:     0,
			currentScaleInCount:      1,
			expectedScaleInAnnValue:  "0",
			expectedScaleOutAnnValue: "0",
			autoScalingPermitted:     true,
		},
		{
			name:                     "scale in, rejected",
			currentReplicas:          3,
			recommendedReplicas:      2,
			currentScaleOutCount:     1,
			currentScaleInCount:      0,
			expectedScaleOutAnnValue: "0",
			expectedScaleInAnnValue:  "1",
			autoScalingPermitted:     false,
		},
		{
			name:                     "no scaling",
			currentReplicas:          2,
			recommendedReplicas:      2,
			currentScaleOutCount:     1,
			currentScaleInCount:      0,
			expectedScaleInAnnValue:  "0",
			expectedScaleOutAnnValue: "0",
			autoScalingPermitted:     false,
		},
	}

	for _, test := range tests {
		testFn(&test)
	}
}
