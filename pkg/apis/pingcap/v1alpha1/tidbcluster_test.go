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

package v1alpha1

import (
	"testing"

	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

func TestPDIsAvailable(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		update   func(*TidbCluster)
		expectFn func(*GomegaWithT, bool)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbCluster()
		test.update(tc)
		test.expectFn(g, tc.PDIsAvailable())
	}
	tests := []testcase{
		{
			name: "pd members count is 1",
			update: func(tc *TidbCluster) {
				tc.Status.PD.Members = map[string]PDMember{
					"pd-0": {Name: "pd-0", Health: true},
				}
			},
			expectFn: func(g *GomegaWithT, b bool) {
				g.Expect(b).To(BeFalse())
			},
		},
		{
			name: "pd members count is 2, but health count is 1",
			update: func(tc *TidbCluster) {
				tc.Status.PD.Members = map[string]PDMember{
					"pd-0": {Name: "pd-0", Health: true},
					"pd-1": {Name: "pd-1", Health: false},
				}
			},
			expectFn: func(g *GomegaWithT, b bool) {
				g.Expect(b).To(BeFalse())
			},
		},
		{
			name: "pd members count is 3, health count is 3, but ready replicas is 1",
			update: func(tc *TidbCluster) {
				tc.Status.PD.Members = map[string]PDMember{
					"pd-0": {Name: "pd-0", Health: true},
					"pd-1": {Name: "pd-1", Health: true},
					"pd-2": {Name: "pd-2", Health: true},
				}
				tc.Status.PD.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 1}
			},
			expectFn: func(g *GomegaWithT, b bool) {
				g.Expect(b).To(BeFalse())
			},
		},
		{
			name: "pd is available",
			update: func(tc *TidbCluster) {
				tc.Status.PD.Members = map[string]PDMember{
					"pd-0": {Name: "pd-0", Health: true},
					"pd-1": {Name: "pd-1", Health: true},
					"pd-2": {Name: "pd-2", Health: true},
				}
				tc.Status.PD.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
			},
			expectFn: func(g *GomegaWithT, b bool) {
				g.Expect(b).To(BeTrue())
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTiKVIsAvailable(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		update   func(*TidbCluster)
		expectFn func(*GomegaWithT, bool)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbCluster()
		test.update(tc)
		test.expectFn(g, tc.TiKVIsAvailable())
	}
	tests := []testcase{
		{
			name: "tikv stores count is 0",
			update: func(tc *TidbCluster) {
				tc.Status.TiKV.Stores = map[string]TiKVStore{}
			},
			expectFn: func(g *GomegaWithT, b bool) {
				g.Expect(b).To(BeFalse())
			},
		},
		{
			name: "tikv stores count is 1, but available count is 0",
			update: func(tc *TidbCluster) {
				tc.Status.TiKV.Stores = map[string]TiKVStore{
					"tikv-0": {PodName: "tikv-0", State: TiKVStateDown},
				}
			},
			expectFn: func(g *GomegaWithT, b bool) {
				g.Expect(b).To(BeFalse())
			},
		},
		{
			name: "tikv stores count is 1, available count is 1, ready replicas is 0",
			update: func(tc *TidbCluster) {
				tc.Status.TiKV.Stores = map[string]TiKVStore{
					"tikv-0": {PodName: "tikv-0", State: TiKVStateUp},
				}
				tc.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 0}
			},
			expectFn: func(g *GomegaWithT, b bool) {
				g.Expect(b).To(BeFalse())
			},
		},
		{
			name: "tikv is available",
			update: func(tc *TidbCluster) {
				tc.Status.TiKV.Stores = map[string]TiKVStore{
					"tikv-0": {PodName: "tikv-0", State: TiKVStateUp},
				}
				tc.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 1}
			},
			expectFn: func(g *GomegaWithT, b bool) {
				g.Expect(b).To(BeTrue())
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestComponentAccessor(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name      string
		cluster   *TidbClusterSpec
		component *ComponentSpec
		expectFn  func(*GomegaWithT, ComponentAccessor)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		accessor := &componentAccessorImpl{test.cluster, test.component}
		test.expectFn(g, accessor)
	}
	affinity := &corev1.Affinity{
		PodAffinity: &corev1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
				TopologyKey: "rack",
			}},
		},
	}
	toleration1 := corev1.Toleration{
		Key: "k1",
	}
	toleration2 := corev1.Toleration{
		Key: "k2",
	}
	tests := []testcase{
		{
			name: "use cluster-level defaults",
			cluster: &TidbClusterSpec{
				ImagePullPolicy:   corev1.PullNever,
				HostNetwork:       pointer.BoolPtr(true),
				Affinity:          affinity,
				PriorityClassName: pointer.StringPtr("test"),
				SchedulerName:     "test",
			},
			component: &ComponentSpec{},
			expectFn: func(g *GomegaWithT, a ComponentAccessor) {
				g.Expect(a.ImagePullPolicy()).Should(Equal(corev1.PullNever))
				g.Expect(a.HostNetwork()).Should(Equal(true))
				g.Expect(a.Affinity()).Should(Equal(affinity))
				g.Expect(*a.PriorityClassName()).Should(Equal("test"))
				g.Expect(a.SchedulerName()).Should(Equal("test"))
			},
		},
		{
			name: "override at component-level",
			cluster: &TidbClusterSpec{
				ImagePullPolicy:   corev1.PullNever,
				HostNetwork:       pointer.BoolPtr(true),
				Affinity:          nil,
				PriorityClassName: pointer.StringPtr("test"),
				SchedulerName:     "test",
			},
			component: &ComponentSpec{
				ImagePullPolicy:   func() *corev1.PullPolicy { a := corev1.PullAlways; return &a }(),
				HostNetwork:       func() *bool { a := false; return &a }(),
				Affinity:          affinity,
				PriorityClassName: pointer.StringPtr("override"),
				SchedulerName:     pointer.StringPtr("override"),
			},
			expectFn: func(g *GomegaWithT, a ComponentAccessor) {
				g.Expect(a.ImagePullPolicy()).Should(Equal(corev1.PullAlways))
				g.Expect(a.HostNetwork()).Should(Equal(false))
				g.Expect(a.Affinity()).Should(Equal(affinity))
				g.Expect(*a.PriorityClassName()).Should(Equal("override"))
				g.Expect(a.SchedulerName()).Should(Equal("override"))
			},
		},
		{
			name: "node selector merge",
			cluster: &TidbClusterSpec{
				NodeSelector: map[string]string{
					"k1": "v1",
				},
			},
			component: &ComponentSpec{
				NodeSelector: map[string]string{
					"k1": "v2",
					"k3": "v3",
				},
			},
			expectFn: func(g *GomegaWithT, a ComponentAccessor) {
				g.Expect(a.NodeSelector()).Should(Equal(map[string]string{
					"k1": "v2",
					"k3": "v3",
				}))
			},
		},
		{
			name: "annotations merge",
			cluster: &TidbClusterSpec{
				Annotations: map[string]string{
					"k1": "v1",
				},
			},
			component: &ComponentSpec{
				Annotations: map[string]string{
					"k1": "v2",
					"k3": "v3",
				},
			},
			expectFn: func(g *GomegaWithT, a ComponentAccessor) {
				g.Expect(a.Annotations()).Should(Equal(map[string]string{
					"k1": "v2",
					"k3": "v3",
				}))
			},
		},
		{
			name: "annotations merge",
			cluster: &TidbClusterSpec{
				Annotations: map[string]string{
					"k1": "v1",
				},
			},
			component: &ComponentSpec{
				Annotations: map[string]string{
					"k1": "v2",
					"k3": "v3",
				},
			},
			expectFn: func(g *GomegaWithT, a ComponentAccessor) {
				g.Expect(a.Annotations()).Should(Equal(map[string]string{
					"k1": "v2",
					"k3": "v3",
				}))
			},
		},
		{
			name: "tolerations merge",
			cluster: &TidbClusterSpec{
				Tolerations: []corev1.Toleration{toleration1},
			},
			component: &ComponentSpec{
				Tolerations: []corev1.Toleration{toleration2},
			},
			expectFn: func(g *GomegaWithT, a ComponentAccessor) {
				g.Expect(a.Tolerations()).Should(ConsistOf(toleration2))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestHelperImage(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		update   func(*TidbCluster)
		expectFn func(*GomegaWithT, string)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbCluster()
		test.update(tc)
		test.expectFn(g, tc.HelperImage())
	}
	tests := []testcase{
		{
			name:   "helper image has defaults",
			update: func(tc *TidbCluster) {},
			expectFn: func(g *GomegaWithT, s string) {
				g.Expect(s).ShouldNot(BeEmpty())
			},
		},
		{
			name: "helper image use .spec.helper.image first",
			update: func(tc *TidbCluster) {
				tc.Spec.Helper = &HelperSpec{
					Image: pointer.StringPtr("helper1"),
				}
				tc.Spec.TiDB.SlowLogTailer = &TiDBSlowLogTailerSpec{
					Image: pointer.StringPtr("helper2"),
				}
			},
			expectFn: func(g *GomegaWithT, s string) {
				g.Expect(s).Should(Equal("helper1"))
			},
		},
		{
			name: "pick .spec.tidb.slowLogTailer.image as helper for backward compatibility",
			update: func(tc *TidbCluster) {
				tc.Spec.Helper = &HelperSpec{
					Image: nil,
				}
				tc.Spec.TiDB.SlowLogTailer = &TiDBSlowLogTailerSpec{
					Image: pointer.StringPtr("helper2"),
				}
			},
			expectFn: func(g *GomegaWithT, s string) {
				g.Expect(s).Should(Equal("helper2"))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestHelperImagePullPolicy(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		update   func(*TidbCluster)
		expectFn func(*GomegaWithT, corev1.PullPolicy)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbCluster()
		test.update(tc)
		test.expectFn(g, tc.HelperImagePullPolicy())
	}
	tests := []testcase{
		{
			name: "use .spec.helper.imagePullPolicy first",
			update: func(tc *TidbCluster) {
				tc.Spec.Helper = &HelperSpec{
					ImagePullPolicy: func() *corev1.PullPolicy { a := corev1.PullAlways; return &a }(),
				}
				tc.Spec.TiDB.SlowLogTailer = &TiDBSlowLogTailerSpec{
					ImagePullPolicy: func() *corev1.PullPolicy { a := corev1.PullIfNotPresent; return &a }(),
				}
				tc.Spec.ImagePullPolicy = corev1.PullNever
			},
			expectFn: func(g *GomegaWithT, p corev1.PullPolicy) {
				g.Expect(p).Should(Equal(corev1.PullAlways))
			},
		},
		{
			name: "pick .spec.tidb.slowLogTailer.imagePullPolicy when .spec.helper.imagePullPolicy is nil",
			update: func(tc *TidbCluster) {
				tc.Spec.Helper = &HelperSpec{
					ImagePullPolicy: nil,
				}
				tc.Spec.TiDB.SlowLogTailer = &TiDBSlowLogTailerSpec{
					ImagePullPolicy: func() *corev1.PullPolicy { a := corev1.PullIfNotPresent; return &a }(),
				}
				tc.Spec.ImagePullPolicy = corev1.PullNever
			},
			expectFn: func(g *GomegaWithT, p corev1.PullPolicy) {
				g.Expect(p).Should(Equal(corev1.PullIfNotPresent))
			},
		},
		{
			name: "pick cluster one if both .spec.tidb.slowLogTailer.imagePullPolicy and .spec.helper.imagePullPolicy are nil",
			update: func(tc *TidbCluster) {
				tc.Spec.ImagePullPolicy = corev1.PullNever
			},
			expectFn: func(g *GomegaWithT, p corev1.PullPolicy) {
				g.Expect(p).Should(Equal(corev1.PullNever))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestPDVersion(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		update   func(*TidbCluster)
		expectFn func(*GomegaWithT, *TidbCluster)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbCluster()
		test.update(tc)
		test.expectFn(g, tc)
	}
	tests := []testcase{
		{
			name: "has tag",
			update: func(tc *TidbCluster) {
				tc.Spec.PD.Image = "pingcap/pd:v3.1.0"
			},
			expectFn: func(g *GomegaWithT, tc *TidbCluster) {
				g.Expect(tc.PDVersion()).To(Equal("v3.1.0"))
			},
		},
		{
			name: "don't have tag",
			update: func(tc *TidbCluster) {
				tc.Spec.PD.Image = "pingcap/pd"
			},
			expectFn: func(g *GomegaWithT, tc *TidbCluster) {
				g.Expect(tc.PDVersion()).To(Equal("latest"))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newTidbCluster() *TidbCluster {
	return &TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pd",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: TidbClusterSpec{
			PD: &PDSpec{
				Replicas: 3,
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("10G"),
					},
				},
			},
			TiKV: &TiKVSpec{
				Replicas: 3,
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("10G"),
					},
				},
			},
			TiDB: &TiDBSpec{
				Replicas: 1,
			},
		},
	}
}
