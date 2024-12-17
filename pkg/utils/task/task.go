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
	"fmt"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Context is a wrapper of any struct which can return its self
// It's defined to avoid calling ctx.Value()
type Context[T any] interface {
	context.Context
	Self() *T
}

// Result defines the result of a task
type Result interface {
	IsFailed() bool
	ShouldContinue() bool
	RequeueAfter() time.Duration
	Message() string
}

// WithMessage defines an interface to set message into task result
type WithMessage interface {
	With(format string, args ...any) Result
}

// BreakableResult defines a result which can stop task execution
type BreakableResult interface {
	WithMessage
	// Break will stop task execution
	Break() WithMessage
}

// ContinuableResult defines a result which can continue task execution
type ContinuableResult interface {
	WithMessage
	// Continue ignores errs of the current task and will continue task execution
	Continue() WithMessage
}

// TaskRunner is an executor to run a series of tasks sequentially
type TaskRunner[T any] interface {
	AddTasks(tasks ...Task[T])
	Run(ctx Context[T]) (ctrl.Result, error)
}

// Task defines a task can be executed by TaskRunner
type Task[T any] interface {
	Name() string
	Sync(ctx Context[T]) Result
}

type taskResult struct {
	isFailed       bool
	shouldContinue bool
	requeueAfter   time.Duration
	message        string
}

func (r *taskResult) IsFailed() bool {
	return r.isFailed
}

func (r *taskResult) ShouldContinue() bool {
	return r.shouldContinue
}

func (r *taskResult) RequeueAfter() time.Duration {
	return r.requeueAfter
}

func (r *taskResult) Message() string {
	if r.requeueAfter > 0 {
		return fmt.Sprintf("%s(requeue after %s)", r.message, r.requeueAfter)
	}
	return r.message
}

func (r *taskResult) With(format string, args ...any) Result {
	r.message = fmt.Sprintf(format, args...)
	return r
}

func (r *taskResult) Break() WithMessage {
	r.shouldContinue = false
	return r
}

func (r *taskResult) Continue() WithMessage {
	r.shouldContinue = true
	return r
}

// Complete means complete the current task and run the next one
func Complete() BreakableResult {
	return &taskResult{
		isFailed:       false,
		shouldContinue: true,
	}
}

// Fail means fail the current task and skip all next tasks
func Fail() ContinuableResult {
	return &taskResult{
		isFailed:       true,
		shouldContinue: false,
	}
}

// Retry means continue all next tasks and retry after dur
func Retry(dur time.Duration) BreakableResult {
	return &taskResult{
		isFailed:       false,
		shouldContinue: true,
		requeueAfter:   dur,
	}
}

// TaskReporter appends results of tasks and output a summary
type TaskReporter interface {
	AddResult(name, status, msg string)
	Summary() string
}

type tableReporter struct {
	table   *tablewriter.Table
	builder *strings.Builder
}

type dummyReporter struct{}

func (*dummyReporter) AddResult(_, _, _ string) {}

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

func (t *tableReporter) AddResult(name, status, msg string) {
	t.table.Append([]string{name, status, msg})
}

func (t *tableReporter) Summary() string {
	t.table.Render()
	return t.builder.String()
}

type taskRunner[T any] struct {
	reporter TaskReporter
	tasks    []Task[T]
}

// There are five status of tasks
// - Complete: means this task is complete and all is expected
// - Failed: means an err occurred
// - Retry: means this task need to wait an interval and retry
// - NotRun: means this task is not run
// - Skip: means this task is skipped
// And five results of reconiling
// 1. All tasks are complete, the key will not be re-added
// 2. Some tasks are failed, return err and wait with backoff
// 3. Some tasks need retry, requeue after an interval
// 4. Some tasks are not run, return err and wait with backoff
// 5. Particular tasks are complete and left are skipped, the key will not be re-added
func NewTaskRunner[T any](reporter TaskReporter) TaskRunner[T] {
	if reporter == nil {
		reporter = &dummyReporter{}
	}
	return &taskRunner[T]{
		reporter: reporter,
	}
}

func (r *taskRunner[T]) AddTasks(ts ...Task[T]) {
	r.tasks = append(r.tasks, ts...)
}

func (r *taskRunner[T]) Run(ctx Context[T]) (ctrl.Result, error) {
	shouldContinue := true
	minRequeueAfter := time.Duration(0)
	failedTasks := []string{}
	for _, t := range r.tasks {
		if !shouldContinue {
			if len(failedTasks) != 0 {
				// write unknown info
				r.reporter.AddResult(t.Name(), "NotRun", "")
			} else {
				r.reporter.AddResult(t.Name(), "Skip", "")
			}
			continue
		}

		res := t.Sync(ctx)

		if res == nil {
			panic("please set result with message for " + t.Name())
		}

		if !res.ShouldContinue() {
			shouldContinue = false
		}

		if res.IsFailed() {
			// write fail info
			r.reporter.AddResult(t.Name(), "Fail", res.Message())
			failedTasks = append(failedTasks, t.Name())
			continue
		}

		dur := res.RequeueAfter()
		if dur > 0 {
			r.reporter.AddResult(t.Name(), "Retry", res.Message())

			if minRequeueAfter == 0 {
				minRequeueAfter = dur
			} else if dur > 0 && dur < minRequeueAfter {
				minRequeueAfter = dur
			}
		} else {
			// write complete info
			r.reporter.AddResult(t.Name(), "Complete", res.Message())
		}
	}

	// some tasks are failed
	if len(failedTasks) != 0 {
		return ctrl.Result{}, fmt.Errorf("some tasks are failed:  %v", failedTasks)
	}

	if minRequeueAfter > 0 {
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: minRequeueAfter,
		}, nil
	}

	return ctrl.Result{}, nil
}
