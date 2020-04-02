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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestExtractCpuRequestsRatio(t *testing.T) {
	g := NewGomegaWithT(t)
	c := newContainer()
	r, err := extractCpuRequestsRatio(c)
	g.Expect(err).Should(BeNil())
	g.Expect(almostEqual(r, 1.0)).Should(Equal(true))

	c.Resources.Requests = map[corev1.ResourceName]resource.Quantity{
		corev1.ResourceCPU: resource.MustParse("1000m"),
	}
	r, err = extractCpuRequestsRatio(c)
	g.Expect(err).Should(BeNil())
	g.Expect(almostEqual(r, 1.0)).Should(Equal(true))

	c.Resources.Requests = map[corev1.ResourceName]resource.Quantity{
		corev1.ResourceCPU: resource.MustParse("1500m"),
	}
	r, err = extractCpuRequestsRatio(c)
	g.Expect(err).Should(BeNil())
	g.Expect(almostEqual(r, 1.5)).Should(Equal(true))

	c.Resources.Requests = map[corev1.ResourceName]resource.Quantity{}
	r, err = extractCpuRequestsRatio(c)
	g.Expect(err).ShouldNot(BeNil())
	g.Expect(err.Error()).Should(Equal(fmt.Sprintf("container[%s] cpu requests is empty", c.Name)))
	g.Expect(almostEqual(r, 0)).Should(Equal(true))

}

func newContainer() *corev1.Container {
	return &corev1.Container{
		Name:  "container",
		Image: "fake:fake",
		Resources: corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU: resource.MustParse("1"),
			},
		},
	}
}
