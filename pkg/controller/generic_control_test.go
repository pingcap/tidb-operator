// Copyright 2019 PingCAP, Inc.
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
	"context"
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
		expectFn func(*GomegaWithT, *FakeClientWithTracker, *appsv1.Deployment, error)
	}
	testFn := func(tt *testCase) {
		t.Log(tt.name)

		var c client.Client
		c = fake.NewFakeClientWithScheme(scheme.Scheme)
		withTracker := NewFakeClientWithTracker(c)
		recorder := record.NewFakeRecorder(10)
		control := NewRealGenericControl(withTracker, recorder)
		controller := newTidbCluster()
		if tt.existing != nil {
			_, err := control.CreateOrUpdate(controller, tt.existing, tt.mergeFn, true)
			g.Expect(err).To(Succeed())
		}
		withTracker.UpdateTracker.SetRequests(0)
		withTracker.CreateTracker.SetRequests(0)
		result, err := control.CreateOrUpdate(controller, tt.desired, tt.mergeFn, true)
		tt.expectFn(g, withTracker, result.(*appsv1.Deployment), err)
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
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, result *appsv1.Deployment, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(result.Spec.Replicas).To(Equal(Int32Ptr(1)))
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(0))
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
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, result *appsv1.Deployment, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(result.Spec.Replicas).To(Equal(Int32Ptr(2)))
				g.Expect(result.Spec.Template.Spec.DNSPolicy).To(Equal(corev1.DNSClusterFirstWithHostNet))
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(1))
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
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, result *appsv1.Deployment, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(result.Spec.Replicas).To(Equal(Int32Ptr(2)))
				g.Expect(result.Spec.Template.Spec.DNSPolicy).To(Equal(corev1.DNSClusterFirstWithHostNet))
				g.Expect(result.Spec.Paused).To(BeTrue())
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(1))
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
		initial  *appsv1.Deployment
		existing *appsv1.Deployment
		desired  *appsv1.Deployment
		expectFn func(*GomegaWithT, *FakeClientWithTracker, *appsv1.Deployment, error)
	}
	testFn := func(tt *testCase) {
		t.Log(tt.name)

		var c client.Client
		c = fake.NewFakeClientWithScheme(scheme.Scheme)
		withTracker := NewFakeClientWithTracker(c)
		recorder := record.NewFakeRecorder(10)
		control := NewRealGenericControl(withTracker, recorder)
		typed := NewTypedControl(control)
		controller := newTidbCluster()
		if tt.initial != nil {
			err := typed.Create(controller, tt.initial)
			g.Expect(err).To(Succeed())
		}
		if tt.existing != nil {
			_, err := typed.CreateOrUpdateDeployment(controller, tt.existing)
			g.Expect(err).To(Succeed())
		}
		withTracker.UpdateTracker.SetRequests(0)
		withTracker.CreateTracker.SetRequests(0)
		result, createErr := typed.CreateOrUpdateDeployment(controller, tt.desired)
		tt.expectFn(g, withTracker, result, createErr)
	}
	dep := &appsv1.Deployment{
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
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, result *appsv1.Deployment, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(result.OwnerReferences).Should(HaveLen(1))
				g.Expect(result.OwnerReferences[0].Kind).Should(Equal("TidbCluster"))
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(0))
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
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, result *appsv1.Deployment, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(result.Spec.Replicas).To(Equal(Int32Ptr(2)))
				g.Expect(result.Spec.Template.Spec.DNSPolicy).To(Equal(corev1.DNSClusterFirstWithHostNet))
				g.Expect(result.Spec.Selector.MatchLabels["k"]).To(Equal("v"))
				g.Expect(result.Spec.Template.Labels["k"]).To(Equal("v"))
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(1))
			},
		},
		{
			name:     "CreateOrUpdate same desired state twice will skip real updating",
			initial:  dep,
			existing: dep,
			desired:  dep,
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, result *appsv1.Deployment, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(0))
			},
		},
	}

	for _, tt := range cases {
		testFn(tt)
	}
}

func TestCreateOrUpdateService(t *testing.T) {
	g := NewGomegaWithT(t)

	type testCase struct {
		name     string
		initial  *corev1.Service
		existing *corev1.Service
		desired  *corev1.Service
		expectFn func(*GomegaWithT, *FakeClientWithTracker, error)
	}
	testFn := func(tt *testCase) {
		t.Log(tt.name)

		var c client.Client
		c = fake.NewFakeClientWithScheme(scheme.Scheme)
		withTracker := NewFakeClientWithTracker(c)
		recorder := record.NewFakeRecorder(10)
		control := NewRealGenericControl(withTracker, recorder)
		typed := NewTypedControl(control)
		controller := newTidbCluster()
		if tt.initial != nil {
			err := typed.Create(controller, tt.initial)
			g.Expect(err).To(Succeed())
		}
		if tt.existing != nil {
			_, err := typed.CreateOrUpdateService(controller, tt.existing)
			g.Expect(err).To(Succeed())
		}
		withTracker.UpdateTracker.SetRequests(0)
		withTracker.CreateTracker.SetRequests(0)
		_, createErr := typed.CreateOrUpdateService(controller, tt.desired)
		tt.expectFn(g, withTracker, createErr)
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"k": "v",
			},
			Ports: []corev1.ServicePort{{
				Port: 9090,
			}},
		},
	}
	cases := []*testCase{
		{
			name:     "CreateOrUpdate same desired state twice will skip real updating",
			initial:  svc,
			existing: svc,
			desired:  svc,
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(0))
			},
		},
	}

	for _, tt := range cases {
		testFn(tt)
	}
}

type FakeClientWithTracker struct {
	client.Client
	CreateTracker RequestTracker
	UpdateTracker RequestTracker
}

func NewFakeClientWithTracker(cli client.Client) *FakeClientWithTracker {
	return &FakeClientWithTracker{
		Client:        cli,
		UpdateTracker: RequestTracker{},
		CreateTracker: RequestTracker{},
	}
}

func (c *FakeClientWithTracker) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	c.CreateTracker.Inc()
	return c.Client.Create(ctx, obj, opts...)
}

func (c *FakeClientWithTracker) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	c.UpdateTracker.Inc()
	return c.Client.Update(ctx, obj, opts...)
}
