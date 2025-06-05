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

import "context"

// Syncer defines an action to sync actual states to desired.
type Syncer interface {
	Sync(ctx context.Context) Result
}

type SyncFunc func(ctx context.Context) Result

func (f SyncFunc) Sync(ctx context.Context) Result {
	return f(ctx)
}

type Condition interface {
	Satisfy() bool
}

type CondFunc func() bool

func (f CondFunc) Satisfy() bool {
	return f()
}

// Task is a Syncer wrapper, which can be orchestrated using control structures
// such as if and break for conditional logic and flow control.
// Task can only be implemented by this package and users should implement the Syncer interface
// and wrap Syncers as Tasks
type Task interface {
	sync(ctx context.Context) (_ Result, done bool)
}

func RunTask(ctx context.Context, t Task) (Result, bool) {
	return t.sync(ctx)
}

type task struct {
	name string
	f    Syncer
}

func (e *task) sync(ctx context.Context) (Result, bool) {
	return nameResult(e.name, e.f.Sync(ctx)), false
}

func NameTaskFunc(name string, f SyncFunc) Task {
	return &task{
		name: name,
		f:    f,
	}
}

type optionalTask struct {
	Task
	cond Condition
}

func (e *optionalTask) sync(ctx context.Context) (Result, bool) {
	if e.cond.Satisfy() {
		return e.Task.sync(ctx)
	}

	return nil, false
}

func If(cond Condition, tasks ...Task) Task {
	return &optionalTask{
		Task: Block(tasks...),
		cond: cond,
	}
}
func IfNot(cond Condition, tasks ...Task) Task {
	return &optionalTask{
		Task: Block(tasks...),
		cond: CondFunc(func() bool { return !cond.Satisfy() }),
	}
}

type breakTask struct {
	Task
}

func (e *breakTask) sync(ctx context.Context) (Result, bool) {
	r, _ := e.Task.sync(ctx)
	return r, true
}

func Break(tasks ...Task) Task {
	return &breakTask{
		Task: Block(tasks...),
	}
}

func IfBreak(cond Condition, tasks ...Task) Task {
	return If(cond, Break(tasks...))
}

type blockTask struct {
	tasks []Task
}

func (e *blockTask) sync(ctx context.Context) (Result, bool) {
	var rs []Result
	for _, expr := range e.tasks {
		r, done := expr.sync(ctx)
		if r == nil {
			continue
		}

		rs = append(rs, r)
		if r.Status() == SFail {
			break
		}

		if done {
			return newAggregate(rs...), true
		}
	}

	return newAggregate(rs...), false
}

func Block(tasks ...Task) Task {
	return &blockTask{tasks: tasks}
}
