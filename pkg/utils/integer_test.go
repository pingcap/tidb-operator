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

package utils

import (
	"math"
	"testing"

	"k8s.io/utils/ptr"
)

type testCase[T Integer] struct {
	name     string
	a        *T
	b        *T
	expected bool
}

func runTests[T Integer](t *testing.T, cases []testCase[T]) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := ValuesDiffer(tc.a, tc.b)
			if actual != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, actual)
			}
		})
	}
}

func TestValuesDiffer(t *testing.T) {
	t.Run("int32 tests", func(t *testing.T) {
		var (
			zero   int32 = 0
			ten    int32 = 10
			twenty int32 = 20
		)

		cases := []testCase[int32]{
			{
				name:     "both nil",
				a:        nil,
				b:        nil,
				expected: false,
			},
			{
				name:     "a nil",
				a:        nil,
				b:        &ten,
				expected: false,
			},
			{
				name:     "b nil",
				a:        &ten,
				b:        nil,
				expected: false,
			},
			{
				name:     "same value",
				a:        &ten,
				b:        &ten,
				expected: false,
			},
			{
				name:     "different values",
				a:        &ten,
				b:        &twenty,
				expected: true,
			},
			{
				name:     "zero vs non-zero",
				a:        &zero,
				b:        &ten,
				expected: true,
			},
		}

		runTests(t, cases)
	})

	t.Run("int64 tests", func(t *testing.T) {
		var (
			zero   int64 = 0
			ten    int64 = 10
			twenty int64 = 20
		)

		cases := []testCase[int64]{
			{
				name:     "different values large",
				a:        &ten,
				b:        &twenty,
				expected: true,
			},
			{
				name:     "same large value",
				a:        &twenty,
				b:        &twenty,
				expected: false,
			},
			{
				name:     "zero vs max",
				a:        &zero,
				b:        ptr.To[int64](math.MaxInt64),
				expected: true,
			},
		}

		runTests(t, cases)
	})
}
