// Copyright 2019. PingCAP, Inc.
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

package controller

import (
	"testing"

	"github.com/pingcap/tidb-operator/pkg/scheme"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/gomega"
)

// TODO: more UTs
func TestGenericControlInterface_CreateOrUpdate(t *testing.T) {
	g := NewGomegaWithT(t)

	type testCase struct {
		name     string
		existing *appsv1.Deployment
		desired  *appsv1.Deployment
		mergeFn  MergeFn
		expectFn func(*GomegaWithT, client.Client, *appsv1.Deployment, error)
	}
	testFn := func(tt *testCase) {
		t.Log(tt.name)

		var c client.Client
		if tt.existing != nil {
			c = fake.NewFakeClientWithScheme(scheme.Scheme, tt.existing)
		} else {
			c = fake.NewFakeClientWithScheme(scheme.Scheme)
		}
		recorder := record.NewFakeRecorder(10)
		control := NewRealGenericControl(c, recorder)
		controller := newTidbCluster()
		result, err := control.CreateOrUpdate(controller, tt.desired, tt.mergeFn)
		tt.expectFn(g, c, result.(*appsv1.Deployment), err)
	}
	mergeFn := func(existing, desired runtime.Object) error {
		e := existing.(*appsv1.Deployment)
		d := desired.(*appsv1.Deployment)
		e.Spec.Replicas = d.Spec.Replicas
		e.Spec.Template = d.Spec.Template
		return nil
	}
	cases := []*testCase{
		{
			name:     "Create",
			existing: nil,
			desired: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: Int32Ptr(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							DNSPolicy: corev1.DNSClusterFirst,
						},
					},
				},
			},
			mergeFn: mergeFn,
			expectFn: func(g *GomegaWithT, c client.Client, result *appsv1.Deployment, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(result.Spec.Replicas).To(Equal(Int32Ptr(1)))
			},
		},
		{
			name: "Update",
			existing: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: Int32Ptr(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							DNSPolicy: corev1.DNSClusterFirst,
						},
					},
				},
			},
			desired: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: Int32Ptr(2),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							DNSPolicy: corev1.DNSClusterFirstWithHostNet,
						},
					},
				},
			},
			mergeFn: mergeFn,
			expectFn: func(g *GomegaWithT, c client.Client, result *appsv1.Deployment, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(result.Spec.Replicas).To(Equal(Int32Ptr(2)))
				g.Expect(result.Spec.Template.Spec.DNSPolicy).To(Equal(corev1.DNSClusterFirstWithHostNet))
			},
		},
		{
			name: "Partial update",
			existing: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: Int32Ptr(1),
					Paused:   true,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							DNSPolicy: corev1.DNSClusterFirst,
						},
					},
				},
			},
			desired: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: Int32Ptr(2),
					Paused:   false,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							DNSPolicy: corev1.DNSClusterFirstWithHostNet,
						},
					},
				},
			},
			mergeFn: mergeFn,
			expectFn: func(g *GomegaWithT, c client.Client, result *appsv1.Deployment, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(result.Spec.Replicas).To(Equal(Int32Ptr(2)))
				g.Expect(result.Spec.Template.Spec.DNSPolicy).To(Equal(corev1.DNSClusterFirstWithHostNet))
				g.Expect(result.Spec.Paused).To(BeTrue())
			},
		},
	}

	for _, tt := range cases {
		testFn(tt)
	}
}

func TestCreateOrUpdateDeployment(t *testing.T) {
	g := NewGomegaWithT(t)

	type testCase struct {
		name     string
		existing *appsv1.Deployment
		desired  *appsv1.Deployment
		expectFn func(*GomegaWithT, client.Client, *appsv1.Deployment, error)
	}
	testFn := func(tt *testCase) {
		t.Log(tt.name)

		var c client.Client
		if tt.existing != nil {
			c = fake.NewFakeClientWithScheme(scheme.Scheme, tt.existing)
		} else {
			c = fake.NewFakeClientWithScheme(scheme.Scheme)
		}
		recorder := record.NewFakeRecorder(10)
		control := NewRealGenericControl(c, recorder)
		typed := NewTypedControl(control)
		controller := newTidbCluster()
		result, createErr := typed.CreateOrUpdateDeployment(controller, tt.desired)
		tt.expectFn(g, c, result, createErr)
	}
	cases := []*testCase{
		{
			name:     "It should adopt the deployment on creation",
			existing: nil,
			desired: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: Int32Ptr(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							DNSPolicy: corev1.DNSClusterFirst,
						},
					},
				},
			},
			expectFn: func(g *GomegaWithT, c client.Client, result *appsv1.Deployment, err error) {
				g.Expect(err).To(Succeed())
				g.Eventually(result.OwnerReferences).Should(HaveLen(1))
				g.Eventually(result.OwnerReferences[0].Kind).Should(Equal("TidbCluster"))
			},
		},
		{
			name: "It should not mutate label selector on update",
			existing: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
						"k": "v",
					}},
					Replicas: Int32Ptr(1),
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"k": "v",
							},
						},
						Spec: corev1.PodSpec{
							DNSPolicy: corev1.DNSClusterFirst,
						},
					},
				},
			},
			desired: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
						"k2": "v2",
					}},
					Replicas: Int32Ptr(2),
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"k2": "v2",
							},
						},
						Spec: corev1.PodSpec{
							DNSPolicy: corev1.DNSClusterFirstWithHostNet,
						},
					},
				},
			},
			expectFn: func(g *GomegaWithT, c client.Client, result *appsv1.Deployment, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(result.Spec.Replicas).To(Equal(Int32Ptr(2)))
				g.Expect(result.Spec.Template.Spec.DNSPolicy).To(Equal(corev1.DNSClusterFirstWithHostNet))
				g.Expect(result.Spec.Selector.MatchLabels["k"]).To(Equal("v"))
				g.Expect(result.Spec.Template.Labels["k"]).To(Equal("v"))
			},
		},
	}

	for _, tt := range cases {
		testFn(tt)
	}
}
