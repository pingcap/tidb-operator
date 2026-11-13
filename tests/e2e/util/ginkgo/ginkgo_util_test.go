// Copyright 2026 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package ginkgo

import "testing"

func TestMatchesFocus(t *testing.T) {
	for _, tt := range []struct {
		name     string
		patterns []string
		text     string
		want     bool
	}{
		{name: "no focus", text: "[Serial] restart"},
		{name: "empty focus", patterns: []string{""}, text: "[Serial] restart"},
		{name: "single match", patterns: []string{`\[Serial\]`}, text: "[Serial] restart", want: true},
		{name: "single mismatch", patterns: []string{`\[Stability\]`}, text: "[Serial] restart"},
		{name: "first pattern matches", patterns: []string{`\[Serial\]`, `\[Stability\]`}, text: "[Serial] restart", want: true},
		{name: "second pattern matches", patterns: []string{`\[Stability\]`, `\[Serial\]`}, text: "[Serial] restart", want: true},
		{name: "no pattern matches", patterns: []string{`\[Stability\]`, `\[Serial\]`}, text: "basic restart"},
		{name: "regular expression", patterns: []string{`^restart.*TiKV$`}, text: "restart one TiKV", want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesFocus(tt.text, tt.patterns); got != tt.want {
				t.Errorf("matchesFocus(%q, %q) = %v, want %v", tt.text, tt.patterns, got, tt.want)
			}
		})
	}
}
