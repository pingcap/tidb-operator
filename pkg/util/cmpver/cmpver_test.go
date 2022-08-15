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
	"testing"

	. "github.com/onsi/gomega"
)

type testcase struct {
	ver1   string
	op     Operation
	ver2   string
	expect bool
}

func TestCompare(t *testing.T) {
	g := NewGomegaWithT(t)

	cases := genTestCases()
	for _, testcase := range cases {
		t.Logf("testcase: %s %s %s should be %v", testcase.ver1, testcase.op, testcase.ver2, testcase.expect)

		ok, err := Compare(testcase.ver1, testcase.op, testcase.ver2)
		g.Expect(err).Should(Succeed())
		g.Expect(ok).Should(Equal(testcase.expect))
	}
}

func TestConstraint(t *testing.T) {
	t.Run("Check", func(t *testing.T) {
		g := NewGomegaWithT(t)

		cases := genTestCases()

		for _, testcase := range cases {
			t.Logf("testcase: %s %s %s should be %v", testcase.ver1, testcase.op, testcase.ver2, testcase.expect)

			c, err := NewConstraint(testcase.op, testcase.ver2)
			g.Expect(err).Should(Succeed())

			ok, err := c.Check(testcase.ver1)
			g.Expect(err).Should(Succeed())
			g.Expect(ok).Should(Equal(testcase.expect))
		}
	})
}

func genTestCases() []testcase {
	return []testcase{
		// Greater
		{"v5.3.1", Greater, "v5.1.2", true},
		{"5.3.1", Greater, "5.1.2", true},
		{"v5.3.1", Greater, "5.1.2", true},
		{"v5.3.1-master-dirty", Greater, "5.1.2-alpha-308-gbd21a6ea5", true},
		{"v5.1.2", Greater, "v5.3.1", false},
		{"v5.3.1", Greater, "v5.3.1", false},
		{"v5.3.1-dev12", Greater, "v5.1.2", true},
		{"v5.3.1-dev12", Greater, "v5.3.1", false},
		{"latest", Greater, "v5.3.1", true},
		{"nightly", Greater, "v5.3.1", true},
		// GreaterOrEqual
		{"v5.3.1", GreaterOrEqual, "v5.1.2", true},
		{"5.3.1", GreaterOrEqual, "5.1.2", true},
		{"5.3.1", GreaterOrEqual, "v5.1.2", true},
		{"5.3.1-master-dirty", GreaterOrEqual, "v5.1.2-alpha-308-gbd21a6ea5", true},
		{"v5.1.2", GreaterOrEqual, "v5.3.1", false},
		{"v5.3.1", GreaterOrEqual, "v5.3.1", true},
		{"v5.3.1-dev12", GreaterOrEqual, "v5.1.2", true},
		{"v5.3.1-dev12", GreaterOrEqual, "v5.3.1", true},
		{"latest", GreaterOrEqual, "v5.3.1", true},
		{"nightly", GreaterOrEqual, "v5.3.1", true},
		// Less
		{"v5.3.1", Less, "v5.1.2", false},
		{"v5.1.2", Less, "v5.3.1", true},
		{"5.1.2", Less, "5.3.1", true},
		{"5.1.2-alpha-308-gbd21a6ea5-dirty", Less, "v5.3.1", true},
		{"v5.3.1", Less, "v5.3.1", false},
		{"v5.3.1-dev12", Less, "v5.1.2", false},
		{"v5.3.1-dev12", Less, "v5.3.1", false},
		{"latest", Less, "v5.3.1", false},
		{"nightly", Less, "v5.3.1", false},
		// LessOrEqual
		{"v5.3.1", LessOrEqual, "v5.1.2", false},
		{"5.3.1", LessOrEqual, "5.1.2", false},
		{"v5.3.1", LessOrEqual, "5.1.2-alpha-308-gbd21a6ea5-dirty", false},
		{"v5.1.2", LessOrEqual, "v5.3.1", true},
		{"v5.3.1", LessOrEqual, "v5.3.1", true},
		{"v5.3.1-dev12", LessOrEqual, "v5.1.2", false},
		{"v5.3.1-dev12", LessOrEqual, "v5.3.1", true},
		{"latest", LessOrEqual, "v5.3.1", false},
		{"nightly", LessOrEqual, "v5.3.1", false},
	}
}
