// Copyright 2018 PingCAP, Inc.
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

package util

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestResourceRequirement(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name            string
		spec            v1alpha1.Resources
		defaultRequests []corev1.ResourceRequirements
		expectFn        func(*GomegaWithT, corev1.ResourceRequirements)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		test.expectFn(g, ResourceRequirement(test.spec, test.defaultRequests...))
	}
	tests := []testcase{
		{
			name: "don't have spec, has one defaultRequests",
			spec: v1alpha1.Resources{},
			defaultRequests: []corev1.ResourceRequirements{
				{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
				},
			},
			expectFn: func(g *GomegaWithT, req corev1.ResourceRequirements) {
				g.Expect(req).To(Equal(corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
				}))
			},
		},
		{
			name: "don't have spec, has two defaultRequests",
			spec: v1alpha1.Resources{},
			defaultRequests: []corev1.ResourceRequirements{
				{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
				},
				{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("200Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("200Gi"),
					},
				},
			},
			expectFn: func(g *GomegaWithT, req corev1.ResourceRequirements) {
				g.Expect(req).To(Equal(corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
				}))
			},
		},
		{
			name: "spec cover defaultRequests",
			spec: v1alpha1.Resources{
				Requests: &v1alpha1.ResourceRequirement{
					Memory: "200Gi",
					CPU:    "200m",
				},
				Limits: &v1alpha1.ResourceRequirement{
					Memory: "200Gi",
					CPU:    "200m",
				},
			},
			defaultRequests: []corev1.ResourceRequirements{
				{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
				},
			},
			expectFn: func(g *GomegaWithT, req corev1.ResourceRequirements) {
				g.Expect(req).To(Equal(corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("200Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("200Gi"),
					},
				}))
			},
		},
		{
			name: "spec is not correct",
			spec: v1alpha1.Resources{
				Requests: &v1alpha1.ResourceRequirement{
					Memory: "200xi",
					CPU:    "200x",
				},
				Limits: &v1alpha1.ResourceRequirement{
					Memory: "200xi",
					CPU:    "200x",
				},
			},
			defaultRequests: []corev1.ResourceRequirements{
				{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
				},
			},
			expectFn: func(g *GomegaWithT, req corev1.ResourceRequirements) {
				g.Expect(req).To(Equal(corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
				}))
			},
		},
		{
			name: "Request don't have CPU",
			spec: v1alpha1.Resources{
				Requests: &v1alpha1.ResourceRequirement{
					Memory: "100Gi",
				},
			},
			expectFn: func(g *GomegaWithT, req corev1.ResourceRequirements) {
				g.Expect(req).To(Equal(corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
				}))
			},
		},
		{
			name: "Request don't have CPU, default has",
			spec: v1alpha1.Resources{
				Requests: &v1alpha1.ResourceRequirement{
					Memory: "100Gi",
				},
			},
			defaultRequests: []corev1.ResourceRequirements{
				{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				},
			},
			expectFn: func(g *GomegaWithT, req corev1.ResourceRequirements) {
				g.Expect(req.Requests[corev1.ResourceMemory]).To(Equal(resource.MustParse("100Gi")))
				g.Expect(req.Requests[corev1.ResourceCPU]).To(Equal(resource.MustParse("100m")))
			},
		},
		{
			name: "Request don't have memory",
			spec: v1alpha1.Resources{
				Requests: &v1alpha1.ResourceRequirement{
					CPU: "100m",
				},
			},
			expectFn: func(g *GomegaWithT, req corev1.ResourceRequirements) {
				g.Expect(req).To(Equal(corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				}))
			},
		},
		{
			name: "Request don't have memory, default has",
			spec: v1alpha1.Resources{
				Requests: &v1alpha1.ResourceRequirement{
					CPU: "100m",
				},
			},
			defaultRequests: []corev1.ResourceRequirements{
				{
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
				},
			},
			expectFn: func(g *GomegaWithT, req corev1.ResourceRequirements) {
				g.Expect(req.Requests[corev1.ResourceMemory]).To(Equal(resource.MustParse("100Gi")))
				g.Expect(req.Requests[corev1.ResourceCPU]).To(Equal(resource.MustParse("100m")))
			},
		},

		{
			name: "Limits don't have CPU",
			spec: v1alpha1.Resources{
				Limits: &v1alpha1.ResourceRequirement{
					Memory: "100Gi",
				},
			},
			expectFn: func(g *GomegaWithT, req corev1.ResourceRequirements) {
				g.Expect(req).To(Equal(corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
				}))
			},
		},
		{
			name: "Limits don't have CPU, default has",
			spec: v1alpha1.Resources{
				Limits: &v1alpha1.ResourceRequirement{
					Memory: "100Gi",
				},
			},
			defaultRequests: []corev1.ResourceRequirements{
				{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				},
			},
			expectFn: func(g *GomegaWithT, req corev1.ResourceRequirements) {
				g.Expect(req.Limits[corev1.ResourceMemory]).To(Equal(resource.MustParse("100Gi")))
				g.Expect(req.Limits[corev1.ResourceCPU]).To(Equal(resource.MustParse("100m")))
			},
		},
		{
			name: "Limits don't have memory",
			spec: v1alpha1.Resources{
				Limits: &v1alpha1.ResourceRequirement{
					CPU: "100m",
				},
			},
			expectFn: func(g *GomegaWithT, req corev1.ResourceRequirements) {
				g.Expect(req).To(Equal(corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				}))
			},
		},
		{
			name: "Limits don't have memory, default has",
			spec: v1alpha1.Resources{
				Limits: &v1alpha1.ResourceRequirement{
					CPU: "100m",
				},
			},
			defaultRequests: []corev1.ResourceRequirements{
				{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("100Gi"),
					},
				},
			},
			expectFn: func(g *GomegaWithT, req corev1.ResourceRequirements) {
				g.Expect(req.Limits[corev1.ResourceMemory]).To(Equal(resource.MustParse("100Gi")))
				g.Expect(req.Limits[corev1.ResourceCPU]).To(Equal(resource.MustParse("100m")))
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestGetOrdinalFromPodName(t *testing.T) {
	g := NewGomegaWithT(t)

	i, err := GetOrdinalFromPodName("pod-1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(i).To(Equal(int32(1)))

	i, err = GetOrdinalFromPodName("pod-notint")
	g.Expect(err).To(HaveOccurred())
	g.Expect(i).To(Equal(int32(0)))
}

func TestIsSubMapOf(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(IsSubMapOf(
		nil,
		map[string]string{
			"k1": "v1",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{
			"k1": "v1",
		},
		map[string]string{
			"k1": "v1",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{
			"k1": "v1",
		},
		map[string]string{
			"k1": "v1",
			"k2": "v2",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{},
		map[string]string{
			"k1": "v1",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{
			"k1": "v1",
			"k2": "v2",
		},
		map[string]string{
			"k1": "v1",
		})).To(BeFalse())
}

func TestGetDesiredPodOrdinals(t *testing.T) {
	tests := []struct {
		name        string
		tc          *v1alpha1.TidbCluster
		memberType  v1alpha1.MemberType
		deleteSlots sets.Int
	}{
		{
			name: "no delete slots",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: v1alpha1.TiDBSpec{
						Replicas: 3,
					},
				},
			},
			memberType:  v1alpha1.TiDBMemberType,
			deleteSlots: sets.NewInt(0, 1, 2),
		},
		{
			name: "delete slots",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						label.AnnTiDBDeleteSlots: "[1,2]",
					},
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: v1alpha1.TiDBSpec{
						Replicas: 3,
					},
				},
			},
			memberType:  v1alpha1.TiDBMemberType,
			deleteSlots: sets.NewInt(0, 3, 4),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDesiredPodOrdinals(tt.tc, tt.memberType)
			if err != nil {
				t.Error(err)
			}
			if !got.Equal(tt.deleteSlots) {
				t.Errorf("expects %v got %v", tt.deleteSlots.List(), got.List())
			}
		})
	}
}
