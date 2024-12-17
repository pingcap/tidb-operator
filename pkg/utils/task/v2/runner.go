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
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
)

// TaskRunner is an executor to run a series of tasks sequentially
type TaskRunner[T any] interface {
	Run(ctx Context[T]) (ctrl.Result, error)
}

type taskRunner[T any] struct {
	reporter  TaskReporter
	taskQueue Task[T]
}

// There are four status of tasks
// - Complete: means this task is complete and all is expected
// - Failed: means an err occurred
// - Retry: means this task need to wait an interval and retry
// - Wait: means this task will wait for next event trigger
// And five results of reconiling
// 1. All tasks are complete, the key will not be re-added
// 2. Some tasks are failed, return err and wait with backoff
// 3. Some tasks need retry, requeue after an interval
// 4. Some tasks are not run, return err and wait with backoff
// 5. Particular tasks are complete and left are skipped, the key will not be re-added
func NewTaskRunner[T any](reporter TaskReporter, ts ...Task[T]) TaskRunner[T] {
	if reporter == nil {
		reporter = &dummyReporter{}
	}
	return &taskRunner[T]{
		reporter:  reporter,
		taskQueue: NewTaskQueue(ts...),
	}
}

func (r *taskRunner[T]) Run(ctx Context[T]) (ctrl.Result, error) {
	res := r.taskQueue.Sync(ctx)
	r.reporter.AddResult(res)

	switch res.Status() {
	case SFail:
		return ctrl.Result{}, fmt.Errorf("some tasks are failed: %v", res.Message())
	case SRetry:
		return ctrl.Result{
			RequeueAfter: res.RequeueAfter(),
		}, nil
	default:
		// SComplete and SWait
		return ctrl.Result{}, nil
	}
}
