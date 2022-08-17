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

package v2

import (
	"regexp"
	"strings"
	"testing"

	"github.com/onsi/gomega"
	"mvdan.cc/sh/v3/syntax"
)

func TestScriptFormat(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	scripts := []string{
		componentCommonScript,
		pdStartScript,
		pdStartSubScript,
		pumpStartScript,
		pumpStartSubScript,
		ticdcStartScript,
		ticdcStartSubScript,
		tidbStartScript,
		tidbStartSubScript,
		tiflashInitScript,
		tiflashInitSubScript,
		tiflashStartScript,
		tiflashStartSubScript,
		tikvStartScript,
		tikvStartSubScript,
	}

	blankLineRegexp := regexp.MustCompile(`^\s*$`)

	for _, script := range scripts {
		for _, line := range strings.Split(script, "\n") {
			g.Expect(line).ShouldNot(gomega.HaveSuffix(" "))
			g.Expect(line).ShouldNot(gomega.ContainSubstring("\t"), "line should use space not tab")
			if blankLineRegexp.MatchString(line) {
				g.Expect(line).ShouldNot(gomega.ContainSubstring(" "), "blank line should not contain space")
			}
		}
	}
}

func validateScript(script string) error {
	_, err := syntax.NewParser().Parse(strings.NewReader(script), "")
	return err
}
