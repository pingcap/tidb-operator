// Copyright 2022 PingCAP, Inc.
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

package suspender

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

func TestSuspendComponent(t *testing.T) {
	g := NewGomegaWithT(t)

	cases := map[string]struct {
		setup          func(cluster v1alpha1.Cluster)
		component      v1alpha1.MemberType
		sts            *appsv1.StatefulSet
		expect         func(suspeded bool, err error)
		expectResource func(cluster v1alpha1.Cluster, s *suspender)
	}{
		"no need to suspend": {
			setup: func(cluster v1alpha1.Cluster) {
				tc := cluster.(*v1alpha1.TidbCluster)
				tc.Spec.TiKV = &v1alpha1.TiKVSpec{}
				tc.Status.TiKV = v1alpha1.TiKVStatus{}
				tc.Spec.TiKV.SuspendAction = &v1alpha1.SuspendAction{SuspendStatefulSet: false}
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			component: v1alpha1.TiKVMemberType,
			sts:       nil,
			expect: func(suspeded bool, err error) {
				g.Expect(suspeded).To(BeFalse())
				g.Expect(err).To(BeNil())
			},
			expectResource: func(cluster v1alpha1.Cluster, s *suspender) {},
		},
		"can not suspend": {
			setup: func(cluster v1alpha1.Cluster) {
				tc := cluster.(*v1alpha1.TidbCluster)
				tc.Spec.TiKV = &v1alpha1.TiKVSpec{}
				tc.Status.TiKV = v1alpha1.TiKVStatus{}
				tc.Spec.TiKV.SuspendAction = &v1alpha1.SuspendAction{SuspendStatefulSet: true}
				// can not suspend
				tc.Spec.TiDB = &v1alpha1.TiDBSpec{}
				tc.Spec.TiDB.SuspendAction = &v1alpha1.SuspendAction{SuspendStatefulSet: true}
				tc.Status.TiDB.Phase = v1alpha1.NormalPhase
			},
			component: v1alpha1.TiKVMemberType,
			sts:       nil,
			expect: func(suspeded bool, err error) {
				g.Expect(suspeded).To(BeFalse())
				g.Expect(err).To(BeNil())
			},
			expectResource: func(cluster v1alpha1.Cluster, s *suspender) {},
		},
		"begin to suspend": {
			setup: func(cluster v1alpha1.Cluster) {
				tc := cluster.(*v1alpha1.TidbCluster)
				tc.Spec.TiKV = &v1alpha1.TiKVSpec{}
				tc.Status.TiKV = v1alpha1.TiKVStatus{}
				tc.Spec.TiKV.SuspendAction = &v1alpha1.SuspendAction{SuspendStatefulSet: true}
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			component: v1alpha1.TiKVMemberType,
			sts:       nil,
			expect: func(suspeded bool, err error) {
				g.Expect(suspeded).To(BeTrue())
				g.Expect(err).To(BeNil())
			},
			expectResource: func(cluster v1alpha1.Cluster, s *suspender) {
				tc := cluster.(*v1alpha1.TidbCluster)
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.SuspendPhase))
			},
		},
		"end to suspend": {
			setup: func(cluster v1alpha1.Cluster) {
				tc := cluster.(*v1alpha1.TidbCluster)
				tc.Spec.TiKV = &v1alpha1.TiKVSpec{}
				tc.Status.TiKV = v1alpha1.TiKVStatus{}
				tc.Spec.TiKV.SuspendAction = &v1alpha1.SuspendAction{SuspendStatefulSet: false}
				tc.Status.TiKV.Phase = v1alpha1.SuspendPhase
			},
			component: v1alpha1.TiKVMemberType,
			sts:       nil,
			expect: func(suspeded bool, err error) {
				g.Expect(suspeded).To(BeTrue())
				g.Expect(err).To(BeNil())
			},
			expectResource: func(cluster v1alpha1.Cluster, s *suspender) {
				tc := cluster.(*v1alpha1.TidbCluster)
				g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.NormalPhase))
			},
		},
		"delete sts if suspend sts": {
			setup: func(cluster v1alpha1.Cluster) {
				tc := cluster.(*v1alpha1.TidbCluster)
				tc.Spec.TiKV = &v1alpha1.TiKVSpec{}
				tc.Status.TiKV = v1alpha1.TiKVStatus{}
				tc.Spec.TiKV.SuspendAction = &v1alpha1.SuspendAction{SuspendStatefulSet: true}
				tc.Status.TiKV.Phase = v1alpha1.SuspendPhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet = &appsv1.StatefulSetStatus{}
			},
			component: v1alpha1.TiKVMemberType,
			sts: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "test-cluster-tikv", Namespace: "test-namespace"},
			},
			expect: func(suspeded bool, err error) {
				g.Expect(suspeded).To(BeTrue())
				g.Expect(err).To(BeNil())
			},
			expectResource: func(cluster v1alpha1.Cluster, s *suspender) {
				tc := cluster.(*v1alpha1.TidbCluster)
				g.Expect(tc.Status.TiKV.Synced).To(BeTrue())        // not changed status
				g.Expect(tc.Status.TiKV.StatefulSet).NotTo(BeNil()) // not changed status

				_, err := s.deps.KubeClientset.AppsV1().StatefulSets(tc.Namespace).Get(context.Background(), "test-cluster-tikv", metav1.GetOptions{})
				g.Expect(err).To(HaveOccurred())
				g.Expect(errors.IsNotFound(err)).Should(BeTrue()) // sts should be deleted
			},
		},
		"clear status if suspend sts": {
			setup: func(cluster v1alpha1.Cluster) {
				tc := cluster.(*v1alpha1.TidbCluster)
				tc.Spec.TiKV = &v1alpha1.TiKVSpec{}
				tc.Status.TiKV = v1alpha1.TiKVStatus{}
				tc.Spec.TiKV.SuspendAction = &v1alpha1.SuspendAction{SuspendStatefulSet: true}
				tc.Status.TiKV.Phase = v1alpha1.SuspendPhase
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet = &appsv1.StatefulSetStatus{}
			},
			component: v1alpha1.TiKVMemberType,
			sts:       nil,
			expect: func(suspeded bool, err error) {
				g.Expect(suspeded).To(BeTrue())
				g.Expect(err).To(BeNil())
			},
			expectResource: func(cluster v1alpha1.Cluster, s *suspender) {
				tc := cluster.(*v1alpha1.TidbCluster)
				g.Expect(tc.Status.TiKV.Synced).To(BeFalse())    // clear status
				g.Expect(tc.Status.TiKV.StatefulSet).To(BeNil()) // clear status
			},
		},
	}

	for name, c := range cases {
		t.Logf("test case: %s\n", name)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		fakeDeps := controller.NewFakeDependencies()
		informerFactory := fakeDeps.KubeInformerFactory

		s := &suspender{
			deps: fakeDeps,
		}

		tc := &v1alpha1.TidbCluster{}
		tc.Name = "test-cluster"
		tc.Namespace = "test-namespace"
		c.setup(tc)

		if c.sts != nil {
			fakeDeps.KubeClientset.AppsV1().StatefulSets(c.sts.Namespace).Create(context.TODO(), c.sts, metav1.CreateOptions{})
		}
		informerFactory.Start(ctx.Done())
		informerFactory.WaitForCacheSync(ctx.Done())

		suspended, err := s.SuspendComponent(tc, c.component)
		c.expect(suspended, err)
		if c.expectResource != nil {
			c.expectResource(tc, s)
		}
	}
}

