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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestTaskRunner(t *testing.T) {
	cases := []struct {
		desc           string
		ts             []Task
		hasErr         bool
		expectedResult ctrl.Result
	}{
		{
			desc: "a task fail",
			ts: []Task{
				NameTaskFunc("aaa", func(context.Context) Result {
					return Complete().With("success")
				}),
				NameTaskFunc("bbb", func(context.Context) Result {
					return Fail().With("fail")
				}),
			},
			hasErr: true,
		},
		{
			desc: "a retry task with 0 interval",
			ts: []Task{
				NameTaskFunc("aaa", func(context.Context) Result {
					return Complete().With("success")
				}),
				NameTaskFunc("bbb", func(context.Context) Result {
					return Retry(0).With("retry")
				}),
			},
			expectedResult: ctrl.Result{
				Requeue: true,
			},
		},
		{
			desc: "a retry task with not 0 interval",
			ts: []Task{
				NameTaskFunc("aaa", func(context.Context) Result {
					return Complete().With("success")
				}),
				NameTaskFunc("bbb", func(context.Context) Result {
					return Retry(5).With("retry")
				}),
			},
			expectedResult: ctrl.Result{
				RequeueAfter: 5,
			},
		},
		{
			desc: "all tasks are Complete or Wait",
			ts: []Task{
				NameTaskFunc("aaa", func(context.Context) Result {
					return Complete().With("success")
				}),
				NameTaskFunc("bbb", func(context.Context) Result {
					return Wait().With("wait")
				}),
				NameTaskFunc("ccc", func(context.Context) Result {
					return Complete().With("success")
				}),
			},
			expectedResult: ctrl.Result{},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			runner := NewTaskRunner(&dummyReporter{}, c.ts...)
			res, err := runner.Run(ctx)
			if c.hasErr {
				assert.Error(tt, err, c.desc)
			} else {
				assert.Equal(tt, c.expectedResult, res, c.desc)
			}
		})
	}
}

func TestTaskReporter(t *testing.T) {
	cases := []struct {
		desc            string
		rs              []Result
		expectedSummary string
	}{
		{
			desc: "no result",
			rs:   nil,
			expectedSummary: `
+----+------+--------+---------+
| ID | NAME | STATUS | MESSAGE |
+----+------+--------+---------+
+----+------+--------+---------+
`,
		},
		{
			desc: "unnamed result",
			rs: []Result{
				Complete().With("success"),
			},
			expectedSummary: `
+------+------+----------+---------+
|  ID  | NAME |  STATUS  | MESSAGE |
+------+------+----------+---------+
| test |      | Complete | success |
+------+------+----------+---------+
`,
		},
		{
			desc: "named result",
			rs: []Result{
				nameResult("aaa", Complete().With("success")),
			},
			expectedSummary: `
+------+------+----------+---------+
|  ID  | NAME |  STATUS  | MESSAGE |
+------+------+----------+---------+
| test | aaa  | Complete | success |
+------+------+----------+---------+
`,
		},
		{
			desc: "aggregate result",
			rs: []Result{
				nameResult("aaa", Complete().With("success")),
				newAggregate(
					nameResult("bbb", Complete().With("success")),
					nameResult("ccc", Complete().With("success")),
				),
			},
			expectedSummary: `
+------+------+----------+---------+
|  ID  | NAME |  STATUS  | MESSAGE |
+------+------+----------+---------+
| test | aaa  | Complete | success |
| test | bbb  | Complete | success |
| test | ccc  | Complete | success |
+------+------+----------+---------+
`,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			reporter := NewTableTaskReporter("test")
			for _, r := range c.rs {
				reporter.AddResult(r)
			}
			summary := reporter.Summary()
			assert.Equal(tt, c.expectedSummary, "\n"+summary, c.desc)
		})
	}
}
