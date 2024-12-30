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

package maputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMerge(t *testing.T) {
	cases := []struct {
		desc     string
		maps     []map[string]string
		expected map[string]string
	}{
		{
			desc:     "nil",
			maps:     nil,
			expected: map[string]string{},
		},
		{
			desc: "overwrite",
			maps: []map[string]string{
				{
					"aa": "aa",
				},
				{
					"bb": "bb",
				},
				{
					"aa": "cc",
				},
			},
			expected: map[string]string{
				"aa": "cc",
				"bb": "bb",
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			res := Merge(c.maps...)
			assert.Equal(tt, c.expected, res)
		})
	}
}

func TestCopy(t *testing.T) {
	cases := []struct {
		desc     string
		original map[string]string
		expected map[string]string
	}{
		{
			desc:     "nil",
			original: nil,
			expected: nil,
		},
		{
			desc:     "empty",
			original: map[string]string{},
			expected: map[string]string{},
		},
		{
			desc: "normal",
			original: map[string]string{
				"aa": "aa",
				"bb": "bb",
			},
			expected: map[string]string{
				"aa": "aa",
				"bb": "bb",
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			res := Copy(c.original)
			assert.Equal(tt, c.expected, res)
		})
	}
}

func TestAreEqual(t *testing.T) {
	cases := []struct {
		desc     string
		map1     map[string]string
		map2     map[string]string
		expected bool
	}{
		{
			desc:     "nil",
			map1:     nil,
			map2:     nil,
			expected: true,
		},
		{
			desc:     "empty",
			map1:     map[string]string{},
			map2:     map[string]string{},
			expected: true,
		},
		{
			desc: "equal",
			map1: map[string]string{
				"aa": "aa",
				"bb": "bb",
			},
			map2: map[string]string{
				"aa": "aa",
				"bb": "bb",
			},
			expected: true,
		},
		{
			desc: "not equal",
			map1: map[string]string{
				"aa": "aa",
				"bb": "bb",
			},
			map2: map[string]string{
				"aa": "aa",
				"bb": "cc",
			},
			expected: false,
		},
		{
			desc: "not equal",
			map1: map[string]string{
				"aa": "aa",
				"bb": "bb",
			},
			map2: map[string]string{
				"aa": "aa",
			},
			expected: false,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			res := AreEqual(c.map1, c.map2)
			assert.Equal(tt, c.expected, res)
		})
	}
}
