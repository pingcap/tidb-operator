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
	"time"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestStatefulSetIsUpgrading(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name            string
		update          func(*apps.StatefulSet)
		expectUpgrading bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		set := &apps.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: metav1.NamespaceDefault,
			},
		}
		if test.update != nil {
			test.update(set)
		}
		b := statefulSetIsUpgrading(set)
		if test.expectUpgrading {
			g.Expect(b).To(BeTrue())
		} else {
			g.Expect(b).To(BeFalse())
		}
	}
	tests := []*testcase{
		{
			name:            "ObservedGeneration is nil",
			update:          nil,
			expectUpgrading: false,
		},
		{
			name: "CurrentRevision not equal UpdateRevision",
			update: func(set *apps.StatefulSet) {
				set.Status.ObservedGeneration = 1000
				set.Status.CurrentRevision = "v1"
				set.Status.UpdateRevision = "v2"
			},
			expectUpgrading: true,
		},
		{
			name: "set.Generation > *set.Status.ObservedGeneration && *set.Spec.Replicas == set.Status.Replicas",
			update: func(set *apps.StatefulSet) {
				set.Generation = 1001
				set.Status.ObservedGeneration = 1000
				set.Status.CurrentRevision = "v1"
				set.Status.UpdateRevision = "v1"
				set.Status.Replicas = 3
				set.Spec.Replicas = func() *int32 { var i int32 = 3; return &i }()
			},
			expectUpgrading: true,
		},
		{
			name: "replicas not equal",
			update: func(set *apps.StatefulSet) {
				set.Generation = 1001
				set.Status.ObservedGeneration = 1000
				set.Status.CurrentRevision = "v1"
				set.Status.UpdateRevision = "v1"
				set.Status.Replicas = 3
				set.Spec.Replicas = func() *int32 { var i int32 = 2; return &i }()
			},
			expectUpgrading: false,
		},
	}

	for _, test := range tests {
		testFn(test, t)
	}
}

func TestGetStsAnnotations(t *testing.T) {
	tests := []struct {
		name      string
		tc        *v1alpha1.TidbCluster
		component string
		expected  map[string]string
	}{
		{
			name: "nil",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: nil,
				},
			},
			component: label.TiDBLabelVal,
			expected:  map[string]string{},
		},
		{
			name: "empty",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			component: label.TiDBLabelVal,
			expected:  map[string]string{},
		},
		{
			name: "tidb",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						label.AnnTiDBDeleteSlots: "[1,2]",
					},
				},
			},
			component: label.TiDBLabelVal,
			expected: map[string]string{
				helper.DeleteSlotsAnn: "[1,2]",
			},
		},
		{
			name: "tidb but component is not tidb",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						label.AnnTiDBDeleteSlots: "[1,2]",
					},
				},
			},
			component: label.PDLabelVal,
			expected:  map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStsAnnotations(tt.tc.Annotations, tt.component)
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
		})
	}
}

