// Copyright 2020 PingCAP, Inc.
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

package features

import "testing"

func TestSet(t *testing.T) {
	tests := []struct {
		name    string
		setStr  string
		wantStr string
	}{
		{
			name:    "set multiple features",
			setStr:  "a=true,b=false",
			wantStr: "a=true,b=false",
		},
		{
			name:    "set multiple features",
			setStr:  "a=True,b=False",
			wantStr: "a=true,b=false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gates := NewFeatureGate()
			gates.Set(tt.setStr)
			got := gates.String()
			if got != tt.wantStr {
				t.Errorf("want: %s, got %s", tt.wantStr, got)
			}
		})
	}
}

func TestSetFromMap(t *testing.T) {
	tests := []struct {
		name    string
		setMap  map[string]bool
		wantStr string
	}{
		{
			name: "set multiple features",
			setMap: map[string]bool{
				"a": true,
				"b": false,
			},
			wantStr: "a=true,b=false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gates := NewFeatureGate()
			gates.SetFromMap(tt.setMap)
			got := gates.String()
			if got != tt.wantStr {
				t.Errorf("want: %s, got %s", tt.wantStr, got)
			}
		})
	}
}

func TestEnabled(t *testing.T) {
	tests := []struct {
		name        string
		setMap      map[string]bool
		wantEnabled map[string]bool
	}{
		{
			name: "set multiple features",
			setMap: map[string]bool{
				"a": true,
				"b": false,
			},
			wantEnabled: map[string]bool{
				"a": true,
				"b": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gates := NewFeatureGate()
			gates.SetFromMap(tt.setMap)
			for k, want := range tt.wantEnabled {
				got := gates.Enabled(k)
				if got != want {
					t.Errorf("[feature: %s] want %v, got %v", k, want, got)
				}
			}
		})
	}
}
