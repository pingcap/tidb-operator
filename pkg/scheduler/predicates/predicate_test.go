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

package predicates

import (
	"testing"

	. "github.com/onsi/gomega"
	apiv1 "k8s.io/api/core/v1"
)

func TestGetNodeFromNames(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		nodes     []apiv1.Node
		nodeNames []string
		expected  []string
	}

	testFn := func(i int, test *testcase, t *testing.T) {
		t.Log(i)
		arr := getNodeFromNames(test.nodes, test.nodeNames)
		g.Expect(len(arr)).To(Equal(len(test.expected)))
		for idx, node := range arr {
			g.Expect(node.GetName()).To(Equal(test.expected[idx]))
		}
	}

	tests := []testcase{
		{
			nodes:     fakeThreeNodes(),
			nodeNames: []string{},
			expected:  []string{},
		},
		{
			nodes:     fakeThreeNodes(),
			nodeNames: []string{"kube-node-1"},
			expected:  []string{"kube-node-1"},
		},
		{
			nodes:     fakeThreeNodes(),
			nodeNames: []string{"kube-node-1", "kube-node-2"},
			expected:  []string{"kube-node-1", "kube-node-2"},
		},
		{
			nodes:     fakeThreeNodes(),
			nodeNames: []string{"kube-node-1", "kube-node-2", "kube-node-3"},
			expected:  []string{"kube-node-1", "kube-node-2", "kube-node-3"},
		},
		{
			nodes:     fakeTwoNodes(),
			nodeNames: []string{"kube-node-1", "kube-node-2", "kube-node-3"},
			expected:  []string{"kube-node-1", "kube-node-3"},
		},
		{
			nodes:     fakeTwoNodes(),
			nodeNames: []string{"kube-node-1", "kube-node-3"},
			expected:  []string{"kube-node-1", "kube-node-3"},
		},
		{
			nodes:     fakeTwoNodes(),
			nodeNames: []string{"kube-node-1"},
			expected:  []string{"kube-node-1"},
		},
		{
			nodes:     fakeTwoNodes(),
			nodeNames: []string{"kube-node-3"},
			expected:  []string{"kube-node-3"},
		},
		{
			nodes:     fakeTwoNodes(),
			nodeNames: []string{"kube-node-2"},
			expected:  []string{},
		},
		{
			nodes:     fakeOneNode(),
			nodeNames: []string{"kube-node-1", "kube-node-2", "kube-node-3"},
			expected:  []string{"kube-node-3"},
		},
		{
			nodes:     fakeOneNode(),
			nodeNames: []string{"kube-node-2", "kube-node-3"},
			expected:  []string{"kube-node-3"},
		},
		{
			nodes:     fakeOneNode(),
			nodeNames: []string{"kube-node-1", "kube-node-3"},
			expected:  []string{"kube-node-3"},
		},
		{
			nodes:     fakeOneNode(),
			nodeNames: []string{"kube-node-1", "kube-node-2"},
			expected:  []string{},
		},
		{
			nodes:     fakeZeroNode(),
			nodeNames: []string{"kube-node-1", "kube-node-2", "kube-node-3"},
			expected:  []string{},
		},
		{
			nodes:     fakeZeroNode(),
			nodeNames: []string{"kube-node-2", "kube-node-3"},
			expected:  []string{},
		},
		{
			nodes:     fakeZeroNode(),
			nodeNames: []string{"kube-node-3"},
			expected:  []string{},
		},
	}

	for i := range tests {
		testFn(i, &tests[i], t)
	}
}
