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

package v1alpha1

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestNGMonitoringImage(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name string

		image         string
		baseImage     string
		version       *string
		globalVersion *string
		expectImage   string
	}

	cases := []testcase{
		{
			name:          "should use baseImage and version firstly",
			image:         "image",
			baseImage:     "baseImage",
			version:       stringPointer("version"),
			globalVersion: stringPointer("globalVersion"),
			expectImage:   "baseImage:version",
		},
		{
			name:          "should use image if baseImage is empty",
			image:         "image",
			baseImage:     "",
			version:       stringPointer("version"),
			globalVersion: stringPointer("globalVersion"),
			expectImage:   "image",
		},
		{
			name:          "should use globalVersion if version is nil",
			image:         "image",
			baseImage:     "baseImage",
			version:       nil,
			globalVersion: stringPointer("globalVersion"),
			expectImage:   "baseImage:globalVersion",
		},
		{
			name:          "should not add tag if version and globalVersion is nil",
			image:         "image",
			baseImage:     "baseImage",
			version:       nil,
			globalVersion: nil,
			expectImage:   "baseImage",
		},
		{
			name:          "should not add tag if version is nil and globalVersion is empty",
			image:         "image",
			baseImage:     "baseImage",
			version:       nil,
			globalVersion: stringPointer(""),
			expectImage:   "baseImage",
		},
		{
			name:          "should not add tag if version is empty",
			image:         "image",
			baseImage:     "baseImage",
			version:       stringPointer(""),
			globalVersion: stringPointer("globalVersion"),
			expectImage:   "baseImage",
		},
	}

	for _, testcase := range cases {
		t.Logf("testcase: %s", testcase.name)

		tngm := &TidbNGMonitoring{}

		tngm.Spec.Version = testcase.globalVersion
		tngm.Spec.NGMonitoring.Version = testcase.version
		tngm.Spec.NGMonitoring.BaseImage = testcase.baseImage
		tngm.Spec.NGMonitoring.Image = testcase.image

		g.Expect(tngm.NGMonitoringImage()).Should(Equal(testcase.expectImage))
	}
}

func stringPointer(s string) *string {
	return &s
}
