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
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestTiKVFailoverFailover(t *testing.T) {
	tests := []struct {
		name     string
		update   func(*v1alpha1.TidbCluster)
		err      bool
		expectFn func(t *testing.T, tc *v1alpha1.TidbCluster)
	}{
		{
			name: "normal",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"1": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-1",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"2": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-2",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-61 * time.Minute)},
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(2))
			},
		},
		{
			name: "tikv state is not Down",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"1": {State: v1alpha1.TiKVStateUp, PodName: "tikv-1"},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(0))
			},
		},
		{
			name: "deadline not exceed",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"1": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-1",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-30 * time.Minute)},
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(0))
			},
		},
		{
			name: "lastTransitionTime is zero",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"1": {
						State:   v1alpha1.TiKVStateDown,
						PodName: "tikv-1",
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(0))
			},
		},
		{
			name: "exist in failureStores",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"1": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-1",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
				}
				tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{
					"1": {
						PodName: "tikv-1",
						StoreID: "1",
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(1))
			},
		},
		{
			name: "not exceed max failover count",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"3": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-0",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"4": {
						State:              v1alpha1.TiKVStateUp,
						PodName:            "tikv-4",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"5": {
						State:              v1alpha1.TiKVStateUp,
						PodName:            "tikv-5",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-61 * time.Minute)},
					},
				}
				tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{
					"1": {
						PodName: "tikv-1",
						StoreID: "1",
					},
					"2": {
						PodName: "tikv-2",
						StoreID: "2",
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(3))
			},
		},
		{
			name: "exceed max failover count1",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"3": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-3",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"4": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-4",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"5": {
						State:              v1alpha1.TiKVStateUp,
						PodName:            "tikv-5",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-61 * time.Minute)},
					},
				}
				tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{
					"1": {
						PodName: "tikv-1",
						StoreID: "1",
					},
					"2": {
						PodName: "tikv-2",
						StoreID: "2",
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(3))
			},
		},
		{
			name: "exceed max failover count2",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"0": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-0",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"4": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-4",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-61 * time.Minute)},
					},
					"5": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-5",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
				}
				tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{
					"1": {
						PodName: "tikv-1",
						StoreID: "1",
					},
					"2": {
						PodName: "tikv-2",
						StoreID: "2",
					},
					"3": {
						PodName: "tikv-3",
						StoreID: "3",
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(3))
			},
		},
		{
			name: "exceed max failover count2 but maxFailoverCount = 0",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiKV.MaxFailoverCount = pointer.Int32Ptr(0)
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"12": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-12",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"13": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-13",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-61 * time.Minute)},
					},
					"14": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-14",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
				}
				tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{
					"1": {
						PodName: "tikv-1",
						StoreID: "1",
					},
					"2": {
						PodName: "tikv-2",
						StoreID: "2",
					},
					"3": {
						PodName: "tikv-3",
						StoreID: "3",
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(3))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			tc := newTidbClusterForPD()
			tc.Spec.TiKV.Replicas = 6
			tc.Spec.TiKV.MaxFailoverCount = pointer.Int32Ptr(3)
			tt.update(tc)
			tikvFailover := newFakeTiKVFailover()

			err := tikvFailover.Failover(tc)
			if tt.err {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			tt.expectFn(t, tc)
		})
	}
}

func newFakeTiKVFailover() *tikvFailover {
	return &tikvFailover{deps: controller.NewFakeDependencies()}
}
