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

package compare

import (
	"testing"
)

func TestSetIfChanged(t *testing.T) {
	t.Run("string tests", func(t *testing.T) {
		var (
			empty = ""
			hello = "hello"
			world = "world"
		)

		tests := []struct {
			name     string
			dst      *string
			src      string
			expected bool
			want     string
		}{
			{
				name:     "same value",
				dst:      &hello,
				src:      hello,
				expected: false,
				want:     hello,
			},
			{
				name:     "different values",
				dst:      &hello,
				src:      world,
				expected: true,
				want:     world,
			},
			{
				name:     "empty vs non-empty",
				dst:      &empty,
				src:      hello,
				expected: true,
				want:     hello,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual := SetIfChanged(tt.dst, tt.src)
				if actual != tt.expected {
					t.Errorf("SetIfChanged() = %v, want %v", actual, tt.expected)
				}
				if *tt.dst != tt.want {
					t.Errorf("After SetIfChanged(), dst = %v, want %v", *tt.dst, tt.want)
				}
			})
		}
	})

	t.Run("int tests", func(t *testing.T) {
		var (
			zero   = 0
			ten    = 10
			twenty = 20
		)

		tests := []struct {
			name     string
			dst      *int
			src      int
			expected bool
			want     int
		}{
			{
				name:     "same value",
				dst:      &ten,
				src:      ten,
				expected: false,
				want:     ten,
			},
			{
				name:     "different values",
				dst:      &ten,
				src:      twenty,
				expected: true,
				want:     twenty,
			},
			{
				name:     "zero vs non-zero",
				dst:      &zero,
				src:      ten,
				expected: true,
				want:     ten,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual := SetIfChanged(tt.dst, tt.src)
				if actual != tt.expected {
					t.Errorf("SetIfChanged() = %v, want %v", actual, tt.expected)
				}
				if *tt.dst != tt.want {
					t.Errorf("After SetIfChanged(), dst = %v, want %v", *tt.dst, tt.want)
				}
			})
		}
	})
}

func TestNewAndSetIfChanged(t *testing.T) {
	t.Run("int tests", func(t *testing.T) {
		var (
			nilInt *int
			zero   = 0
			ten    = 10
			twenty = 20
		)

		tests := []struct {
			name     string
			dst      *int
			src      int
			expected bool
			want     int
		}{
			{
				name:     "nil destination",
				dst:      nilInt,
				src:      ten,
				expected: true,
				want:     ten,
			},
			{
				name:     "same value",
				dst:      &ten,
				src:      ten,
				expected: false,
				want:     ten,
			},
			{
				name:     "different values",
				dst:      &ten,
				src:      twenty,
				expected: true,
				want:     twenty,
			},
			{
				name:     "zero vs non-zero",
				dst:      &zero,
				src:      ten,
				expected: true,
				want:     ten,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create a copy of the destination pointer to avoid modifying the original
				var dst *int
				if tt.dst != nil {
					val := *tt.dst
					dst = &val
				}

				actual := NewAndSetIfChanged(&dst, tt.src)
				if actual != tt.expected {
					t.Errorf("NewAndSetIfChanged() = %v, want %v", actual, tt.expected)
				}
				if dst == nil {
					t.Errorf("After NewAndSetIfChanged(), dst is nil, want non-nil")
				} else if *dst != tt.want {
					t.Errorf("After NewAndSetIfChanged(), dst = %v, want %v", *dst, tt.want)
				}
			})
		}
	})

	t.Run("string tests", func(t *testing.T) {
		var (
			nilString *string
			empty     = ""
			hello     = "hello"
			world     = "world"
		)

		tests := []struct {
			name     string
			dst      *string
			src      string
			expected bool
			want     string
		}{
			{
				name:     "nil destination",
				dst:      nilString,
				src:      hello,
				expected: true,
				want:     hello,
			},
			{
				name:     "same value",
				dst:      &hello,
				src:      hello,
				expected: false,
				want:     hello,
			},
			{
				name:     "different values",
				dst:      &hello,
				src:      world,
				expected: true,
				want:     world,
			},
			{
				name:     "empty vs non-empty",
				dst:      &empty,
				src:      hello,
				expected: true,
				want:     hello,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create a copy of the destination pointer to avoid modifying the original
				var dst *string
				if tt.dst != nil {
					val := *tt.dst
					dst = &val
				}

				actual := NewAndSetIfChanged(&dst, tt.src)
				if actual != tt.expected {
					t.Errorf("NewAndSetIfChanged() = %v, want %v", actual, tt.expected)
				}
				if dst == nil {
					t.Errorf("After NewAndSetIfChanged(), dst is nil, want non-nil")
				} else if *dst != tt.want {
					t.Errorf("After NewAndSetIfChanged(), dst = %v, want %v", *dst, tt.want)
				}
			})
		}
	})
}
