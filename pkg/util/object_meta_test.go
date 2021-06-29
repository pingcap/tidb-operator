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

package util

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestCombineStringMap(t *testing.T) {
	g := NewGomegaWithT(t)

	a := map[string]string{
		"a": "av",
	}
	origA := map[string]string{
		"a": "av",
	}
	b := map[string]string{
		"b": "bv",
	}
	c := map[string]string{
		"c": "cv",
	}
	dropped := map[string]string{
		"a": "aov",
	}

	expect1 := map[string]string{
		"a": "av",
		"b": "bv",
		"c": "cv",
	}
	expect2 := map[string]string{
		"a": "aov",
		"b": "bv",
		"c": "cv",
	}

	res := CombineStringMap(a, b, c, dropped)
	g.Expect(res).Should(Equal(expect1))
	g.Expect(a).Should(Equal(origA))

	res = CombineStringMap(nil, b, c, dropped)
	g.Expect(res).Should(Equal(expect2))

}

func TestCopyStringMap(t *testing.T) {
	g := NewGomegaWithT(t)

	src := map[string]string{
		"a": "av",
	}

	// modify res and check src unchanged
	res := CopyStringMap(src)
	res["test"] = "v"
	res["a"] = "overwrite"
	g.Expect(src).Should(Equal(map[string]string{"a": "av"}))
}
