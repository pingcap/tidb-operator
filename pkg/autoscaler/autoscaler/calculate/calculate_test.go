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

package calculate

import (
	. "github.com/onsi/gomega"
	"testing"
)

func TestCalculate(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name             string
		currentReplicas  int32
		currentValue     float64
		targetValue      float64
		expectedReplicas int32
		errMsg           string
	}{
		{
			name:             "under target value",
			currentReplicas:  4,
			currentValue:     20.0,
			targetValue:      30.0,
			expectedReplicas: 3,
		},
		{
			name:             "equal target value",
			currentReplicas:  4,
			currentValue:     30.0,
			targetValue:      30.0,
			expectedReplicas: 4,
		},
		{
			name:             "greater than target value",
			currentReplicas:  4,
			currentValue:     35.0,
			targetValue:      30.0,
			expectedReplicas: 5,
		},
		{
			name:             "target value is zero",
			currentReplicas:  4,
			currentValue:     35.0,
			targetValue:      0,
			expectedReplicas: -1,
			errMsg:           "targetValue in calculate func can't be zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := calculate(tt.currentValue, tt.targetValue, tt.currentReplicas)
			if len(tt.errMsg) < 1 {
				g.Expect(err).Should(BeNil())
			} else {
				g.Expect(err).ShouldNot(BeNil())
				g.Expect(err.Error()).Should(Equal(tt.errMsg))
			}
			g.Expect(r).Should(Equal(tt.expectedReplicas))
		})
	}
}
