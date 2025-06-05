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
)

func condition(b bool) Condition {
	return CondFunc(func() bool {
		return b
	})
}

func TestNameTask(t *testing.T) {
	cases := []struct {
		desc           string
		name           string
		syncer         SyncFunc
		expectedResult Result
	}{
		{
			desc: "normal",
			name: "aaa",
			syncer: SyncFunc(func(context.Context) Result {
				return Complete().With("success")
			}),
			expectedResult: nameResult(
				"aaa",
				Complete().With("success"),
			),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()
			ctx := context.Background()
			task := NameTaskFunc(c.name, c.syncer)
			res, done := task.sync(ctx)
			assert.Equal(tt, c.expectedResult, res, c.desc)
			assert.False(tt, done, c.desc)
		})
	}
}

func TestIf(t *testing.T) {
	cases := []struct {
		desc           string
		task           Task
		cond           Condition
		expectedResult Result
	}{
		{
			desc: "cond is true",
			task: NameTaskFunc("aaa", func(context.Context) Result {
				return Complete().With("success")
			}),
			cond: condition(true),
			expectedResult: newAggregate(
				nameResult("aaa", Complete().With("success")),
			),
		},
		{
			desc: "cond is false",
			task: NameTaskFunc("aaa", func(context.Context) Result {
				return Complete().With("success")
			}),
			cond:           condition(false),
			expectedResult: nil,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			task := If(c.cond, c.task)
			res, done := task.sync(ctx)
			assert.Equal(tt, c.expectedResult, res, c.desc)
			assert.False(tt, done, c.desc)
		})
	}
}

func TestIfNot(t *testing.T) {
	cases := []struct {
		desc           string
		task           Task
		cond           Condition
		expectedResult Result
	}{
		{
			desc: "cond is true",
			task: NameTaskFunc("aaa", func(context.Context) Result {
				return Complete().With("success")
			}),
			cond:           condition(true),
			expectedResult: nil,
		},
		{
			desc: "cond is false",
			task: NameTaskFunc("aaa", func(context.Context) Result {
				return Complete().With("success")
			}),
			cond: condition(false),
			expectedResult: newAggregate(
				nameResult("aaa", Complete().With("success")),
			),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			task := IfNot(c.cond, c.task)
			res, done := task.sync(ctx)
			assert.Equal(tt, c.expectedResult, res, c.desc)
			assert.False(tt, done, c.desc)
		})
	}
}

func TestBreak(t *testing.T) {
	cases := []struct {
		desc           string
		task           Task
		expectedResult Result
	}{
		{
			desc: "cond is true",
			task: NameTaskFunc("aaa", func(context.Context) Result {
				return Complete().With("success")
			}),
			expectedResult: newAggregate(
				nameResult("aaa", Complete().With("success")),
			),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			task := Break(c.task)
			res, done := task.sync(ctx)
			assert.Equal(tt, c.expectedResult, res, c.desc)
			assert.True(tt, done, c.desc)
		})
	}
}

func TestIfBreak(t *testing.T) {
	cases := []struct {
		desc           string
		task           Task
		cond           Condition
		expectedResult Result
		expectedDone   bool
	}{
		{
			desc: "cond is true",
			task: NameTaskFunc("aaa", func(context.Context) Result {
				return Complete().With("success")
			}),
			cond: CondFunc(func() bool { return true }),
			expectedResult: newAggregate(
				nameResult("aaa", Complete().With("success")),
			),
			expectedDone: true,
		},
		{
			desc: "cond is false",
			task: NameTaskFunc("aaa", func(context.Context) Result {
				return Complete().With("success")
			}),
			cond:           CondFunc(func() bool { return false }),
			expectedResult: nil,
			expectedDone:   false,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			task := IfBreak(c.cond, c.task)
			res, done := task.sync(ctx)
			assert.Equal(tt, c.expectedResult, res, c.desc)
			assert.Equal(tt, c.expectedDone, done, c.desc)
		})
	}
}

func TestBlock(t *testing.T) {
	cases := []struct {
		desc           string
		tasks          []Task
		expectedResult Result
		expectedDone   bool
	}{
		{
			desc:           "no task",
			tasks:          nil,
			expectedResult: newAggregate(),
			expectedDone:   false,
		},
		{
			desc: "1 complete task",
			tasks: []Task{
				NameTaskFunc("aaa", func(context.Context) Result {
					return Complete().With("success")
				}),
			},
			expectedResult: newAggregate(
				nameResult("aaa", Complete().With("success")),
			),
			expectedDone: false,
		},
		{
			desc: "2 complete tasks",
			tasks: []Task{
				NameTaskFunc("aaa", func(context.Context) Result {
					return Complete().With("success")
				}),
				NameTaskFunc("bbb", func(context.Context) Result {
					return Complete().With("success")
				}),
			},
			expectedResult: newAggregate(
				nameResult("aaa", Complete().With("success")),
				nameResult("bbb", Complete().With("success")),
			),
			expectedDone: false,
		},
		{
			desc: "2 tasks with 1 fail task",
			tasks: []Task{
				NameTaskFunc("aaa", func(context.Context) Result {
					return Fail().With("fail")
				}),
				NameTaskFunc("bbb", func(context.Context) Result {
					return Complete().With("success")
				}),
			},
			expectedResult: newAggregate(
				nameResult("aaa", Fail().With("fail")),
			),
			expectedDone: false,
		},
		{
			desc: "if task",
			tasks: []Task{
				If(condition(false), NameTaskFunc("aaa", func(context.Context) Result {
					return Fail().With("fail")
				})),
				NameTaskFunc("bbb", func(context.Context) Result {
					return Complete().With("success")
				}),
			},
			expectedResult: newAggregate(
				nameResult("bbb", Complete().With("success")),
			),
			expectedDone: false,
		},
		{
			desc: "break task",
			tasks: []Task{
				NameTaskFunc("aaa", func(context.Context) Result {
					return Complete().With("success")
				}),
				Break(NameTaskFunc("bbb", func(context.Context) Result {
					return Complete().With("success")
				})),
				NameTaskFunc("ccc", func(context.Context) Result {
					return Complete().With("success")
				}),
			},
			expectedResult: newAggregate(
				nameResult("aaa", Complete().With("success")),
				nameResult("bbb", Complete().With("success")),
			),
			expectedDone: true,
		},
		{
			desc: "if break task",
			tasks: []Task{
				IfBreak(condition(true), NameTaskFunc("aaa", func(context.Context) Result {
					return Complete().With("success")
				})),
				NameTaskFunc("bbb", func(context.Context) Result {
					return Complete().With("success")
				}),
			},
			expectedResult: newAggregate(
				nameResult("aaa", Complete().With("success")),
			),
			expectedDone: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			task := Block(c.tasks...)
			r, done := task.sync(ctx)
			assert.Equal(tt, c.expectedResult, r, c.desc)
			assert.Equal(tt, c.expectedDone, done, c.desc)
		})
	}
}
