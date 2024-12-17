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

package task

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeTask[T any] struct {
	name string
	res  Result
}

func (t *fakeTask[T]) Name() string {
	return t.name
}

func (t *fakeTask[T]) Sync(_ Context[T]) Result {
	return t.res
}

func NewFakeTask[T any](name string, res Result) Task[T] {
	return &fakeTask[T]{
		name: name,
		res:  res,
	}
}

type task struct {
	res    WithMessage
	status string
}

func TestTaskRunner(t *testing.T) {
	mc := gomock.NewController(t)

	cases := []struct {
		desc   string
		tasks  []task
		hasErr bool
		res    ctrl.Result
	}{
		{
			desc:   "empty tasks",
			hasErr: false,
		},
		{
			desc: "two complete tasks",
			tasks: []task{
				{
					res:    Complete(),
					status: "Complete",
				},
				{
					res:    Complete(),
					status: "Complete",
				},
			},
			hasErr: false,
		},
		{
			desc: "a complete but break task",
			tasks: []task{
				{
					res:    Complete().Break(),
					status: "Complete",
				},
				{
					res:    Complete(),
					status: "Skip",
				},
			},
			hasErr: false,
		},
		{
			desc: "a fail task",
			tasks: []task{
				{
					res:    Fail(),
					status: "Fail",
				},
				{
					res:    Complete(),
					status: "NotRun",
				},
			},
			hasErr: true,
			res:    ctrl.Result{},
		},
		{
			desc: "a fail but continue task",
			tasks: []task{
				{
					res:    Fail().Continue(),
					status: "Fail",
				},
				{
					res:    Complete(),
					status: "Complete",
				},
			},
			hasErr: true,
		},
		{
			desc: "a retry task",
			tasks: []task{
				{
					res:    Retry(time.Second),
					status: "Retry",
				},
				{
					res:    Complete(),
					status: "Complete",
				},
			},
			hasErr: false,
			res: ctrl.Result{
				Requeue:      true,
				RequeueAfter: time.Second,
			},
		},
		{
			desc: "many retry tasks",
			tasks: []task{
				{
					res:    Retry(3 * time.Second),
					status: "Retry",
				},
				{
					res:    Retry(2 * time.Second),
					status: "Retry",
				},
				{
					res:    Retry(1 * time.Second),
					status: "Retry",
				},
			},
			hasErr: false,
			res: ctrl.Result{
				Requeue:      true,
				RequeueAfter: time.Second,
			},
		},
		{
			desc: "retry + failed tasks",
			tasks: []task{
				{
					res:    Retry(time.Second),
					status: "Retry",
				},
				{
					res:    Fail(),
					status: "Fail",
				},
			},
			hasErr: true,
		},
		{
			desc: "retry break + failed tasks",
			tasks: []task{
				{
					res:    Retry(time.Second).Break(),
					status: "Retry",
				},
				{
					res:    Fail(),
					status: "Skip",
				},
			},
			hasErr: false,
			res: ctrl.Result{
				Requeue:      true,
				RequeueAfter: time.Second,
			},
		},
		{
			desc: "retry + complete tasks",
			tasks: []task{
				{
					res:    Retry(time.Second),
					status: "Retry",
				},
				{
					res:    Complete(),
					status: "Complete",
				},
			},
			hasErr: false,
			res: ctrl.Result{
				Requeue:      true,
				RequeueAfter: time.Second,
			},
		},
		{
			desc: "retry break + complete tasks",
			tasks: []task{
				{
					res:    Retry(time.Second).Break(),
					status: "Retry",
				},
				{
					res:    Complete(),
					status: "Skip",
				},
			},
			hasErr: false,
			res: ctrl.Result{
				Requeue:      true,
				RequeueAfter: time.Second,
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			r := NewMockTaskReporter(mc)
			tr := NewTaskRunner[struct{}](r)

			for i, task := range c.tasks {
				name := strconv.Itoa(i)
				r.EXPECT().AddResult(
					name,
					task.status,
					// ignore msg
					gomock.Any(),
				)
				tr.AddTasks(NewFakeTask[struct{}](name, task.res.With("")))
			}

			res, err := tr.Run(nil)
			if c.hasErr {
				assert.Error(tt, err, c.desc)
			} else {
				require.NoError(tt, err, c.desc)
				assert.Equal(tt, c.res, res, c.desc)
			}
		})
	}

	tr := NewTaskRunner[struct{}](nil)
	tr.AddTasks(NewFakeTask[struct{}]("panic", nil))
	assert.Panics(t, func() {
		_, err := tr.Run(nil)
		assert.NoError(t, err)
	})
}
