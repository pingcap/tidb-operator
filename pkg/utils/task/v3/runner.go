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

	"github.com/olekukonko/tablewriter"
	ctrl "sigs.k8s.io/controller-runtime"
)

// TaskRunner is an executor to run a series of tasks sequentially
type TaskRunner interface {
	Run(ctx context.Context) (ctrl.Result, error)
}

type taskRunner struct {
	reporter TaskReporter
	task     Task
}

// There are four status of tasks
// - Complete: means this task is complete and all is expected
// - Failed: means an err occurred
// - Retry: means this task need to wait an interval and retry
// - Wait: means this task will wait for next event trigger
// And three results of reconiling
// 1. A task is failed, return err and wait with backoff
// 2. Some tasks need retry and max interval is 0, requeue directly with backoff
// 3. Some tasks need retry and max interval is higher than MinRequeueAfter, requeue after the max interval without backoff
// 4. All tasks are complete or wait, the key will not be re-added
func NewTaskRunner(reporter TaskReporter, ts ...Task) TaskRunner {
	if reporter == nil {
		reporter = &dummyReporter{}
	}
	return &taskRunner{
		reporter: reporter,
		task:     Block(ts...),
	}
}

func (r *taskRunner) Run(ctx context.Context) (ctrl.Result, error) {
	res, _ := RunTask(ctx, r.task)
	r.reporter.AddResult(res)

	switch res.Status() {
	case SFail:
		return ctrl.Result{}, fmt.Errorf("some tasks are failed: %v", res.Message())
	case SRetry:
		if res.RequeueAfter() == 0 {
			return ctrl.Result{
				Requeue: true,
			}, nil
		}
		return ctrl.Result{
			RequeueAfter: res.RequeueAfter(),
		}, nil
	default:
		// SComplete and SWait
		return ctrl.Result{}, nil
	}
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

const (
	tableColWidth = 80
)

func NewTableTaskReporter() TaskReporter {
	builder := strings.Builder{}
	table := tablewriter.NewWriter(&builder)
	table.SetColWidth(tableColWidth)
	table.SetHeader([]string{"Name", "Status", "Message"})
	return &tableReporter{
		table:   table,
		builder: &builder,
	}
}

func (t *tableReporter) AddResult(r Result) {
	switch underlying := r.(type) {
	case AggregateResult:
		for _, rr := range underlying.results() {
			t.AddResult(rr)
		}
	case NamedResult:
		t.table.Append([]string{underlying.name(), r.Status().String(), r.Message()})
	default:
		t.table.Append([]string{"", r.Status().String(), r.Message()})
	}
}

func (t *tableReporter) Summary() string {
	t.table.Render()
	return t.builder.String()
}
