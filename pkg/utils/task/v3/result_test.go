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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAggregateResult(t *testing.T) {
	cases := []struct {
		desc                 string
		rs                   []Result
		expectedResults      []Result
		expectedStatus       Status
		expectedRequeueAfter time.Duration
		expectedMessage      string
	}{
		{
			desc:                 "no result",
			rs:                   nil,
			expectedResults:      nil,
			expectedStatus:       SComplete,
			expectedRequeueAfter: 0,
			expectedMessage:      "",
		},
		{
			desc: "only complete result",
			rs: []Result{
				Complete().With("success"),
				Complete().With("success"),
			},
			expectedResults: []Result{
				Complete().With("success"),
				Complete().With("success"),
			},
			expectedStatus:       SComplete,
			expectedRequeueAfter: 0,
			expectedMessage:      "success\nsuccess\n",
		},
		{
			desc: "contains a fail result",
			rs: []Result{
				Complete().With("success"),
				Retry(1).With("retry"),
				Wait().With("wait"),
				Fail().With("fail"),
			},
			expectedResults: []Result{
				Complete().With("success"),
				Retry(1).With("retry"),
				Wait().With("wait"),
				Fail().With("fail"),
			},
			expectedStatus:       SFail,
			expectedRequeueAfter: 1,
			expectedMessage:      "success\nretry(requeue after 1ns)\nwait\nfail\n",
		},
		{
			desc: "contains a retry result and no fail result",
			rs: []Result{
				Complete().With("success"),
				Retry(1).With("retry"),
				Wait().With("wait"),
			},
			expectedResults: []Result{
				Complete().With("success"),
				Retry(1).With("retry"),
				Wait().With("wait"),
			},
			expectedStatus:       SRetry,
			expectedRequeueAfter: 1,
			expectedMessage:      "success\nretry(requeue after 1ns)\nwait\n",
		},
		{
			desc: "contains two retry results",
			rs: []Result{
				Retry(1).With("retry"),
				Retry(2).With("retry"),
			},
			expectedResults: []Result{
				Retry(1).With("retry"),
				Retry(2).With("retry"),
			},
			expectedStatus:       SRetry,
			expectedRequeueAfter: 2,
			expectedMessage:      "retry(requeue after 1ns)\nretry(requeue after 2ns)\n",
		},
		{
			desc: "contains a wait result and no fail and retry result",
			rs: []Result{
				Complete().With("success"),
				Wait().With("wait"),
				Complete().With("success"),
			},
			expectedResults: []Result{
				Complete().With("success"),
				Wait().With("wait"),
				Complete().With("success"),
			},
			expectedStatus:       SWait,
			expectedRequeueAfter: 0,
			expectedMessage:      "success\nwait\nsuccess\n",
		},
		{
			desc: "contains an aggregate result",
			rs: []Result{
				Complete().With("success"),
				newAggregate(
					Wait().With("wait"),
					Complete().With("success"),
				),
			},
			expectedResults: []Result{
				Complete().With("success"),
				Wait().With("wait"),
				Complete().With("success"),
			},
			expectedStatus:       SWait,
			expectedRequeueAfter: 0,
			expectedMessage:      "success\nwait\nsuccess\n",
		},
		{
			desc: "contains a named result",
			rs: []Result{
				nameResult("aaa", Complete().With("success")),
				newAggregate(
					Wait().With("wait"),
					Complete().With("success"),
				),
			},
			expectedResults: []Result{
				nameResult("aaa", Complete().With("success")),
				Wait().With("wait"),
				Complete().With("success"),
			},
			expectedStatus:       SWait,
			expectedRequeueAfter: 0,
			expectedMessage:      "aaa: success\nwait\nsuccess\n",
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ar := newAggregate(c.rs...)
			assert.Equal(tt, c.expectedResults, ar.results(), c.desc)
			assert.Equal(tt, c.expectedStatus, ar.Status(), c.desc)
			assert.Equal(tt, c.expectedRequeueAfter, ar.RequeueAfter(), c.desc)
			assert.Equal(tt, c.expectedMessage, ar.Message(), c.desc)
		})
	}
}
