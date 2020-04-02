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
	currentValue, targetValue := 20.0, 30.0
	currentReplicas := int32(4)
	r, err := calculate(currentValue, targetValue, currentReplicas)
	g.Expect(err).Should(BeNil())
	g.Expect(r).Should(Equal(int32(3)))

	currentValue, targetValue = 30.0, 30.0
	currentReplicas = int32(4)
	r, err = calculate(currentValue, targetValue, currentReplicas)
	g.Expect(err).Should(BeNil())
	g.Expect(r).Should(Equal(int32(4)))

	currentValue, targetValue = 35.0, 30.0
	currentReplicas = int32(4)
	r, err = calculate(currentValue, targetValue, currentReplicas)
	g.Expect(err).Should(BeNil())
	g.Expect(r).Should(Equal(int32(5)))

	currentValue, targetValue = 0.0, 0.0
	currentReplicas = int32(4)
	r, err = calculate(currentValue, targetValue, currentReplicas)
	g.Expect(err).ShouldNot(BeNil())
	g.Expect(err.Error()).Should(Equal("targetValue in calculate func can't be zero"))
	g.Expect(r).Should(Equal(int32(-1)))

}
