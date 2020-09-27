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

package v1alpha1

import (
	"testing"

	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

func TestDMMasterIsAvailable(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		update   func(*DMCluster)
		expectFn func(*GomegaWithT, bool)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		dc := newDMCluster()
		test.update(dc)
		test.expectFn(g, dc.MasterIsAvailable())
	}
	tests := []testcase{
		{
			name: "dm-master members count is 1",
			update: func(dc *DMCluster) {
				dc.Status.Master.Members = map[string]MasterMember{
					"dm-master-0": {Name: "dm-master-0", Health: true},
				}
			},
			expectFn: func(g *GomegaWithT, b bool) {
				g.Expect(b).To(BeFalse())
			},
		},
		{
			name: "dm-master members count is 2, but health count is 1",
			update: func(dc *DMCluster) {
				dc.Status.Master.Members = map[string]MasterMember{
					"dm-master-0": {Name: "dm-master-0", Health: true},
					"dm-master-1": {Name: "dm-master-1", Health: false},
				}
			},
			expectFn: func(g *GomegaWithT, b bool) {
				g.Expect(b).To(BeFalse())
			},
		},
		{
			name: "dm-master members count is 3, health count is 3, but ready replicas is 1",
			update: func(dc *DMCluster) {
				dc.Status.Master.Members = map[string]MasterMember{
					"dm-master-0": {Name: "dm-master-0", Health: true},
					"dm-master-1": {Name: "dm-master-1", Health: true},
					"dm-master-2": {Name: "dm-master-2", Health: true},
				}
				dc.Status.Master.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 1}
			},
			expectFn: func(g *GomegaWithT, b bool) {
				g.Expect(b).To(BeFalse())
			},
		},
		{
			name: "dm-master is available",
			update: func(dc *DMCluster) {
				dc.Status.Master.Members = map[string]MasterMember{
					"dm-master-0": {Name: "dm-master-0", Health: true},
					"dm-master-1": {Name: "dm-master-1", Health: true},
					"dm-master-2": {Name: "dm-master-2", Health: true},
				}
				dc.Status.Master.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
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

func TestDMComponentAccessor(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name      string
		cluster   *DMClusterSpec
		component *ComponentSpec
		expectFn  func(*GomegaWithT, ComponentAccessor)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		accessor := buildDMClusterComponentAccessor(test.cluster, test.component)
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
			cluster: &DMClusterSpec{
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
			cluster: &DMClusterSpec{
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
			cluster: &DMClusterSpec{
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
			cluster: &DMClusterSpec{
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
			cluster: &DMClusterSpec{
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
			cluster: &DMClusterSpec{
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

func TestMasterVersion(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		update   func(*DMCluster)
		expectFn func(*GomegaWithT, *DMCluster)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		dc := newDMCluster()
		test.update(dc)
		test.expectFn(g, dc)
	}
	tests := []testcase{
		{
			name: "has tag",
			update: func(dc *DMCluster) {
				dc.Spec.Master.BaseImage = "pingcap/dm:v2.0.0-rc.2"
			},
			expectFn: func(g *GomegaWithT, dc *DMCluster) {
				g.Expect(dc.MasterVersion()).To(Equal("v2.0.0-rc.2"))
			},
		},
		{
			name: "don't have tag",
			update: func(dc *DMCluster) {
				dc.Spec.Master.BaseImage = "pingcap/pd"
			},
			expectFn: func(g *GomegaWithT, dc *DMCluster) {
				g.Expect(dc.MasterVersion()).To(Equal("latest"))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newDMCluster() *DMCluster {
	return &DMCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DMCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dm-master",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: DMClusterSpec{
			Master: MasterSpec{
				Replicas:    3,
				StorageSize: "10G",
			},
			Worker: &WorkerSpec{
				Replicas:    3,
				StorageSize: "10G",
			},
		},
	}
}
