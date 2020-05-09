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
	apiv1 "k8s.io/api/core/v1"
)

// Predicate is an interface as extender-implemented predicate functions
type Predicate interface {
	// Name return the predicate name
	Name() string

	// Filter function receives a set of nodes and returns a set of candidate nodes.
	Filter(string, *apiv1.Pod, []apiv1.Node) ([]apiv1.Node, error)
}

func getNodeFromTopologies(nodes []apiv1.Node, topologyKey string, topologies []string) []apiv1.Node {
	var retNodes []apiv1.Node
	for _, node := range nodes {
		if _, ok := node.Labels[topologyKey]; !ok {
			continue
		}
		for _, topology := range topologies {
			if node.Labels[topologyKey] == topology {
				retNodes = append(retNodes, node)
				break
			}
		}
	}

	return retNodes
}
