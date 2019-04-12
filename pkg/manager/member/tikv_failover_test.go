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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTiKVFailoverFailover(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		update   func(*v1alpha1.TidbCluster)
		err      bool
		expectFn func(*v1alpha1.TidbCluster)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := newTidbClusterForPD()
		test.update(tc)
		tikvFailover := newFakeTiKVFailover()

		err := tikvFailover.Failover(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		test.expectFn(tc)
	}

	tests := []testcase{
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
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiKV.Replicas)).To(Equal(3))
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
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiKV.Replicas)).To(Equal(3))
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
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiKV.Replicas)).To(Equal(3))
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
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiKV.Replicas)).To(Equal(3))
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
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiKV.Replicas)).To(Equal(3))
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(1))
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakeTiKVFailover() *tikvFailover {
	return &tikvFailover{1 * time.Hour}
}
