// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func TestCondObjectIsDeleting(t *testing.T) {
	cases := []struct {
		desc         string
		state        ObjectState[*v1alpha1.PD]
		expectedCond bool
	}{
		{
			desc: "cond is false",
			state: newFakeObjectState(
				fake.FakeObj[v1alpha1.PD]("test"),
			),
		},
		{
			desc: "cond is true",
			state: newFakeObjectState(
				fake.FakeObj("test", fake.DeleteNow[v1alpha1.PD]()),
			),
			expectedCond: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			cond := CondObjectIsDeleting[scope.PD](c.state)
			assert.Equal(tt, c.expectedCond, cond.Satisfy(), c.desc)
		})
	}
}

func TestCondObjectHasBeenDeleted(t *testing.T) {
	cases := []struct {
		desc         string
		state        ObjectState[*v1alpha1.PD]
		expectedCond bool
	}{
		{
			desc: "cond is false",
			state: newFakeObjectState(
				fake.FakeObj[v1alpha1.PD]("test"),
			),
		},
		{
			desc:         "cond is true",
			state:        newFakeObjectState[*v1alpha1.PD](nil),
			expectedCond: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			cond := CondObjectHasBeenDeleted[scope.PD](c.state)
			assert.Equal(tt, c.expectedCond, cond.Satisfy(), c.desc)
		})
	}
}

func TestCondClusterIsSuspending(t *testing.T) {
	cases := []struct {
		desc         string
		state        *fakeState[v1alpha1.Cluster]
		expectedCond bool
	}{
		{
			desc: "cond is false",
			state: &fakeState[v1alpha1.Cluster]{
				obj: fake.FakeObj[v1alpha1.Cluster]("test"),
			},
		},
		{
			desc: "cond is true",
			state: &fakeState[v1alpha1.Cluster]{
				obj: fake.FakeObj("test", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
					obj.Spec.SuspendAction = &v1alpha1.SuspendAction{
						SuspendCompute: true,
					}
					return obj
				}),
			},
			expectedCond: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			s := &fakeClusterState{s: c.state}
			cond := CondClusterIsSuspending(s)
			assert.Equal(tt, c.expectedCond, cond.Satisfy(), c.desc)
		})
	}
}

func TestCondClusterIsPaused(t *testing.T) {
	cases := []struct {
		desc         string
		state        *fakeState[v1alpha1.Cluster]
		expectedCond bool
	}{
		{
			desc: "cond is false",
			state: &fakeState[v1alpha1.Cluster]{
				obj: fake.FakeObj[v1alpha1.Cluster]("test"),
			},
		},
		{
			desc: "cond is true",
			state: &fakeState[v1alpha1.Cluster]{
				obj: fake.FakeObj("test", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
					obj.Spec.Paused = true
					return obj
				}),
			},
			expectedCond: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			s := &fakeClusterState{s: c.state}
			cond := CondClusterIsPaused(s)
			assert.Equal(tt, c.expectedCond, cond.Satisfy(), c.desc)
		})
	}
}

func TestCondClusterPDAddrIsNotRegistered(t *testing.T) {
	cases := []struct {
		desc         string
		state        *fakeState[v1alpha1.Cluster]
		expectedCond bool
	}{
		{
			desc: "cond is true",
			state: &fakeState[v1alpha1.Cluster]{
				obj: fake.FakeObj[v1alpha1.Cluster]("test"),
			},
			expectedCond: true,
		},
		{
			desc: "cond is false",
			state: &fakeState[v1alpha1.Cluster]{
				obj: fake.FakeObj("test", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
					obj.Status.PD = "http://pd:2379"
					return obj
				}),
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			s := &fakeClusterState{s: c.state}
			cond := CondClusterPDAddrIsNotRegistered(s)
			assert.Equal(tt, c.expectedCond, cond.Satisfy(), c.desc)
		})
	}
}

func TestCondFeatureGatesIsNotSynced(t *testing.T) {
	type state struct {
		ClusterStateFunc
		ObjectStateFunc[*v1alpha1.PDGroup]
	}

	cases := []struct {
		desc         string
		state        *state
		expectedCond bool
	}{
		{
			desc: "synced, cond is false",
			state: &state{
				ClusterStateFunc: func() *v1alpha1.Cluster {
					return &v1alpha1.Cluster{
						Spec: v1alpha1.ClusterSpec{
							FeatureGates: []metav1alpha1.FeatureGate{
								{
									Name: "xxx",
								},
							},
						},
					}
				},
				ObjectStateFunc: func() *v1alpha1.PDGroup {
					return &v1alpha1.PDGroup{
						Spec: v1alpha1.PDGroupSpec{
							Features: []metav1alpha1.Feature{
								"xxx",
							},
						},
					}
				},
			},
			expectedCond: false,
		},
		{
			desc: "not synced, cond is true",
			state: &state{
				ClusterStateFunc: func() *v1alpha1.Cluster {
					return &v1alpha1.Cluster{
						Spec: v1alpha1.ClusterSpec{
							FeatureGates: []metav1alpha1.FeatureGate{
								{
									Name: "xxx",
								},
							},
						},
					}
				},
				ObjectStateFunc: func() *v1alpha1.PDGroup {
					return &v1alpha1.PDGroup{
						Spec: v1alpha1.PDGroupSpec{
							Features: []metav1alpha1.Feature{},
						},
					}
				},
			},
			expectedCond: true,
		},
		{
			desc: "unordered, cond is true",
			state: &state{
				ClusterStateFunc: func() *v1alpha1.Cluster {
					return &v1alpha1.Cluster{
						Spec: v1alpha1.ClusterSpec{
							FeatureGates: []metav1alpha1.FeatureGate{
								{
									Name: "xxx",
								},
								{
									Name: "yyy",
								},
							},
						},
					}
				},
				ObjectStateFunc: func() *v1alpha1.PDGroup {
					return &v1alpha1.PDGroup{
						Spec: v1alpha1.PDGroupSpec{
							Features: []metav1alpha1.Feature{
								"yyy",
								"xxx",
							},
						},
					}
				},
			},
			expectedCond: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			cond := CondFeatureGatesIsNotSynced[scope.PDGroup](c.state)
			assert.Equal(tt, c.expectedCond, cond.Satisfy(), c.desc)
		})
	}
}

func TestCondObjectIsNotDeletingButOfflined(t *testing.T) {
	cases := []struct {
		desc         string
		state        ObjectState[*v1alpha1.TiKV]
		expectedCond bool
	}{
		{
			desc: "cond is false, not offlined",
			state: newFakeObjectState(
				fake.FakeObj[v1alpha1.TiKV]("test"),
			),
		},
		{
			desc: "cond is false, deleting",
			state: newFakeObjectState(
				fake.FakeObj("test", fake.DeleteNow[v1alpha1.TiKV]()),
			),
		},
		{
			desc: "cond is true",
			state: newFakeObjectState(
				fake.FakeObj("test", fake.DeleteNow[v1alpha1.TiKV](), func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
					obj.Status.Conditions = []metav1.Condition{
						{
							Type:   v1alpha1.StoreOfflinedConditionType,
							Status: metav1.ConditionTrue,
						},
					}
					return obj
				}),
			),
			expectedCond: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			cond := CondObjectIsNotDeletingButOfflined[scope.TiKV](c.state)
			assert.Equal(tt, c.expectedCond, cond.Satisfy(), c.desc)
		})
	}
}
