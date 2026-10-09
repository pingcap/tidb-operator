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
	"fmt"

	"github.com/Masterminds/semver/v3"
)

func Check(v *semver.Version, constraints ...Constraints) bool {
	for _, c := range constraints {
		if !c.check(v) {
			return false
		}
	}

	return true
}

type Constraints interface {
	check(v *semver.Version) bool
}

type constraints struct {
	*semver.Constraints
}

func (c *constraints) check(v *semver.Version) bool {
	return c.Check(v)
}

// smoothUpgradeMinVersion is the minimum version that supports smooth upgrade DDL pause.
var smoothUpgradeMinVersion = MustNewConstraints(">= 7.5.0")

// SupportsSmoothUpgrade returns true if the given version string supports the smooth upgrade
// DDL pause/resume mechanism (requires TiDB >= v7.5.0).
func SupportsSmoothUpgrade(version string) bool {
	v, err := semver.NewVersion(version)
	if err != nil {
		return false
	}
	return Check(v, smoothUpgradeMinVersion)
}

func MustNewConstraints(expr string) Constraints {
	v, err := semver.NewConstraint(expr)
	if err != nil {
		panic(fmt.Errorf("cannot parse constraints %v: %w", expr, err))
	}

	v.IncludePrerelease = true

	return &constraints{
		Constraints: v,
	}
}
