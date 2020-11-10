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

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

		c := fake.NewFakeClientWithScheme(scheme.Scheme)
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
					Replicas: pointer.Int32Ptr(1),
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
				g.Expect(result.Spec.Replicas).To(Equal(pointer.Int32Ptr(1)))
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
					Replicas: pointer.Int32Ptr(1),
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
					Replicas: pointer.Int32Ptr(2),
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
				g.Expect(result.Spec.Replicas).To(Equal(pointer.Int32Ptr(2)))
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
					Replicas: pointer.Int32Ptr(1),
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
					Replicas: pointer.Int32Ptr(2),
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
				g.Expect(result.Spec.Replicas).To(Equal(pointer.Int32Ptr(2)))
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

func TestCreateOrUpdatePVC(t *testing.T) {
	g := NewGomegaWithT(t)

	type testCase struct {
		name     string
		initial  *corev1.PersistentVolumeClaim
		existing *corev1.PersistentVolumeClaim
		desired  *corev1.PersistentVolumeClaim
		expectFn func(*GomegaWithT, *FakeClientWithTracker, error)
	}
	testFn := func(tt *testCase) {
		t.Log(tt.name)

		c := fake.NewFakeClientWithScheme(scheme.Scheme)
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
			_, err := typed.CreateOrUpdatePVC(controller, tt.existing, false)
			g.Expect(err).To(Succeed())
		}
		withTracker.UpdateTracker.SetRequests(0)
		withTracker.CreateTracker.SetRequests(0)
		_, createErr := typed.CreateOrUpdatePVC(controller, tt.desired, true)
		tt.expectFn(g, withTracker, createErr)
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k": "v",
				},
			},
			StorageClassName: pointer.StringPtr("local-storage"),
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("100Gi"),
				},
			},
		},
	}
	pvc2 := pvc.DeepCopy()
	pvc2.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("200Gi")
	cases := []*testCase{
		{
			name:     "CreateOrUpdate same desired state twice will skip real updating",
			initial:  pvc,
			existing: pvc,
			desired:  pvc,
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(0))
			},
		},
		{
			name:     "update value",
			initial:  pvc2,
			existing: pvc,
			desired:  pvc2,
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(1))
			},
		},
	}

	for _, tt := range cases {
		testFn(tt)
	}
}

func TestCreateOrUpdateClusterRoleBinding(t *testing.T) {
	g := NewGomegaWithT(t)

	type testCase struct {
		name     string
		initial  *rbacv1.ClusterRoleBinding
		existing *rbacv1.ClusterRoleBinding
		desired  *rbacv1.ClusterRoleBinding
		expectFn func(*GomegaWithT, *FakeClientWithTracker, error)
	}
	testFn := func(tt *testCase) {
		t.Log(tt.name)

		c := fake.NewFakeClientWithScheme(scheme.Scheme)
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
			_, err := typed.CreateOrUpdateClusterRoleBinding(controller, tt.existing)
			g.Expect(err).To(Succeed())
		}
		withTracker.UpdateTracker.SetRequests(0)
		withTracker.CreateTracker.SetRequests(0)
		_, createErr := typed.CreateOrUpdateClusterRoleBinding(controller, tt.desired)
		tt.expectFn(g, withTracker, createErr)
	}
	rb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind: "*",
				Name: "update",
			},
		},
	}
	rb2 := rb.DeepCopy()
	rb2.Subjects[0].APIGroup = "tidbcluster.pingcap.com"
	cases := []*testCase{
		{
			name:     "CreateOrUpdate same desired state twice will skip real updating",
			initial:  rb,
			existing: rb,
			desired:  rb,
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(0))
			},
		},
		{
			name:     "update value",
			initial:  rb2,
			existing: rb,
			desired:  rb2,
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(1))
			},
		},
	}

	for _, tt := range cases {
		testFn(tt)
	}
}

func TestCreateOrUpdateClusterRole(t *testing.T) {
	g := NewGomegaWithT(t)

	type testCase struct {
		name     string
		initial  *rbacv1.ClusterRole
		existing *rbacv1.ClusterRole
		desired  *rbacv1.ClusterRole
		expectFn func(*GomegaWithT, *FakeClientWithTracker, error)
	}
	testFn := func(tt *testCase) {
		t.Log(tt.name)

		c := fake.NewFakeClientWithScheme(scheme.Scheme)
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
			_, err := typed.CreateOrUpdateClusterRole(controller, tt.existing)
			g.Expect(err).To(Succeed())
		}
		withTracker.UpdateTracker.SetRequests(0)
		withTracker.CreateTracker.SetRequests(0)
		_, createErr := typed.CreateOrUpdateClusterRole(controller, tt.desired)
		tt.expectFn(g, withTracker, createErr)
	}
	rb := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				Verbs: []string{"create", "patch"},
			},
		},
	}
	rb2 := rb.DeepCopy()
	rb2.Rules[0].APIGroups = []string{"*", "tidbcluster.pingcap.com"}
	rb2.Rules[0].Verbs = append(rb2.Rules[0].Verbs, "delete")
	cases := []*testCase{
		{
			name:     "CreateOrUpdate same desired state twice will skip real updating",
			initial:  rb,
			existing: rb,
			desired:  rb,
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(0))
			},
		},
		{
			name:     "update value",
			initial:  rb2,
			existing: rb,
			desired:  rb2,
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(1))
			},
		},
	}

	for _, tt := range cases {
		testFn(tt)
	}
}

func TestCreateOrUpdateSecret(t *testing.T) {
	g := NewGomegaWithT(t)

	type testCase struct {
		name     string
		initial  *corev1.Secret
		existing *corev1.Secret
		desired  *corev1.Secret
		expectFn func(*GomegaWithT, *FakeClientWithTracker, error)
	}
	testFn := func(tt *testCase) {
		t.Log(tt.name)

		c := fake.NewFakeClientWithScheme(scheme.Scheme)
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
			_, err := typed.CreateOrUpdateSecret(controller, tt.existing)
			g.Expect(err).To(Succeed())
		}
		withTracker.UpdateTracker.SetRequests(0)
		withTracker.CreateTracker.SetRequests(0)
		_, createErr := typed.CreateOrUpdateSecret(controller, tt.desired)
		tt.expectFn(g, withTracker, createErr)
	}
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}
	sec2 := sec.DeepCopy()
	sec2.Data["key"] = []byte("value2")
	cases := []*testCase{
		{
			name:     "CreateOrUpdate same desired state twice will skip real updating",
			initial:  sec,
			existing: sec,
			desired:  sec,
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(0))
			},
		},
		{
			name:     "update value",
			initial:  sec2,
			existing: sec,
			desired:  sec2,
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, err error) {
				g.Expect(err).To(Succeed())
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

		c := fake.NewFakeClientWithScheme(scheme.Scheme)
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
			Replicas: pointer.Int32Ptr(1),
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
					Replicas: pointer.Int32Ptr(1),
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
					Replicas: pointer.Int32Ptr(1),
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
					Replicas: pointer.Int32Ptr(2),
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
				g.Expect(result.Spec.Replicas).To(Equal(pointer.Int32Ptr(2)))
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

		c := fake.NewFakeClientWithScheme(scheme.Scheme)
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
	svc2 := svc.DeepCopy()
	svc2.Spec.Selector["k"] = "v2"
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
		{
			name:     "update value",
			initial:  svc2,
			existing: svc,
			desired:  svc2,
			expectFn: func(g *GomegaWithT, c *FakeClientWithTracker, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(c.CreateTracker.GetRequests()).To(Equal(1))
				g.Expect(c.UpdateTracker.GetRequests()).To(Equal(1))
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