func TestSuspendSts(t *testing.T) {
	g := NewGomegaWithT(t)

	cases := map[string]struct {
		setup     func(cluster v1alpha1.Cluster)
		component v1alpha1.MemberType
		sts       *appsv1.StatefulSet
		expect    func(err error, ctx *suspendComponentCtx, s *suspender)
	}{
		"delete sts if sts is existing": {
			setup: func(cluster v1alpha1.Cluster) {
				tc := cluster.(*v1alpha1.TidbCluster)
				tc.Spec.TiKV = &v1alpha1.TiKVSpec{}
				tc.Status.TiKV = v1alpha1.TiKVStatus{}

				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet = &appsv1.StatefulSetStatus{}
			},
			component: v1alpha1.TiKVMemberType,
			sts: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "test-cluster-tikv", Namespace: "test-namespace"},
			},
			expect: func(err error, ctx *suspendComponentCtx, s *suspender) {
				tc := ctx.cluster.(*v1alpha1.TidbCluster)

				g.Expect(err).To(BeNil())
				g.Expect(tc.Status.TiKV.Synced).To(BeTrue())        // not changed status
				g.Expect(tc.Status.TiKV.StatefulSet).NotTo(BeNil()) // not changed status

				_, err = s.deps.KubeClientset.AppsV1().StatefulSets(tc.Namespace).Get(context.Background(), "test-cluster-tikv", metav1.GetOptions{})
				g.Expect(err).To(HaveOccurred())
				g.Expect(errors.IsNotFound(err)).Should(BeTrue()) // sts should be deleted
			},
		},
		"clear status if sts is not existing": {
			setup: func(cluster v1alpha1.Cluster) {
				tc := cluster.(*v1alpha1.TidbCluster)
				tc.Spec.TiKV = &v1alpha1.TiKVSpec{}
				tc.Status.TiKV = v1alpha1.TiKVStatus{}

				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.StatefulSet = &appsv1.StatefulSetStatus{}
			},
			component: v1alpha1.TiKVMemberType,
			sts:       nil,
			expect: func(err error, ctx *suspendComponentCtx, s *suspender) {
				tc := ctx.cluster.(*v1alpha1.TidbCluster)

				g.Expect(err).To(BeNil())
				g.Expect(tc.Status.TiKV.Synced).To(BeFalse())    // clear status
				g.Expect(tc.Status.TiKV.StatefulSet).To(BeNil()) // clear status
			},
		},
	}

	for name, c := range cases {
		t.Logf("test case: %s\n", name)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		fakeDeps := controller.NewFakeDependencies()
		informerFactory := fakeDeps.KubeInformerFactory

		s := &suspender{
			deps: fakeDeps,
		}

		tc := &v1alpha1.TidbCluster{}
		tc.Name = "test-cluster"
		tc.Namespace = "test-namespace"
		c.setup(tc)
		scctx := &suspendComponentCtx{
			cluster:   tc,
			component: c.component,
			spec:      tc.ComponentSpec(c.component),
			status:    tc.ComponentStatus(c.component),
		}

		if c.sts != nil {
			fakeDeps.KubeClientset.AppsV1().StatefulSets(c.sts.Namespace).Create(context.TODO(), c.sts, metav1.CreateOptions{})
		}
		informerFactory.Start(ctx.Done())
		informerFactory.WaitForCacheSync(ctx.Done())

		err := s.suspendSts(scctx)
		c.expect(err, scctx, s)
	}
}