func TestShouldRecover(t *testing.T) {
	notReadyPods := []*v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-tikv-0",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-tikv-1",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-tiflash-0",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-tiflash-1",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		},
	}
	pods := []*v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-tikv-0",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-tikv-1",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-tiflash-0",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-tiflash-1",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
	}
	podsWithFailover := append(pods, []*v1.Pod{
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-tikv-2",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		},
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-tiflash-2",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		},
	}...)
	tests := []struct {
		name string
		tc   *v1alpha1.TidbCluster
		pods []*v1.Pod
		want bool
	}{
		{
			name: "should not recover if no failure members",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiKV: &v1alpha1.TiKVSpec{
						Replicas: 3,
					},
					TiFlash: &v1alpha1.TiFlashSpec{
						Replicas: 3,
					},
				},
				Status: v1alpha1.TidbClusterStatus{},
			},
			pods: pods,
			want: false,
		},
		{
			name: "should not recover if get pod failure",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiKV: &v1alpha1.TiKVSpec{
						Replicas: 3,
					},
					TiFlash: &v1alpha1.TiFlashSpec{
						Replicas: 3,
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					TiKV: v1alpha1.TiKVStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"1": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tikv-1",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
							"3": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tikv-0",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
						},
						FailureStores: map[string]v1alpha1.TiKVFailureStore{
							"1": {
								PodName: "failover-tikv-1",
								StoreID: "1",
							},
						},
					},
					TiFlash: v1alpha1.TiFlashStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"2": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tiflash-1",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
							"4": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tiflash-0",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
						},
						FailureStores: map[string]v1alpha1.TiKVFailureStore{
							"2": {
								PodName: "failover-tiflash-1",
								StoreID: "2",
							},
						},
					},
				},
			},
			pods: pods,
			want: false,
		},
		{
			name: "should not recover if pod not ready",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiKV: &v1alpha1.TiKVSpec{
						Replicas: 2,
					},
					TiFlash: &v1alpha1.TiFlashSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					TiKV: v1alpha1.TiKVStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"1": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tikv-1",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
							"3": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tikv-0",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
						},
						FailureStores: map[string]v1alpha1.TiKVFailureStore{
							"1": {
								PodName: "failover-tikv-1",
								StoreID: "1",
							},
						},
					},
					TiFlash: v1alpha1.TiFlashStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"2": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tiflash-1",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
							"4": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tiflash-0",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
						},
						FailureStores: map[string]v1alpha1.TiKVFailureStore{
							"2": {
								PodName: "failover-tiflash-1",
								StoreID: "2",
							},
						},
					},
				},
			},
			pods: notReadyPods,
			want: false,
		},
		{
			name: "should not recover if a member is not healthy",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiKV: &v1alpha1.TiKVSpec{
						Replicas: 2,
					},
					TiFlash: &v1alpha1.TiFlashSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					TiKV: v1alpha1.TiKVStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"1": {
								State:              v1alpha1.TiKVStateDown,
								PodName:            "failover-tikv-1",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
							"3": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tikv-0",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
						},
						FailureStores: map[string]v1alpha1.TiKVFailureStore{
							"1": {
								PodName: "failover-tikv-1",
								StoreID: "1",
							},
						},
					},
					TiFlash: v1alpha1.TiFlashStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"2": {
								State:              v1alpha1.TiKVStateDown,
								PodName:            "failover-tiflash-1",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
							"4": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tiflash-0",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
						},
						FailureStores: map[string]v1alpha1.TiKVFailureStore{
							"2": {
								PodName: "failover-tiflash-1",
								StoreID: "2",
							},
						},
					},
				},
			},
			pods: pods,
			want: false,
		},
		{
			name: "should not recover if some stores do not exist",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiKV: &v1alpha1.TiKVSpec{
						Replicas: 2,
					},
					TiFlash: &v1alpha1.TiFlashSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					TiKV: v1alpha1.TiKVStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"3": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tikv-0",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
						},
						FailureStores: map[string]v1alpha1.TiKVFailureStore{
							"1": {
								PodName: "failover-tikv-1",
								StoreID: "1",
							},
						},
					},
					TiFlash: v1alpha1.TiFlashStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"4": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tiflash-0",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
						},
						FailureStores: map[string]v1alpha1.TiKVFailureStore{
							"2": {
								PodName: "failover-tiflash-1",
								StoreID: "2",
							},
						},
					},
				},
			},
			pods: pods,
			want: false,
		},
		{
			name: "should recover if all members are ready and healthy",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiKV: &v1alpha1.TiKVSpec{
						Replicas: 2,
					},
					TiFlash: &v1alpha1.TiFlashSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					TiKV: v1alpha1.TiKVStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"1": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tikv-1",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
							"3": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tikv-0",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
						},
						FailureStores: map[string]v1alpha1.TiKVFailureStore{
							"1": {
								PodName: "failover-tikv-1",
								StoreID: "1",
							},
						},
					},
					TiFlash: v1alpha1.TiFlashStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"2": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tiflash-1",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
							"4": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tiflash-0",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
						},
						FailureStores: map[string]v1alpha1.TiKVFailureStore{
							"2": {
								PodName: "failover-tiflash-1",
								StoreID: "2",
							},
						},
					},
				},
			},
			pods: pods,
			want: true,
		},
		{
			name: "should recover if all members are ready and healthy (ignore auto-created failover pods)",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiKV: &v1alpha1.TiKVSpec{
						Replicas: 2,
					},
					TiFlash: &v1alpha1.TiFlashSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.TidbClusterStatus{
					TiKV: v1alpha1.TiKVStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"1": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tikv-1",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
							"5": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tikv-0",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
						},
						FailureStores: map[string]v1alpha1.TiKVFailureStore{
							"1": {
								PodName: "failover-tikv-1",
								StoreID: "1",
							},
							"3": {
								PodName: "failover-tikv-2",
								StoreID: "3",
							},
						},
					},
					TiFlash: v1alpha1.TiFlashStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"2": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tiflash-1",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
							"6": {
								State:              v1alpha1.TiKVStateUp,
								PodName:            "failover-tiflash-0",
								LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
							},
						},
						FailureStores: map[string]v1alpha1.TiKVFailureStore{
							"2": {
								PodName: "failover-tiflash-1",
								StoreID: "2",
							},
							"4": {
								PodName: "failover-tiflash-2",
								StoreID: "4",
							},
						},
					},
				},
			},
			pods: podsWithFailover,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := kubefake.NewSimpleClientset()
			for _, pod := range tt.pods {
				client.CoreV1().Pods(pod.Namespace).Create(pod)
			}
			kubeInformerFactory := kubeinformers.NewSharedInformerFactory(client, 0)
			podLister := kubeInformerFactory.Core().V1().Pods().Lister()
			kubeInformerFactory.Start(ctx.Done())
			kubeInformerFactory.WaitForCacheSync(ctx.Done())

			got := shouldRecover(tt.tc, label.TiFlashLabelVal, podLister)
			if got != tt.want {
				t.Fatalf("wants %v, got %v", tt.want, got)
			}
			got = shouldRecover(tt.tc, label.TiKVLabelVal, podLister)
			if got != tt.want {
				t.Fatalf("wants %v, got %v", tt.want, got)
			}
			got = shouldRecover(tt.tc, label.PDLabelVal, podLister)
			if got != false {
				t.Fatalf("wants %v, got %v", false, got)
			}
		})
	}
}

func TestTOMLEqual(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		d1    string
		d2    string
		equal bool
	}

	tests := []*testcase{
		{
			d1:    "a = 1",
			d2:    "a = 1",
			equal: true,
		},
		{
			d1:    "a = 1",
			d2:    "a = 2",
			equal: false,
		},
		{
			d1:    "a =  1",
			d2:    "a = 1",
			equal: true,
		},
		{
			d1:    "a =  1",
			d2:    "a = 2",
			equal: false,
		},
		{
			d1:    "[user]\n[user.default]\np = 'ok'",
			d2:    "[user.default]\np = 'ok'",
			equal: true,
		},
	}

	for _, test := range tests {
		equal, err := TOMLEqual([]byte(test.d1), []byte(test.d2))
		g.Expect(err).Should(BeNil())
		t.Logf("check '%s' and '%s'", test.d1, test.d2)
		g.Expect(equal).Should(Equal(test.equal))
	}
}
