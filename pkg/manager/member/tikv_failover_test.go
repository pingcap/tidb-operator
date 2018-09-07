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
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/pd/pkg/typeutil"
	"github.com/pingcap/pd/server"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

func TestTiKVFailoverFailover(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name      string
		update    func(*v1alpha1.TidbCluster)
		getCfgErr bool
		err       bool
		expectFn  func(*v1alpha1.TidbCluster)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := newTidbClusterForPD()
		test.update(tc)
		tikvFailover, fakePDControl := newFakeTiKVFailover()
		pdClient := controller.NewFakePDClient()
		fakePDControl.SetPDClient(tc, pdClient)

		pdClient.AddReaction(controller.GetConfigActionType, func(action *controller.Action) (interface{}, error) {
			if test.getCfgErr {
				return nil, fmt.Errorf("get config failed")
			} else {
				return &server.Config{
					Schedule: server.ScheduleConfig{MaxStoreDownTime: typeutil.Duration{Duration: 1 * time.Hour}},
				}, nil
			}
		})

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
			getCfgErr: false,
			err:       false,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiKV.Replicas)).To(Equal(5))
			},
		},
		{
			name:      "get config failed",
			update:    func(tc *v1alpha1.TidbCluster) {},
			getCfgErr: true,
			err:       true,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiKV.Replicas)).To(Equal(3))
			},
		},
		{
			name: "tikv state is not Down",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"1": {State: v1alpha1.TiKVStateUp, PodName: "tikv-1"},
				}
			},
			getCfgErr: false,
			err:       false,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiKV.Replicas)).To(Equal(3))
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
			getCfgErr: false,
			err:       false,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiKV.Replicas)).To(Equal(3))
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
			getCfgErr: false,
			err:       false,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiKV.Replicas)).To(Equal(3))
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
			getCfgErr: false,
			err:       false,
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiKV.Replicas)).To(Equal(3))
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTiKVFailoverRecover(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		expectFn func(*v1alpha1.TidbCluster)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := newTidbClusterForPD()
		tikvFailover, _ := newFakeTiKVFailover()
		tikvFailover.Recover(tc)
		test.expectFn(tc)
	}
	tests := []testcase{
		{
			name: "normal",
			expectFn: func(tc *v1alpha1.TidbCluster) {
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(0))
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakeTiKVFailover() (*tikvFailover, *controller.FakePDControl) {
	pdControl := controller.NewFakePDControl()
	return &tikvFailover{pdControl}, pdControl
}
