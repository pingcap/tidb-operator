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

package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	tests := []struct {
		name    string
		setStr  string
		wantStr string
	}{
		{
			name:    "set multiple features",
			setStr:  "a=true,b=false",
			wantStr: "NativeVolumeModifying=false,a=true,b=false",
		},
		{
			name:    "set multiple features in different order",
			setStr:  "b=False,a=True",
			wantStr: "NativeVolumeModifying=false,a=true,b=false",
		},
		{
			name:    "overwrite the default feature",
			setStr:  "NativeVolumeModifying=true",
			wantStr: "NativeVolumeModifying=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gates := NewDefaultFeatureGate()
			assert.NoError(t, gates.Set(tt.setStr))
			assert.Equal(t, tt.wantStr, gates.String())
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
			assert.Equal(t, tt.wantStr, gates.String())
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
				assert.Equal(t, want, gates.Enabled(k))
			}
		})
	}
}