func TestNeedsSuspendComponent(t *testing.T) {
	g := NewGomegaWithT(t)

	cases := map[string]struct {
		setup     func(tc *v1alpha1.TidbCluster)
		component v1alpha1.MemberType
		expect    func(need bool)
	}{
		"suspend statefuleset": {
			setup: func(tc *v1alpha1.TidbCluster) {
			},
			component: v1alpha1.TiKVMemberType,
			expect: func(need bool) {
				g.Expect(need).To(BeTrue())
			},
		},
		"not set spec": {
			setup: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiKV = nil
			},
			component: v1alpha1.TiKVMemberType,
			expect: func(need bool) {
				g.Expect(need).To(BeFalse())
			},
		},
		"not set suspendAction": {
			setup: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.SuspendAction = nil
			},
			component: v1alpha1.TiKVMemberType,
			expect: func(need bool) {
				g.Expect(need).To(BeFalse())
			},
		},
		"not suspend any resource": {
			setup: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.SuspendAction.SuspendStatefulSet = false
			},
			component: v1alpha1.TiKVMemberType,
			expect: func(need bool) {
				g.Expect(need).To(BeFalse())
			},
		},
	}

	for name, c := range cases {
		t.Logf("test case: %s\n", name)

		tc := &v1alpha1.TidbCluster{}
		tc.Name = "test-cluster"
		tc.Namespace = "test-namespace"
		tc.Spec.SuspendAction = &v1alpha1.SuspendAction{SuspendStatefulSet: true}
		tc.Spec.PD = &v1alpha1.PDSpec{}
		tc.Spec.TiKV = &v1alpha1.TiKVSpec{}
		tc.Spec.TiDB = &v1alpha1.TiDBSpec{}
		tc.Spec.TiFlash = &v1alpha1.TiFlashSpec{}
		tc.Spec.Pump = &v1alpha1.PumpSpec{}
		tc.Spec.TiCDC = &v1alpha1.TiCDCSpec{}

		c.setup(tc)

		needs := needsSuspendComponent(tc, c.component)
		c.expect(needs)
	}
}

