// Copyright 2019 PingCAP, Inc.
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

func TestMakeDockerSocketMount(t *testing.T) {
	g := NewGomegaWithT(t)
	type testCase struct {
		socket   string
		readOnly bool
	}
	tests := []testCase{
		{
			socket:   "/var/run/docker.sock",
			readOnly: false,
		},
		{
			socket:   "/test.sock",
			readOnly: true,
		},
		{
			socket:   "/test.sock",
			readOnly: false,
		},
	}
	testFn := func(test testCase, g *GomegaWithT) {
		volume, mount := MakeDockerSocketMount(test.socket, test.readOnly)
		g.Expect(volume.HostPath).NotTo(BeNil())
		g.Expect(volume.HostPath.Path).To(Equal(test.socket))
		g.Expect(mount.MountPath).To(Equal(DockerSocket))
		g.Expect(mount.ReadOnly).To(Equal(test.readOnly))
	}
	for i := range tests {
		testFn(tests[i], g)
	}
}

func TestGetTidbServiceName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(GetTidbServiceName("demo")).To(Equal("demo-tidb"))
}
