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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis"
)

func TestAffinityForNodeSelector(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name       string
		required   bool
		antiLabels map[string]string
		selector   map[string]string
		expectFn   func(*GomegaWithT, *corev1.Affinity)
	}

	antiLabels := map[string]string{"region": "region1", "zone": "zone1", "rack": "rack1", apis.LabelHostname: "host1"}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		test.expectFn(g, AffinityForNodeSelector(metav1.NamespaceDefault, test.required, test.antiLabels, test.selector))
	}

	tests := []testcase{
		{
			name:       "selector is nil",
			required:   false,
			antiLabels: nil,
			selector:   nil,
			expectFn: func(g *GomegaWithT, affinity *corev1.Affinity) {
				g.Expect(affinity).To(BeNil())
			},
		},
		{
			name:       "required, antiLabels is nil",
			required:   true,
			antiLabels: nil,
			selector:   map[string]string{"a": "a1,a2,a3", "b": "b1"},
			expectFn: func(g *GomegaWithT, affinity *corev1.Affinity) {
				affi := &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      "a",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{"a1", "a2", "a3"},
										},
										{
											Key:      "b",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{"b1"},
										},
									},
								},
							},
						},
					},
				}
				g.Expect(affinity).To(Equal(affi))
			},
		},
		{
			name:       "required, antiLabels is not nil",
			required:   true,
			antiLabels: antiLabels,
			selector:   map[string]string{"a": "a1,a2,a3", "b": "b1"},
			expectFn: func(g *GomegaWithT, affinity *corev1.Affinity) {
				affi := &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      "a",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{"a1", "a2", "a3"},
										},
										{
											Key:      "b",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{"b1"},
										},
									},
								},
							},
						},
					},
					PodAntiAffinity: &corev1.PodAntiAffinity{
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
							{
								Weight: 80,
								PodAffinityTerm: corev1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{MatchLabels: antiLabels},
									TopologyKey:   apis.LabelHostname,
									Namespaces:    []string{metav1.NamespaceDefault},
								},
							},
							{
								Weight: 40,
								PodAffinityTerm: corev1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{MatchLabels: antiLabels},
									TopologyKey:   "rack",
									Namespaces:    []string{metav1.NamespaceDefault},
								},
							},
							{
								Weight: 10,
								PodAffinityTerm: corev1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{MatchLabels: antiLabels},
									TopologyKey:   "region",
									Namespaces:    []string{metav1.NamespaceDefault},
								},
							},
							{
								Weight: 20,
								PodAffinityTerm: corev1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{MatchLabels: antiLabels},
									TopologyKey:   "zone",
									Namespaces:    []string{metav1.NamespaceDefault},
								},
							},
						},
					},
				}
				g.Expect(affinity).To(Equal(affi))
			},
		},
		{
			name:       "not required",
			required:   false,
			antiLabels: nil,
			selector: map[string]string{
				"region": "region1",
				"zone":   "zone1,zone2",
				"rack":   "",
				"a":      "a1,a2,a3",
				"b":      "b1",
			},
			expectFn: func(g *GomegaWithT, affinity *corev1.Affinity) {
				affi := &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      "a",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{"a1,a2,a3"},
										},
										{
											Key:      "b",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{"b1"},
										},
									},
								},
							},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
							{
								Weight: 10,
								Preference: corev1.NodeSelectorTerm{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      "region",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{"region1"},
										},
									},
								},
							},
							{
								Weight: 20,
								Preference: corev1.NodeSelectorTerm{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      "zone",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{"zone1", "zone2"},
										},
									},
								},
							},
						},
					},
				}
				g.Expect(affinity).To(Equal(affi))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestResourceRequirement(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name            string
		spec            v1alpha1.ContainerSpec
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
			spec: v1alpha1.ContainerSpec{},
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
			spec: v1alpha1.ContainerSpec{},
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
			spec: v1alpha1.ContainerSpec{
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
			spec: v1alpha1.ContainerSpec{
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
			spec: v1alpha1.ContainerSpec{
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
			spec: v1alpha1.ContainerSpec{
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
			name: "Request don't have Momory",
			spec: v1alpha1.ContainerSpec{
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
			name: "Request don't have Momory, default has",
			spec: v1alpha1.ContainerSpec{
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
			spec: v1alpha1.ContainerSpec{
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
			spec: v1alpha1.ContainerSpec{
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
			name: "Limits don't have Momory",
			spec: v1alpha1.ContainerSpec{
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
			name: "Limits don't have Momory, default has",
			spec: v1alpha1.ContainerSpec{
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

func TestGetNextOrdinalPodName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(GetNextOrdinalPodName("pod-1", 1)).To(Equal("pod-2"))
}
