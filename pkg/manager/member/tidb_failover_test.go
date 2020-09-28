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

package member

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTiDBFailoverFailover(t *testing.T) {
	tests := []struct {
		name        string
		pods        []*corev1.Pod
		update      func(*v1alpha1.TidbCluster)
		errExpectFn func(*GomegaWithT, error)
		expectFn    func(*GomegaWithT, *v1alpha1.TidbCluster)
	}{
		{
			name: "all tidb members are ready",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: true,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
				}
			},
			errExpectFn: func(t *GomegaWithT, err error) {
				t.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(t *GomegaWithT, tc *v1alpha1.TidbCluster) {
				t.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
				t.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(2))
			},
		},
		{
			name: "one tidb member failed",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "failover-tidb-0",
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "failover-tidb-1",
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: false,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
				}
			},
			errExpectFn: func(t *GomegaWithT, err error) {
				t.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(t *GomegaWithT, tc *v1alpha1.TidbCluster) {
				t.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(1))
				t.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(2))
			},
		},
		{
			name: "one tidb member failed but not scheduled yet",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "failover-tidb-0",
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionUnknown,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "failover-tidb-1",
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: false,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
				}
			},
			errExpectFn: func(t *GomegaWithT, err error) {
				t.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(t *GomegaWithT, tc *v1alpha1.TidbCluster) {
				t.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
				t.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(2))
			},
		},
		{
			name: "two tidb members failed",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "failover-tidb-0",
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: corev1.NamespaceDefault,
						Name:      "failover-tidb-1",
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: false,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: false,
					},
				}
			},
			errExpectFn: func(t *GomegaWithT, err error) {
				t.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(t *GomegaWithT, tc *v1alpha1.TidbCluster) {
				t.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(1))
				t.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(2))
			},
		},
		{
			name: "max failover count",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: false,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: false,
					},
					"failover-tidb-2": {
						Name:   "failover-tidb-2",
						Health: false,
					},
					"failover-tidb-3": {
						Name:   "failover-tidb-3",
						Health: false,
					},
					"failover-tidb-4": {
						Name:   "failover-tidb-4",
						Health: false,
					},
				}
				tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{
					"failover-tidb-0": {
						PodName: "failover-tidb-0",
					},
					"failover-tidb-1": {
						PodName: "failover-tidb-1",
					},
					"failover-tidb-2": {
						PodName: "failover-tidb-2",
					},
				}
			},
			errExpectFn: func(t *GomegaWithT, err error) {
				t.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(t *GomegaWithT, tc *v1alpha1.TidbCluster) {
				t.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(3))
				t.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(2))
			},
		},
		{
			name: "max failover count but maxFailoverCount = 0",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiDB.MaxFailoverCount = pointer.Int32Ptr(0)
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: false,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: false,
					},
					"failover-tidb-2": {
						Name:   "failover-tidb-2",
						Health: false,
					},
					"failover-tidb-3": {
						Name:   "failover-tidb-3",
						Health: false,
					},
					"failover-tidb-4": {
						Name:   "failover-tidb-4",
						Health: false,
					},
				}
				tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{
					"failover-tidb-0": {
						PodName: "failover-tidb-0",
					},
					"failover-tidb-1": {
						PodName: "failover-tidb-1",
					},
					"failover-tidb-2": {
						PodName: "failover-tidb-2",
					},
				}
			},
			errExpectFn: func(t *GomegaWithT, err error) {
				t.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(t *GomegaWithT, tc *v1alpha1.TidbCluster) {
				t.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(3))
				t.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(2))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fakeDeps := controller.NewFakeDependencies()
			fakeDeps.KubeInformerFactory.Start(ctx.Done())
			fakeDeps.KubeInformerFactory.WaitForCacheSync(ctx.Done())

			for _, pod := range test.pods {
				fakeDeps.KubeClientset.CoreV1().Pods(pod.Namespace).Create(pod)
			}
			tidbFailover := NewTiDBFailover(fakeDeps)
			tc := newTidbClusterForTiDBFailover()
			test.update(tc)
			err := tidbFailover.Failover(tc)
			test.errExpectFn(g, err)
			test.expectFn(g, tc)
		})
	}
}

func TestTiDBFailoverRecover(t *testing.T) {
	tests := []struct {
		name     string
		update   func(*v1alpha1.TidbCluster)
		expectFn func(*GomegaWithT, *v1alpha1.TidbCluster)
	}{
		{
			name: "have not failure tidb member to recover",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: true,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(2))
				g.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "one failure tidb member to recover",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiDB.Replicas = 3
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: true,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
					"failover-tidb-2": {
						Name:   "failover-tidb-2",
						Health: true,
					},
				}
				tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{
					"failover-tidb-0": {
						PodName: "failover-tidb-0",
					},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(3))
				g.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure tidb members to recover",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiDB.Replicas = 4
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: true,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
					"failover-tidb-2": {
						Name:   "failover-tidb-2",
						Health: true,
					},
					"failover-tidb-3": {
						Name:   "failover-tidb-3",
						Health: true,
					},
				}
				tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{
					"failover-tidb-0": {
						PodName: "failover-tidb-0",
					},
					"failover-tidb-1": {
						PodName: "failover-tidb-1",
					},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(4))
				g.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure tidb members to recover and user have set a larger replicas",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiDB.Replicas = 5
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: true,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
					"failover-tidb-2": {
						Name:   "failover-tidb-2",
						Health: true,
					},
					"failover-tidb-3": {
						Name:   "failover-tidb-3",
						Health: true,
					},
				}
				tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{
					"failover-tidb-0": {
						PodName: "failover-tidb-0",
					},
					"failover-tidb-1": {
						PodName: "failover-tidb-1",
					},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(5))
				g.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure tidb members to recover and user have set a smaller replicas",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiDB.Replicas = 1
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: true,
					},
				}
				tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{
					"failover-tidb-0": {
						PodName: "failover-tidb-0",
					},
					"failover-tidb-1": {
						PodName: "failover-tidb-1",
					},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(1))
				g.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fakeDeps := controller.NewFakeDependencies()
			fakeDeps.KubeInformerFactory.Start(ctx.Done())
			fakeDeps.KubeInformerFactory.WaitForCacheSync(ctx.Done())
			tidbFailover := NewTiDBFailover(fakeDeps)
			tc := newTidbClusterForTiDBFailover()
			test.update(tc)
			tidbFailover.Recover(tc)
			test.expectFn(g, tc)
		})
	}
}

func newTidbClusterForTiDBFailover() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failover",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("failover"),
		},
		Spec: v1alpha1.TidbClusterSpec{
			TiDB: &v1alpha1.TiDBSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tidb-test-image",
				},
				Replicas:         2,
				MaxFailoverCount: pointer.Int32Ptr(3),
			},
		},
	}
}