func TestCanSuspendComponent(t *testing.T) {
	g := NewGomegaWithT(t)

	cases := map[string]struct {
		setup     func(tc *v1alpha1.TidbCluster)
		component v1alpha1.MemberType
		expect    func(can bool, reason string)
	}{
		"can suspend component when phase is Normal": {
			setup: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase

				// set prior component to be suspended
				tc.Status.TiDB.Phase = v1alpha1.SuspendPhase
				tc.Status.TiDB.StatefulSet = nil
				tc.Status.TiFlash.Phase = v1alpha1.SuspendPhase
				tc.Status.TiFlash.StatefulSet = nil
				tc.Status.TiCDC.Phase = v1alpha1.SuspendPhase
				tc.Status.TiCDC.StatefulSet = nil
			},
			component: v1alpha1.TiKVMemberType,
			expect: func(can bool, reason string) {
				g.Expect(can).To(BeTrue())
				g.Expect(reason).To(BeEmpty())
			},
		},
		"can suspend component when phase is Suspend": {
			setup: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Phase = v1alpha1.SuspendPhase

				// set prior component to be suspended
				tc.Status.TiDB.Phase = v1alpha1.SuspendPhase
				tc.Status.TiDB.StatefulSet = nil
				tc.Status.TiFlash.Phase = v1alpha1.SuspendPhase
				tc.Status.TiFlash.StatefulSet = nil
				tc.Status.TiCDC.Phase = v1alpha1.SuspendPhase
				tc.Status.TiCDC.StatefulSet = nil
			},
			component: v1alpha1.TiKVMemberType,
			expect: func(can bool, reason string) {
				g.Expect(can).To(BeTrue())
				g.Expect(reason).To(BeEmpty())
			},
		},
		"component's phase is not Normal and Suspend": {
			setup: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
			},
			component: v1alpha1.TiKVMemberType,
			expect: func(can bool, reason string) {
				g.Expect(can).To(BeFalse())
				g.Expect(reason).To(Equal("component phase is not Normal or Suspend"))
			},
		},
		"wait for other components to be suspended": {
			setup: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase

				tc.Status.TiDB.Phase = v1alpha1.SuspendPhase
				tc.Status.TiDB.StatefulSet = nil
				tc.Status.TiFlash.Phase = v1alpha1.SuspendPhase
				tc.Status.TiFlash.StatefulSet = &appsv1.StatefulSetStatus{} // suspending
				tc.Status.TiCDC.Phase = v1alpha1.SuspendPhase
				tc.Status.TiCDC.StatefulSet = nil
			},
			component: v1alpha1.TiKVMemberType,
			expect: func(can bool, reason string) {
				g.Expect(can).To(BeFalse())
				g.Expect(reason).To(Equal("wait another component tiflash to be suspended"))
			},
		},
		"don't need to wait for other components to be suspended": {
			setup: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase

				tc.Status.TiDB.Phase = v1alpha1.SuspendPhase
				tc.Status.TiDB.StatefulSet = nil
				tc.Status.TiFlash.Phase = v1alpha1.SuspendPhase
				tc.Status.TiFlash.StatefulSet = &appsv1.StatefulSetStatus{} // suspending
				tc.Status.TiCDC.Phase = v1alpha1.SuspendPhase
				tc.Status.TiCDC.StatefulSet = nil

				tc.Spec.TiFlash.SuspendAction = &v1alpha1.SuspendAction{SuspendStatefulSet: false} // not need
			},
			component: v1alpha1.TiKVMemberType,
			expect: func(can bool, reason string) {
				g.Expect(can).To(BeTrue())
				g.Expect(reason).To(BeEmpty())
			},
		},
	}

	for name, c := range cases {
		t.Logf("test case: %s\n", name)

		tc := &v1alpha1.TidbCluster{}
		tc.Name = "test-cluster"
		tc.Namespace = "test-namespace"
		tc.Spec.SuspendAction = &v1alpha1.SuspendAction{SuspendStatefulSet: true}
		tc.Spec.PD = &v1alpha1.PDSpec{}
		tc.Spec.TiKV = &v1alpha1.TiKVSpec{}
		tc.Spec.TiDB = &v1alpha1.TiDBSpec{}
		tc.Spec.TiFlash = &v1alpha1.TiFlashSpec{}
		tc.Spec.Pump = &v1alpha1.PumpSpec{}
		tc.Spec.TiCDC = &v1alpha1.TiCDCSpec{}

		c.setup(tc)

		can, reason := canSuspendComponent(tc, c.component)
		c.expect(can, reason)
	}
}
