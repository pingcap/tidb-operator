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

package datacontainer

import "testing"

func TestContainsString(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		s        string
		modifier func(string) string
		want     bool
	}{
		{
			name:     "empty slice",
			slice:    []string{},
			s:        "test",
			modifier: nil,
			want:     false,
		},
		{
			name:     "string exists",
			slice:    []string{"a", "b", "test", "c"},
			s:        "test",
			modifier: nil,
			want:     true,
		},
		{
			name:     "string does not exist",
			slice:    []string{"a", "b", "c"},
			s:        "test",
			modifier: nil,
			want:     false,
		},
		{
			name:  "with modifier - string exists",
			slice: []string{"A", "B", "TEST", "C"},
			s:     "test",
			modifier: func(s string) string {
				return s
			},
			want: false,
		},
		{
			name:  "with modifier - modified string exists",
			slice: []string{"A", "B", "TEST", "C"},
			s:     "test",
			modifier: func(s string) string {
				if s == "TEST" {
					return "test"
				}
				return s
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsString(tt.slice, tt.s, tt.modifier); got != tt.want {
				t.Errorf("ContainsString() = %v, want %v", got, tt.want)
			}
		})
	}
}
