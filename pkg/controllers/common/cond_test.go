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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func TestCondGroupHasBeenDeleted(t *testing.T) {
	t.Run("PDGroup", testCondGroupHasBeenDeleted[runtime.PDGroup])
	t.Run("TiDBGroup", testCondGroupHasBeenDeleted[runtime.TiDBGroup])
	t.Run("TiKVGroup", testCondGroupHasBeenDeleted[runtime.TiKVGroup])
	t.Run("TiFlashGroup", testCondGroupHasBeenDeleted[runtime.TiFlashGroup])
}

func testCondGroupHasBeenDeleted[
	G runtime.GroupSet,
	RG runtime.GroupT[G],
](t *testing.T) {
	cases := []struct {
		desc         string
		state        GroupState[RG]
		expectedCond bool
	}{
		{
			desc: "cond is false",
			state: FakeGroupState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("test")
					return obj
				}),
			),
		},
		{
			desc:         "cond is true",
			state:        FakeGroupState[RG](nil),
			expectedCond: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			cond := CondGroupHasBeenDeleted(c.state)
			assert.Equal(tt, c.expectedCond, cond.Satisfy(), c.desc)
		})
	}
}

func TestCondGroupIsDeleting(t *testing.T) {
	t.Run("PDGroup", testCondGroupIsDeleting[runtime.PDGroup])
	t.Run("TiDBGroup", testCondGroupIsDeleting[runtime.TiDBGroup])
	t.Run("TiKVGroup", testCondGroupIsDeleting[runtime.TiKVGroup])
	t.Run("TiFlashGroup", testCondGroupIsDeleting[runtime.TiFlashGroup])
}

func testCondGroupIsDeleting[
	G runtime.GroupSet,
	RG runtime.GroupT[G],
](t *testing.T) {
	cases := []struct {
		desc         string
		state        GroupState[RG]
		expectedCond bool
	}{
		{
			desc: "cond is false",
			state: FakeGroupState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("test")
					return obj
				}),
			),
		},
		{
			desc: "cond is true",
			state: FakeGroupState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("test")
					now := metav1.Now()
					obj.SetDeletionTimestamp(&now)
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

			cond := CondGroupIsDeleting(c.state)
			assert.Equal(tt, c.expectedCond, cond.Satisfy(), c.desc)
		})
	}
}

func TestCondPDHasBeenDeleted(t *testing.T) {
	cases := []struct {
		desc         string
		state        *fakeState[v1alpha1.PD]
		expectedCond bool
	}{
		{
			desc: "cond is false",
			state: &fakeState[v1alpha1.PD]{
				obj: fake.FakeObj[v1alpha1.PD]("test"),
			},
		},
		{
			desc:         "cond is true",
			state:        &fakeState[v1alpha1.PD]{},
			expectedCond: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			s := &fakePDState{s: c.state}
			cond := CondPDHasBeenDeleted(s)
			assert.Equal(tt, c.expectedCond, cond.Satisfy(), c.desc)
		})
	}
}

func TestCondPDIsDeleting(t *testing.T) {
	cases := []struct {
		desc         string
		state        *fakeState[v1alpha1.PD]
		expectedCond bool
	}{
		{
			desc: "cond is false",
			state: &fakeState[v1alpha1.PD]{
				obj: fake.FakeObj[v1alpha1.PD]("test"),
			},
		},
		{
			desc: "cond is true",
			state: &fakeState[v1alpha1.PD]{
				obj: fake.FakeObj("test", fake.DeleteNow[v1alpha1.PD]()),
			},
			expectedCond: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			s := &fakePDState{s: c.state}
			cond := CondPDIsDeleting(s)
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
