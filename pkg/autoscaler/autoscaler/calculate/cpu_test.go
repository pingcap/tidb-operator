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
	tests := []struct {
		name          string
		defineRequest bool
		cpuValue      string
		expectedRadio float64
		occurError    bool
		errMsg        string
	}{
		{
			name:          "cpu 1",
			defineRequest: true,
			cpuValue:      "1",
			occurError:    false,
			expectedRadio: 1.0,
			errMsg:        "",
		},
		{
			name:          "cpu 1000m",
			defineRequest: true,
			cpuValue:      "1000m",
			occurError:    false,
			expectedRadio: 1.0,
			errMsg:        "",
		},
		{
			name:          "cpu 1500m",
			defineRequest: true,
			cpuValue:      "1500m",
			occurError:    false,
			expectedRadio: 1.5,
			errMsg:        "",
		},
		{
			name:          "no cpu request",
			defineRequest: false,
			cpuValue:      "",
			occurError:    true,
			expectedRadio: 0,
			errMsg:        fmt.Sprintf("container[%s] cpu requests is empty", "container"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newContainer()
			if tt.defineRequest {
				c.Resources.Requests = map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU: resource.MustParse(tt.cpuValue),
				}
			} else {
				c.Resources.Requests = map[corev1.ResourceName]resource.Quantity{}
			}
			r, err := extractCpuRequestsRatio(c)
			if !tt.occurError {
				g.Expect(err).Should(BeNil())
			} else {
				g.Expect(err).ShouldNot(BeNil())
				g.Expect(err.Error()).Should(Equal(tt.errMsg))
			}
			g.Expect(almostEqual(r, tt.expectedRadio)).Should(Equal(true))
		})
	}
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
