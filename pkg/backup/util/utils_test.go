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
		name      string
		image     string
		imageName string
		tag       string
	}

	tests := []*testcase{
		{
			name:      "with repo",
			image:     "localhost:5000/tikv:v3.1.0",
			imageName: "localhost:5000/tikv",
			tag:       "v3.1.0",
		},
		{
			name:      "no colon",
			image:     "tikv",
			imageName: "tikv",
			tag:       "",
		},
		{
			name:      "no repo",
			image:     "tikv:nightly",
			imageName: "tikv",
			tag:       "nightly",
		},
		{
			name:      "start with colon",
			image:     ":v4.0.0",
			imageName: "",
			tag:       "v4.0.0",
		},
		{
			name:      "end with colon",
			image:     "tikv:",
			imageName: "tikv",
			tag:       "",
		},
		{
			name:      "only colon",
			image:     ":",
			imageName: "",
			tag:       "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name, version := ParseImage(test.image)
			g.Expect(version).To(Equal(test.tag))
			g.Expect(name).To(Equal(test.imageName))
		})
	}
}
