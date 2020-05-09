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

func TestGetNodeFromTopologies(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name        string
		nodes       []apiv1.Node
		topologyKey string
		topologies  []string
		expected    []string
	}

	testFn := func(i int, test *testcase, t *testing.T) {
		t.Log(i)
		t.Logf("name: %s, topologies: %s, expected: %s", test.name, test.topologies, test.expected)
		arr := getNodeFromTopologies(test.nodes, test.topologyKey, test.topologies)
		g.Expect(len(arr)).To(Equal(len(test.expected)))
		for idx, node := range arr {
			g.Expect(node.GetName()).To(Equal(test.expected[idx]))
		}
	}

	tests := []testcase{
		{
			name:        "topologyKey: kubernetes.io/hostname, three nodes, return zero node",
			nodes:       fakeThreeNodes(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{},
			expected:    []string{},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, three nodes, return one node",
			nodes:       fakeThreeNodes(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-1"},
			expected:    []string{"kube-node-1"},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, three nodes, return two nodes",
			nodes:       fakeThreeNodes(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-1", "kube-node-2"},
			expected:    []string{"kube-node-1", "kube-node-2"},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, three nodes, return three nodes",
			nodes:       fakeThreeNodes(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-1", "kube-node-2", "kube-node-3"},
			expected:    []string{"kube-node-1", "kube-node-2", "kube-node-3"},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, two nodes, return two nodes",
			nodes:       fakeTwoNodes(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-1", "kube-node-2", "kube-node-3"},
			expected:    []string{"kube-node-1", "kube-node-2"},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, two nodes, return one node",
			nodes:       fakeTwoNodes(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-1", "kube-node-3"},
			expected:    []string{"kube-node-1"},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, one node, return one node",
			nodes:       fakeTwoNodes(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-1"},
			expected:    []string{"kube-node-1"},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, one nodes, return zero node",
			nodes:       fakeTwoNodes(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-3"},
			expected:    []string{},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, one node, return one node",
			nodes:       fakeTwoNodes(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-2"},
			expected:    []string{"kube-node-2"},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, three nodes, return one node",
			nodes:       fakeOneNode(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-1", "kube-node-2", "kube-node-3"},
			expected:    []string{"kube-node-1"},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, two nodes, return zero node",
			nodes:       fakeOneNode(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-2", "kube-node-3"},
			expected:    []string{},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, two nodes, return one node",
			nodes:       fakeOneNode(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-1", "kube-node-3"},
			expected:    []string{"kube-node-1"},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, two node, return one node",
			nodes:       fakeOneNode(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-1", "kube-node-2"},
			expected:    []string{"kube-node-1"},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, three nodes, return zero node",
			nodes:       fakeZeroNode(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-1", "kube-node-2", "kube-node-3"},
			expected:    []string{},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, two nodes, return zero node",
			nodes:       fakeZeroNode(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-2", "kube-node-3"},
			expected:    []string{},
		},
		{
			name:        "topologyKey: kubernetes.io/hostname, one node, return zero node",
			nodes:       fakeZeroNode(),
			topologyKey: "kubernetes.io/hostname",
			topologies:  []string{"kube-node-3"},
			expected:    []string{},
		},
	}

	for i := range tests {
		testFn(i, &tests[i], t)
	}
}
