// Copyright 2025 PingCAP, Inc.
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

package volumes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
)

func TestHasVAC(t *testing.T) {
	testcases := []struct {
		volumes  []DesiredVolume
		expected bool
	}{
		{
			volumes:  []DesiredVolume{},
			expected: false,
		},
		{
			volumes: []DesiredVolume{
				{
					VolumeAttributesClassName: nil,
				},
			},
			expected: false,
		},
		{
			volumes: []DesiredVolume{
				{
					VolumeAttributesClassName: pointer.String(""),
				},
			},
			expected: false,
		},
		{
			volumes: []DesiredVolume{
				{
					VolumeAttributesClassName: pointer.String("vac-1"),
				},
			},
			expected: true,
		},
		{
			volumes: []DesiredVolume{
				{
					VolumeAttributesClassName: nil,
				},
				{
					VolumeAttributesClassName: pointer.String("vac-1"),
				},
				{
					VolumeAttributesClassName: pointer.String(""),
				},
			},
			expected: true,
		},
	}

	for _, tt := range testcases {
		actual := hasVAC(tt.volumes)
		assert.Equal(t, tt.expected, actual)
	}
}
