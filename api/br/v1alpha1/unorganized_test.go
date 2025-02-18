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

package v1alpha1

import (
	"testing"
	"time"
)

func TestParseTSString(t *testing.T) {
	tests := []struct {
		name    string
		ts      string
		want    uint64
		wantErr bool
	}{
		{
			name:    "empty string",
			ts:      "",
			want:    0,
			wantErr: false,
		},
		{
			name:    "valid TSO",
			ts:      "400036290571534337",
			want:    400036290571534337,
			wantErr: false,
		},
		{
			name:    "valid datetime",
			ts:      "2006-01-02 15:04:05",
			want:    GoTimeToTS(time.Date(2006, 1, 2, 15, 4, 5, 0, time.Local)),
			wantErr: false,
		},
		{
			name:    "valid RFC3339",
			ts:      "2006-01-02T15:04:05Z",
			want:    GoTimeToTS(time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)),
			wantErr: false,
		},
		{
			name:    "invalid format",
			ts:      "invalid",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTSString(tt.ts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTSString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseTSString() = %v, want %v", got, tt.want)
			}
		})
	}
}
