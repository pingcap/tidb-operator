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
	"strings"
	"time"
)

type Status int

const (
	// task is complete and will not be requeue
	SComplete Status = iota
	// task is unexpectedly failed, runner will be interrupted
	SFail
	// some preconditions are not met, wait update events to trigger next run
	SWait
	// retry tasks after specified duration
	SRetry
)

func (s Status) String() string {
	switch s {
	case SComplete:
		return "Complete"
	case SFail:
		return "Fail"
	case SWait:
		return "Wait"
	case SRetry:
		return "Retry"
	}

	return "Unknown"
}

// Result defines the result of a task
type Result interface {
	Status() Status
	RequeueAfter() time.Duration
	Message() string
}

type NamedResult interface {
	Result
	Name() string
}

type AggregateResult interface {
	Result
	Results() []Result
}

// WithMessage defines an interface to set message into task result
type WithMessage interface {
	With(format string, args ...any) Result
}

type taskResult struct {
	status       Status
	requeueAfter time.Duration
	message      string
}

func (r *taskResult) Status() Status {
	return r.status
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

// Complete means complete the current task and run the next one
func Complete() WithMessage {
	return &taskResult{
		status: SComplete,
	}
}

// Fail means fail the current task and skip all next tasks
func Fail() WithMessage {
	return &taskResult{
		status: SFail,
	}
}

// Retry means continue all next tasks and retry after dur
func Retry(dur time.Duration) WithMessage {
	return &taskResult{
		status:       SRetry,
		requeueAfter: dur,
	}
}

// Wait means continue all next tasks and wait until next event triggers task run
func Wait() WithMessage {
	return &taskResult{
		status: SWait,
	}
}

type namedResult struct {
	Result
	name string
}

func AnnotateName(name string, r Result) Result {
	if _, ok := r.(AggregateResult); ok {
		return r
	}
	return &namedResult{
		Result: r,
		name:   name,
	}
}

func (r *namedResult) Name() string {
	return r.name
}

type aggregateResult struct {
	rs []Result
}

func NewAggregateResult(rs ...Result) AggregateResult {
	return &aggregateResult{rs: rs}
}

func (r *aggregateResult) Results() []Result {
	return r.rs
}

func (r *aggregateResult) Status() Status {
	needRetry := false
	needWait := false
	for _, res := range r.rs {
		switch res.Status() {
		case SFail:
			return SFail
		case SRetry:
			needRetry = true
		case SWait:
			needWait = true
		}
	}

	if needRetry {
		return SRetry
	}

	if needWait {
		return SWait
	}

	return SComplete
}

func (r *aggregateResult) RequeueAfter() time.Duration {
	var minDur time.Duration = 0
	for _, res := range r.rs {
		if minDur < res.RequeueAfter() {
			minDur = res.RequeueAfter()
		}
	}

	return minDur
}

func (r *aggregateResult) Message() string {
	sb := strings.Builder{}
	for _, res := range r.rs {
		sb.WriteString(res.Message())
	}

	return sb.String()
}
