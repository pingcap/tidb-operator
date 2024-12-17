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

package time

import "time"

type Clock interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

var (
	_ Clock = &RealClock{}
	_ Clock = &FakeClock{}
)

type RealClock struct{}

func (RealClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (RealClock) Now() time.Time {
	return time.Now()
}

type FakeClock struct {
	NowFunc   func() time.Time
	SinceFunc func(time.Time) time.Duration
}

func (f FakeClock) Now() time.Time {
	return f.NowFunc()
}

func (f FakeClock) Since(t time.Time) time.Duration {
	return f.SinceFunc(t)
}
