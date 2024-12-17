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

package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithVersion(t *testing.T) {
	cases := []struct {
		desc     string
		image    string
		version  string
		expected string
		hasErr   bool
	}{
		{
			desc:   "image is empty",
			hasErr: true,
		},
		{
			desc:   "invalid image",
			image:  "test:test:test",
			hasErr: true,
		},
		{
			desc:     "image with tag",
			image:    "test:test",
			expected: "test:test",
		},
		{
			desc:     "image with digest",
			image:    "test@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expected: "test@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:     "image with tag and digest",
			image:    "test:test@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expected: "test:test@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:     "image without tag and digest",
			image:    "test",
			version:  "test",
			expected: "test:test",
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()
			s, err := withVersion(c.image, c.version)
			if c.hasErr {
				require.Error(tt, err, c.desc)
			}
			assert.Equal(tt, c.expected, s, c.desc)
		})
	}
}

func TestUntagged(t *testing.T) {
	cases := []struct {
		desc     string
		m        Untagged
		version  string
		expected string
	}{
		{
			desc:     "pd default",
			m:        PD,
			version:  "test",
			expected: "pingcap/pd:test",
		},
		{
			desc:     "tidb default",
			m:        TiDB,
			version:  "test",
			expected: "pingcap/tidb:test",
		},
		{
			desc:     "tikv default",
			m:        TiKV,
			version:  "test",
			expected: "pingcap/tikv:test",
		},
		{
			desc:     "tiflash default",
			m:        TiFlash,
			version:  "test",
			expected: "pingcap/tiflash:test",
		},
	}
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()
			assert.Equal(tt, c.expected, c.m.Image(nil, c.version), c.desc)
		})
	}
}
