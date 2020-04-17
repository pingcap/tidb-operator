// Copyright 2018 PingCAP, Inc.
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

func TestGetImageTag(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name   string
		image  string
		expect string
	}

	tests := []*testcase{
		{
			name:   "with repo",
			image:  "localhost:5000/tikv:v3.1.0",
			expect: "v3.1.0",
		},
		{
			name:   "no colon",
			image:  "tikv",
			expect: "",
		},
		{
			name:   "no repo",
			image:  "tikv:nightly",
			expect: "nightly",
		},
		{
			name:   "start with colon",
			image:  ":v4.0.0",
			expect: "v4.0.0",
		},
		{
			name:   "end with colon",
			image:  "tikv:",
			expect: "",
		},
		{
			name:   "only colon",
			image:  ":",
			expect: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version := GetImageTag(test.image)
			g.Expect(version).To(Equal(test.expect))
		})
	}
}
