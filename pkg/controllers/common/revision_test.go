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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type fakeRevisionStateInitializer[G runtime.Group] struct {
	fakeRevisionState

	currentRevision CurrentRevisionOption
	collisionCount  CollisionCountOption
	parent          ParentOption[G]
	labels          LabelsOption
}

func (f *fakeRevisionStateInitializer[G]) RevisionInitializer() RevisionInitializer[G] {
	return NewRevision[G](&f.fakeRevisionState).
		WithCurrentRevision(f.currentRevision).
		WithCollisionCount(f.collisionCount).
		WithParent(f.parent).
		WithLabels(f.labels).
		Initializer()
}

const (
	fakeOldRevision = "aaa-c9f48df69"
	fakeNewRevision = "aaa-c9f48df68"
)

func TestTaskRevision(t *testing.T) {
	cases := []struct {
		desc  string
		state *fakeRevisionStateInitializer[*runtime.PDGroup]
		objs  []client.Object

		expectedResult          task.Status
		expectedUpdateRevision  string
		expectedCurrentRevision string
		expectedCollisionCount  int32
	}{
		{
			desc: "no revisions",
			state: &fakeRevisionStateInitializer[*runtime.PDGroup]{
				currentRevision: Lazy[string](func() string {
					return ""
				}),
				collisionCount: Lazy[*int32](func() *int32 {
					return nil
				}),
				parent: Lazy[*runtime.PDGroup](func() *runtime.PDGroup {
					return fake.Fake(func(obj *runtime.PDGroup) *runtime.PDGroup {
						obj.SetName("aaa")
						obj.Labels = map[string]string{
							"aaa": "bbb",
						}
						return obj
					})
				}),
				labels: Labels{
					"ccc": "ddd",
				},
			},
			expectedUpdateRevision:  fakeOldRevision,
			expectedCurrentRevision: fakeOldRevision,
		},
		{
			desc: "has a revision",
			state: &fakeRevisionStateInitializer[*runtime.PDGroup]{
				currentRevision: Lazy[string](func() string {
					return "xxx"
				}),
				collisionCount: Lazy[*int32](func() *int32 {
					return nil
				}),
				parent: Lazy[*runtime.PDGroup](func() *runtime.PDGroup {
					return fake.Fake(func(obj *runtime.PDGroup) *runtime.PDGroup {
						obj.SetName("aaa")
						obj.Labels = map[string]string{
							"aaa": "bbb",
						}
						return obj
					})
				}),
				labels: Labels{
					"ccc": "ddd",
				},
			},
			objs: []client.Object{
				fake.FakeObj("xxx", fake.Label[appsv1.ControllerRevision]("ccc", "ddd")),
			},
			expectedUpdateRevision:  fakeOldRevision,
			expectedCurrentRevision: "xxx",
		},
		{
			desc: "has a coflict revision",
			state: &fakeRevisionStateInitializer[*runtime.PDGroup]{
				currentRevision: Lazy[string](func() string {
					return fakeOldRevision
				}),
				collisionCount: Lazy[*int32](func() *int32 {
					return nil
				}),
				parent: Lazy[*runtime.PDGroup](func() *runtime.PDGroup {
					return fake.Fake(func(obj *runtime.PDGroup) *runtime.PDGroup {
						obj.SetName("aaa")
						obj.Labels = map[string]string{
							"aaa": "bbb",
						}
						return obj
					})
				}),
				labels: Labels{
					"ccc": "ddd",
				},
			},
			objs: []client.Object{
				fake.FakeObj(fakeOldRevision, fake.Label[appsv1.ControllerRevision]("ccc", "ddd")),
			},
			expectedUpdateRevision:  fakeNewRevision,
			expectedCurrentRevision: fakeOldRevision,
			expectedCollisionCount:  1,
		},
		{
			desc: "has two coflict revision",
			state: &fakeRevisionStateInitializer[*runtime.PDGroup]{
				currentRevision: Lazy[string](func() string {
					return fakeOldRevision
				}),
				collisionCount: Lazy[*int32](func() *int32 {
					return nil
				}),
				parent: Lazy[*runtime.PDGroup](func() *runtime.PDGroup {
					return fake.Fake(func(obj *runtime.PDGroup) *runtime.PDGroup {
						obj.SetName("aaa")
						obj.Labels = map[string]string{
							"aaa": "bbb",
						}
						return obj
					})
				}),
				labels: Labels{
					"ccc": "ddd",
				},
			},
			objs: []client.Object{
				fake.FakeObj(fakeOldRevision, fake.Label[appsv1.ControllerRevision]("ccc", "ddd")),
				fake.FakeObj(fakeNewRevision, fake.Label[appsv1.ControllerRevision]("ccc", "ddd")),
			},
			expectedUpdateRevision:  "aaa-c9f48df6c",
			expectedCurrentRevision: fakeOldRevision,
			expectedCollisionCount:  2,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.objs...)

			res, done := task.RunTask(context.Background(), TaskRevision[runtime.PDGroupTuple](c.state, fc))
			assert.Equal(tt, c.expectedResult.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)
			update, current, collisionCount := c.state.Revision()
			assert.Equal(tt, c.expectedUpdateRevision, update, c.desc)
			assert.Equal(tt, c.expectedCurrentRevision, current, c.desc)
			assert.Equal(tt, c.expectedCollisionCount, collisionCount, c.desc)

			// record currentRevision and collisionCount
			c.state.currentRevision = Lazy[string](func() string {
				return current
			})
			c.state.collisionCount = Lazy[*int32](func() *int32 {
				return &collisionCount
			})

			// rerun Revision task and make sure that status is unchanged
			res2, _ := task.RunTask(context.Background(), TaskRevision[runtime.PDGroupTuple](c.state, fc))
			assert.Equal(tt, c.expectedResult.String(), res2.Status().String(), c.desc)
			update, current, collisionCount = c.state.Revision()
			assert.Equal(tt, c.expectedUpdateRevision, update, c.desc)
			assert.Equal(tt, c.expectedCurrentRevision, current, c.desc)
			assert.Equal(tt, c.expectedCollisionCount, collisionCount, c.desc)
		})
	}
}
