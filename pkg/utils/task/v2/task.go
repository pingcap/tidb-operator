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
	"reflect"
	"strings"

	"github.com/olekukonko/tablewriter"
)

// Context is a wrapper of any struct which can return its self
// It's defined to avoid calling ctx.Value()
type Context[T any] interface {
	context.Context
	Self() *T
}

type Task[T any] interface {
	Sync(ctx Context[T]) Result
}

type Condition[T any] interface {
	Satisfy(ctx Context[T]) bool
}

type OptionalTask[T any] interface {
	Task[T]
	Satisfied(ctx Context[T]) bool
}

type FinalTask[T any] interface {
	Task[T]
	IsFinal() bool
}

type TaskNamer interface {
	Name() string
}

type TaskFunc[T any] func(ctx Context[T]) Result

func (f TaskFunc[T]) Sync(ctx Context[T]) Result {
	return f(ctx)
}

type namedTask[T any] struct {
	Task[T]
	name string
}

func NameTaskFunc[T any](name string, t TaskFunc[T]) Task[T] {
	return &namedTask[T]{
		Task: t,
		name: name,
	}
}

func (t *namedTask[T]) Name() string {
	return t.name
}

type CondFunc[T any] func(ctx Context[T]) bool

func (f CondFunc[T]) Satisfy(ctx Context[T]) bool {
	return f(ctx)
}

type optional[T any] struct {
	Task[T]
	cond Condition[T]
}

var (
	_ OptionalTask[int] = &optional[int]{}
	_ FinalTask[int]    = &optional[int]{}
)

func (t *optional[T]) Satisfied(ctx Context[T]) bool {
	return t.cond.Satisfy(ctx)
}

func (t *optional[T]) IsFinal() bool {
	final, ok := t.Task.(FinalTask[T])
	if ok {
		return final.IsFinal()
	}
	return false
}

func NewOptionalTask[T any](cond Condition[T], ts ...Task[T]) Task[T] {
	return &optional[T]{
		Task: NewTaskQueue(ts...),
		cond: cond,
	}
}

func NewSwitchTask[T any](cond Condition[T], ts ...Task[T]) Task[T] {
	return NewOptionalTask(cond, NewFinalTask(ts...))
}

type final[T any] struct {
	Task[T]
}

func (t *final[T]) Satisfied(ctx Context[T]) bool {
	optional, ok := t.Task.(OptionalTask[T])
	if ok {
		return optional.Satisfied(ctx)
	}
	return true
}

func (*final[T]) IsFinal() bool {
	return true
}

var (
	_ OptionalTask[int] = &final[int]{}
	_ FinalTask[int]    = &final[int]{}
)

func NewFinalTask[T any](ts ...Task[T]) Task[T] {
	return &final[T]{Task: NewTaskQueue(ts...)}
}

type queue[T any] struct {
	ts      []Task[T]
	isFinal bool
}

func (t *queue[T]) Sync(ctx Context[T]) Result {
	rs := []Result{}
	for _, tt := range t.ts {
		optional, ok := tt.(OptionalTask[T])
		if ok && !optional.Satisfied(ctx) {
			continue
		}

		r := tt.Sync(ctx)
		rs = append(rs, AnnotateName(Name(tt), r))
		if r.Status() == SFail {
			break
		}

		if final, ok := tt.(FinalTask[T]); ok && final.IsFinal() {
			t.isFinal = true
			break
		}
	}

	return NewAggregateResult(rs...)
}

func (*queue[T]) Satisfied(Context[T]) bool {
	return true
}

func (t *queue[T]) IsFinal() bool {
	return t.isFinal
}

func NewTaskQueue[T any](ts ...Task[T]) Task[T] {
	return &queue[T]{ts: ts}
}

type TaskReporter interface {
	AddResult(r Result)
	Summary() string
}

type tableReporter struct {
	table   *tablewriter.Table
	builder *strings.Builder
}

type dummyReporter struct{}

func (*dummyReporter) AddResult(Result) {}

func (*dummyReporter) Summary() string {
	return ""
}

func NewTableTaskReporter() TaskReporter {
	builder := strings.Builder{}
	table := tablewriter.NewWriter(&builder)
	table.SetHeader([]string{"Name", "Status", "Message"})
	return &tableReporter{
		table:   table,
		builder: &builder,
	}
}

func (t *tableReporter) AddResult(r Result) {
	switch underlying := r.(type) {
	case AggregateResult:
		for _, rr := range underlying.Results() {
			t.AddResult(rr)
		}
	case NamedResult:
		t.table.Append([]string{underlying.Name(), r.Status().String(), r.Message()})
	default:
		t.table.Append([]string{"", r.Status().String(), r.Message()})
	}
}

func (t *tableReporter) Summary() string {
	t.table.Render()
	return t.builder.String()
}

func Name[T any](t Task[T]) string {
	namer, ok := t.(TaskNamer)
	if ok {
		return namer.Name()
	}

	return reflect.TypeOf(t).Name()
}
