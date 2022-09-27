// Copyright 2021 PingCAP, Inc.
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

package cmpver

import (
	"fmt"

	semver "github.com/Masterminds/semver"
)

type Operation string

const (
	Greater        Operation = ">"
	GreaterOrEqual Operation = ">="
	Less           Operation = "<"
	LessOrEqual    Operation = "<="
)

// Compare compare ver1 and ver2 by operation
//
// For example:
//
//	Compare(ver1, Greater, ver2) is same as "ver1 > ver2"
//
// Dirty versions and pre versions are regarded as standard version.
// For example: 'v5.1.2-dev' and 'v5.1.2-betav1' are regarded as 'v5.1.2'.
//
// Latest or nightly version is larger than any version
func Compare(ver1 string, op Operation, ver2 string) (bool, error) {
	err := validateOperation(op)
	if err != nil {
		return false, err
	}

	c, err := NewConstraint(op, ver2)
	if err != nil {
		return false, err
	}

	return c.Check(ver1)
}

// CompareByStr compare ver1 and ver2
//
// For example:
//
//	CompareByStr(ver1, ">", ver2) is same as "ver1 > ver2"
//
// Dirty versions and pre versions are regarded as standard version.
// For example: 'v5.1.2-dev' and 'v5.1.2-betav1' are regarded as 'v5.1.2'.
//
// Latest or nightly version is larger than any version
func CompareByStr(ver1, opstr, ver2 string) (bool, error) {
	op := Operation(opstr)
	return Compare(ver1, op, ver2)
}

type Constraint struct {
	op         Operation
	version    string
	constraint *semver.Constraints
}

func NewConstraint(op Operation, version string) (*Constraint, error) {
	err := validateOperation(op)
	if err != nil {
		return nil, err
	}

	// default to support dirty version
	constraint, err := semver.NewConstraint(fmt.Sprintf("%s%s", op, version))
	if err != nil {
		return nil, err
	}

	return &Constraint{
		op:         Operation(op),
		version:    version,
		constraint: constraint,
	}, nil
}

// Check tests if a version satisfies the constraints.
//
// Dirty versions and pre versions are regarded as standard version.
// For example: 'v5.1.2-dev' and 'v5.1.2-betav1' are regarded as 'v5.1.2'.
//
// Latest or nightly version is larger than any version
func (c *Constraint) Check(version string) (bool, error) {
	if isLatest(version) {
		return compareLatest(c.op), nil
	}

	ver, err := semver.NewVersion(version)
	if err != nil {
		return false, err
	}

	// set prerelease to empty str in order to support dirty version
	ckVer := *ver
	if ver.Prerelease() != "" {
		ckVer, err = ver.SetPrerelease("")
		if err != nil {
			return false, err
		}
	}

	return c.constraint.Check(&ckVer), nil
}

func validateOperation(op Operation) error {
	switch op {
	case Greater, GreaterOrEqual, Less, LessOrEqual:
		return nil
	}

	return fmt.Errorf("not support operation %s", op)
}

func isLatest(version string) bool {
	switch version {
	case "latest", "nightly":
		return true
	}

	return false
}

func compareLatest(op Operation) bool {
	// latest is greater than any version
	switch op {
	case Greater, GreaterOrEqual:
		return true
	case Less, LessOrEqual:
		return false
	}

	return false
}
