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
	"k8s.io/utils/ptr"
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
			desc:     "image only digest",
			image:    "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expected: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:     "image without tag and digest",
			image:    "test",
			version:  "test",
			expected: "test:test",
		},
		{
			desc:     "cannot override image with tag",
			image:    "test:xxx",
			version:  "test",
			expected: "test:xxx",
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
		img      *string
		version  string
		expected string
		panic    bool
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
		{
			desc:     "pd override image",
			m:        PD,
			img:      ptr.To("pd"),
			version:  "test",
			expected: "pd:test",
		},
		{
			desc:     "pd specifies a specific tag",
			m:        PD,
			img:      ptr.To("pd:mmm"),
			version:  "test",
			expected: "pd:mmm",
		},
		{
			desc:    "invalid version",
			m:       PD,
			img:     ptr.To("pd"),
			version: "-test_xxx",
			panic:   true,
		},
	}
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()
			if c.panic {
				assert.Panics(tt, func() {
					c.m.Image(c.img, c.version)
				}, c.desc)
			} else {
				assert.Equal(tt, c.expected, c.m.Image(c.img, c.version), c.desc)
			}
		})
	}
}

func TestTagged(t *testing.T) {
	cases := []struct {
		desc     string
		m        Tagged
		img      *string
		expected string
	}{
		{
			desc:     "prestop checker default",
			m:        PrestopChecker,
			expected: "pingcap/tidb-operator-prestop-checker:latest",
		},
		{
			desc:     "override prestop checker",
			m:        PrestopChecker,
			img:      ptr.To("test"),
			expected: "test",
		},
	}
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()
			assert.Equal(tt, c.expected, c.m.Image(c.img), c.desc)
		})
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		desc   string
		img    string
		hasErr bool
	}{
		{
			desc: "pd default",
			img:  "pingcap/pd",
		},
		{
			desc:   "invalid img",
			img:    "pingcap/pd:-xx_xxx",
			hasErr: true,
		},
	}
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()
			err := Validate(c.img)
			if c.hasErr {
				assert.Error(tt, err, c.desc)
			}
		})
	}
}
