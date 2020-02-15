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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"k8s.io/utils/pointer"
)

func TestUpdateConsecutiveCount(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name                     string
		memberType               v1alpha1.MemberType
		currentReplicas          int32
		recommendedReplicas      int32
		currentScaleInCount      int32
		currentScaleOutCount     int32
		expectedScaleOutAnnValue string
		expectedScaleInAnnValue  string
	}

	testFn := func(test *testcase) {
		t.Log(test.name)
		tc := newTidbCluster()
		tac := newTidbClusterAutoScaler()
		tc.Annotations[fmt.Sprintf("%s.%s", test.memberType, annScaleOutSuffix)] = fmt.Sprintf("%d", test.currentScaleOutCount)
		tc.Annotations[fmt.Sprintf("%s.%s", test.memberType, annScaleInSuffix)] = fmt.Sprintf("%d", test.currentScaleInCount)

		err := updateConsecutiveCount(tc, tac, test.memberType, test.currentReplicas, test.recommendedReplicas)
		g.Expect(err).ShouldNot(HaveOccurred())
		updatedScaleOutCountAnnValue := tc.Annotations[fmt.Sprintf("%s.%s", test.memberType, annScaleOutSuffix)]
		updatedScaleInCountAnnValue := tc.Annotations[fmt.Sprintf("%s.%s", test.memberType, annScaleInSuffix)]
		g.Expect(updatedScaleOutCountAnnValue).Should(Equal(test.expectedScaleOutAnnValue))
		g.Expect(updatedScaleInCountAnnValue).Should(Equal(test.expectedScaleInAnnValue))
	}

	tests := []testcase{
		{
			name:                     "tikv, no scale",
			memberType:               v1alpha1.TiKVMemberType,
			currentReplicas:          3,
			recommendedReplicas:      3,
			currentScaleInCount:      1,
			currentScaleOutCount:     0,
			expectedScaleInAnnValue:  "0",
			expectedScaleOutAnnValue: "0",
		},
		{
			name:                     "tikv, would scale-out,first time",
			memberType:               v1alpha1.TiKVMemberType,
			currentReplicas:          3,
			recommendedReplicas:      4,
			currentScaleInCount:      1,
			currentScaleOutCount:     0,
			expectedScaleInAnnValue:  "0",
			expectedScaleOutAnnValue: "1",
		},
		{
			name:                     "tikv, would scale-out,second time",
			memberType:               v1alpha1.TiKVMemberType,
			currentReplicas:          3,
			recommendedReplicas:      4,
			currentScaleInCount:      0,
			currentScaleOutCount:     1,
			expectedScaleInAnnValue:  "0",
			expectedScaleOutAnnValue: "2",
		},
		{
			name:                     "tikv, would scale-in, first time",
			memberType:               v1alpha1.TiKVMemberType,
			currentReplicas:          4,
			recommendedReplicas:      3,
			currentScaleInCount:      0,
			currentScaleOutCount:     1,
			expectedScaleInAnnValue:  "1",
			expectedScaleOutAnnValue: "0",
		},
		{
			name:                     "tikv, would scale-in, second time",
			memberType:               v1alpha1.TiKVMemberType,
			currentReplicas:          4,
			recommendedReplicas:      3,
			currentScaleInCount:      1,
			currentScaleOutCount:     0,
			expectedScaleInAnnValue:  "2",
			expectedScaleOutAnnValue: "0",
		},
	}
	for _, test := range tests {
		testFn(&test)
	}

}

func TestCheckConsecutiveCount(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name                string
		memberType          v1alpha1.MemberType
		currentReplicas     int32
		recommendedReplicas int32
		scaleInCount        int
		scaleOutCount       int
		ableScale           bool
	}

	testFn := func(test *testcase) {
		t.Log(test.name)

		tc := newTidbCluster()
		tc.Annotations[fmt.Sprintf("%s.%s", test.memberType, annScaleOutSuffix)] = fmt.Sprintf("%d", test.scaleOutCount)
		tc.Annotations[fmt.Sprintf("%s.%s", test.memberType, annScaleInSuffix)] = fmt.Sprintf("%d", test.scaleInCount)

		tac := newTidbClusterAutoScaler()
		ableScale, err := checkConsecutiveCount(tc, tac, test.memberType, test.currentReplicas, test.recommendedReplicas)
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(ableScale).Should(Equal(test.ableScale))
	}

	tests := []testcase{
		{
			name:                "tikv success scale-out",
			memberType:          v1alpha1.TiKVMemberType,
			currentReplicas:     3,
			recommendedReplicas: 4,
			scaleInCount:        0,
			scaleOutCount:       2,
			ableScale:           true,
		},
		{
			name:                "tikv can't scale-out",
			memberType:          v1alpha1.TiKVMemberType,
			currentReplicas:     3,
			recommendedReplicas: 4,
			scaleInCount:        0,
			scaleOutCount:       1,
			ableScale:           false,
		},
		{
			name:                "tikv success scale-in",
			memberType:          v1alpha1.TiKVMemberType,
			currentReplicas:     4,
			recommendedReplicas: 3,
			scaleInCount:        2,
			scaleOutCount:       0,
			ableScale:           true,
		},
		{
			name:                "tikv can't scale-in",
			memberType:          v1alpha1.TiKVMemberType,
			currentReplicas:     4,
			recommendedReplicas: 3,
			scaleOutCount:       1,
			scaleInCount:        0,
			ableScale:           false,
		},
	}

	for _, test := range tests {
		testFn(&test)
	}
}

func newTidbClusterAutoScaler() *v1alpha1.TidbClusterAutoScaler {
	tac := &v1alpha1.TidbClusterAutoScaler{}
	tac.Spec.TiKV = &v1alpha1.TikvAutoScalerSpec{}
	tac.Spec.TiDB = &v1alpha1.TidbAutoScalerSpec{}
	tac.Spec.TiKV.ScaleOutThreshold = pointer.Int32Ptr(2)
	tac.Spec.TiKV.ScaleInThreshold = pointer.Int32Ptr(2)
	tac.Spec.TiDB.ScaleOutThreshold = pointer.Int32Ptr(2)
	tac.Spec.TiDB.ScaleInThreshold = pointer.Int32Ptr(2)
	return tac
}

func newTidbCluster() *v1alpha1.TidbCluster {
	tc := &v1alpha1.TidbCluster{}
	tc.Annotations = map[string]string{}
	tc.Name = "tc"
	tc.Namespace = "ns"
	return tc
}
