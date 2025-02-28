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
	"errors"
	"fmt"
	"time"
)

// RequeueError is used to requeue the item, this error type should't be considered as a real error
type RequeueError struct {
	s        string
	Duration time.Duration
}

func (re *RequeueError) Error() string {
	return re.s
}

// RequeueErrorf returns a RequeueError
func RequeueErrorf(duration time.Duration, format string, a ...any) error {
	return &RequeueError{
		Duration: duration,
		s:        fmt.Sprintf(format, a...),
	}
}

// IsRequeueError returns whether err is a RequeueError
func IsRequeueError(err error) bool {
	rerr := &RequeueError{}
	return errors.As(err, &rerr)
}

// IgnoreError is used to ignore this item, this error type shouldn't be considered as a real error, no need to requeue
type IgnoreError struct {
	s string
}

func (re *IgnoreError) Error() string {
	return re.s
}

// IgnoreErrorf returns a IgnoreError
func IgnoreErrorf(format string, a ...any) error {
	return &IgnoreError{fmt.Sprintf(format, a...)}
}

// IsIgnoreError returns whether err is a IgnoreError
func IsIgnoreError(err error) bool {
	_, ok := err.(*IgnoreError)
	return ok
}
