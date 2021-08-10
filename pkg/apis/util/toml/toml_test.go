// Copyright 2020 PingCAP, Inc.
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

package toml

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestMarshal(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mp := map[string]interface{}{
		"int":    int64(1),
		"float":  1.1,
		"string": "string",
		"object": map[string]interface{}{
			"int":    int64(1),
			"float":  1.1,
			"string": "string",
		},
	}

	data, err := Marshal(&mp)
	g.Expect(err).Should(gomega.BeNil())

	var mpback map[string]interface{}
	err = Unmarshal(data, &mpback)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(mpback).Should(gomega.Equal(mp))
}

func TestEqual(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	type testcase struct {
		d1    string
		d2    string
		equal bool
	}

	tests := []*testcase{
		{
			d1:    "a = 1",
			d2:    "a = 1",
			equal: true,
		},
		{
			d1:    "a = 1",
			d2:    "a = 2",
			equal: false,
		},
		{
			d1:    "a =  1",
			d2:    "a = 1",
			equal: true,
		},
		{
			d1:    "a =  1",
			d2:    "a = 2",
			equal: false,
		},
		{
			d1:    "[user]\n[user.default]\np = 'ok'",
			d2:    "[user.default]\np = 'ok'",
			equal: true,
		},
	}

	for _, test := range tests {
		equal, err := Equal([]byte(test.d1), []byte(test.d2))
		g.Expect(err).Should(gomega.BeNil())
		t.Logf("check '%s' and '%s'", test.d1, test.d2)
		g.Expect(equal).Should(gomega.Equal(test.equal))
	}
}
