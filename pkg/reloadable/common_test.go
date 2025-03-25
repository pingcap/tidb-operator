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

package reloadable

import (
	"testing"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

func TestRestartAnnotationsChanged(t *testing.T) {
	tests := []struct {
		name string
		a1   map[string]string
		a2   map[string]string
		want bool
	}{
		{
			name: "both nil",
			a1:   nil,
			a2:   nil,
			want: false,
		},
		{
			name: "both empty",
			a1:   map[string]string{},
			a2:   map[string]string{},
			want: false,
		},
		{
			name: "one empty, one with restart annotation",
			a1:   map[string]string{},
			a2:   map[string]string{metav1alpha1.RestartAnnotationPrefix + "key": "value"},
			want: true,
		},
		{
			name: "one nil, one with restart annotation",
			a1:   map[string]string{metav1alpha1.RestartAnnotationPrefix + "key": "value"},
			a2:   nil,
			want: true,
		},
		{
			name: "both with same restart annotation",
			a1:   map[string]string{metav1alpha1.RestartAnnotationPrefix + "key": "value"},
			a2:   map[string]string{metav1alpha1.RestartAnnotationPrefix + "key": "value"},
			want: false,
		},
		{
			name: "both with different restart annotations",
			a1:   map[string]string{metav1alpha1.RestartAnnotationPrefix + "key1": "value1"},
			a2:   map[string]string{metav1alpha1.RestartAnnotationPrefix + "key2": "value2"},
			want: true,
		},
		{
			name: "one with non-restart annotation",
			a1:   map[string]string{"non-restart-key": "value"},
			a2:   map[string]string{metav1alpha1.RestartAnnotationPrefix + "key": "value"},
			want: true,
		},
		{
			name: "both with same non-restart annotation",
			a1:   map[string]string{"non-restart-key": "value"},
			a2:   map[string]string{"non-restart-key": "value"},
			want: false,
		},
		{
			name: "same restart key with different values",
			a1:   map[string]string{metav1alpha1.RestartAnnotationPrefix + "key": "value1"},
			a2:   map[string]string{metav1alpha1.RestartAnnotationPrefix + "key": "value2"},
			want: true,
		},
		{
			name: "multiple restart keys with one differing value",
			a1: map[string]string{
				metav1alpha1.RestartAnnotationPrefix + "k1": "v1",
				metav1alpha1.RestartAnnotationPrefix + "k2": "v2",
			},
			a2: map[string]string{
				metav1alpha1.RestartAnnotationPrefix + "k1": "v1",
				metav1alpha1.RestartAnnotationPrefix + "k2": "v3",
			},
			want: true,
		},
		{
			name: "different count of restart annotations",
			a1:   map[string]string{metav1alpha1.RestartAnnotationPrefix + "k1": "v1"},
			a2: map[string]string{
				metav1alpha1.RestartAnnotationPrefix + "k1": "v1",
				metav1alpha1.RestartAnnotationPrefix + "k2": "v2",
			},
			want: true,
		},
		{
			name: "same restart keys but different non-restart annotations",
			a1: map[string]string{
				metav1alpha1.RestartAnnotationPrefix + "key": "value",
				"other1": "x",
			},
			a2: map[string]string{
				metav1alpha1.RestartAnnotationPrefix + "key": "value",
				"other2": "y",
			},
			want: false,
		},
		{
			name: "multiple same restart annotations",
			a1: map[string]string{
				metav1alpha1.RestartAnnotationPrefix + "k1": "v1",
				metav1alpha1.RestartAnnotationPrefix + "k2": "v2",
			},
			a2: map[string]string{
				metav1alpha1.RestartAnnotationPrefix + "k1": "v1",
				metav1alpha1.RestartAnnotationPrefix + "k2": "v2",
			},
			want: false,
		},
		{
			name: "same number of restart keys but different keys",
			a1: map[string]string{
				metav1alpha1.RestartAnnotationPrefix + "k1": "v1",
				metav1alpha1.RestartAnnotationPrefix + "k2": "v2",
			},
			a2: map[string]string{
				metav1alpha1.RestartAnnotationPrefix + "k1": "v1",
				metav1alpha1.RestartAnnotationPrefix + "k3": "v3",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := restartAnnotationsChanged(tt.a1, tt.a2); got != tt.want {
				t.Errorf("restartAnnotationsChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}
