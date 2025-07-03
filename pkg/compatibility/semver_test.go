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

package compatibility

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	cases := []struct {
		desc            string
		constraints     []Constraints
		allowedVersions []string
		blockedVersions []string
	}{
		{
			desc: "higher than or equal a version",
			constraints: []Constraints{
				MustNewConstraints(">= v8.3.0"),
			},
			allowedVersions: []string{
				"v8.3.0",
				"v8.3.1",
				"v8.3.1-alpha.0",
				"v8.4.0",
				"v8.4.0-alpha.0",
				"v9.0.0",
				"v9.0.0-alpha.0",
			},
			blockedVersions: []string{
				"v8.3.0-alpha.0",
				"v8.2.10",
				"v7.4.0",
				"v7.4.10",
			},
		},
		{
			desc: "allow in some release versions",
			constraints: []Constraints{
				MustNewConstraints(">= v8.3.0 || ^v7.5.6 || ^v6.8.10 "),
			},
			allowedVersions: []string{
				"v7.5.6",
				"v7.5.7-alpha.0",
				"v7.5.7",
				"v7.6.0",
				"v7.6.0-beta.0",
				"v6.8.10",
				"v6.8.11-alpha.0",
				"v6.8.11",
				"v6.9.0",
			},
			blockedVersions: []string{
				"v8.0.0",
				"v8.3.0-alpha.0",
				"v8.2.10",
				"v7.5.6-alpha.0",
				"v7.5.5",
				"v7.4.0",
				"v7.4.10",
				"v7.0.0",
				"v6.8.10-alpha.0",
				"v6.7.0",
				"v6.0.0",
				"v5.100.0",
			},
		},
	}
	for i := range cases {
		c := &cases[i]
		for _, v := range c.allowedVersions {
			t.Run(c.desc+"_allowed_"+v, func(tt *testing.T) {
				tt.Parallel()
				ver := semver.MustParse(v)
				assert.True(tt, Check(ver, c.constraints...))
			})
		}
		for _, v := range c.blockedVersions {
			t.Run(c.desc+"_blocked_"+v, func(tt *testing.T) {
				tt.Parallel()
				ver := semver.MustParse(v)
				assert.False(tt, Check(ver, c.constraints...))
			})
		}
	}
}
