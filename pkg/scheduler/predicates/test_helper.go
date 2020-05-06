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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func fakeThreeNodes() []apiv1.Node {
	return []apiv1.Node{
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-1",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-1",
					"zone":                   "zone1",
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-2",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-2",
					"zone":                   "zone2",
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-3",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-3",
					"zone":                   "zone3",
				},
			},
		},
	}
}

func fakeFourNodes() []apiv1.Node {
	return []apiv1.Node{
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-1",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-1",
					"zone":                   "zone1",
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-2",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-2",
					"zone":                   "zone2",
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-3",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-3",
					"zone":                   "zone3",
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-4",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-4",
					"zone":                   "zone4",
				},
			},
		},
	}
}

func fakeFourNodesWithThreeTopologies() []apiv1.Node {
	return []apiv1.Node{
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-1",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-1",
					"zone":                   "zone1",
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-2",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-2",
					"zone":                   "zone2",
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-3",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-3",
					"zone":                   "zone3",
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-4",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-4",
					"zone":                   "zone3",
				},
			},
		},
	}
}

func fakeTwoNodes() []apiv1.Node {
	return []apiv1.Node{
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-1",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-1",
					"zone":                   "zone1",
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-2",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-2",
					"zone":                   "zone2",
				},
			},
		},
	}
}

func fakeOneNode() []apiv1.Node {
	return []apiv1.Node{
		{
			TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-node-1",
				Labels: map[string]string{
					"kubernetes.io/hostname": "kube-node-1",
					"zone":                   "zone1",
				},
			},
		},
	}
}

func fakeZeroNode() []apiv1.Node {
	return []apiv1.Node{}
}

func CollectEvents(source <-chan string) []string {
	done := false
	events := make([]string, 0)
	for !done {
		select {
		case event := <-source:
			events = append(events, event)
		default:
			done = true
		}
	}
	return events
}
