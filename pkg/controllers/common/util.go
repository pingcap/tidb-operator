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
func RequeueErrorf(duration time.Duration, format string, a ...interface{}) error {
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
func IgnoreErrorf(format string, a ...interface{}) error {
	return &IgnoreError{fmt.Sprintf(format, a...)}
}

// IsIgnoreError returns whether err is a IgnoreError
func IsIgnoreError(err error) bool {
	_, ok := err.(*IgnoreError)
	return ok
}
