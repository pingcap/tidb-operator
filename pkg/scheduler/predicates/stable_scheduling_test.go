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

package predicates

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	pingcapfake "github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/label"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"
)

const (
	instanceName = "demo"
)

func makeTidbCluster(name, node string) *v1alpha1.TidbCluster {
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: instanceName,
		},
		Status: v1alpha1.TidbClusterStatus{
			TiDB: v1alpha1.TiDBStatus{},
		},
	}
	if name != "" && node != "" {
		tc.Status.TiDB.Members = make(map[string]v1alpha1.TiDBMember)
		tc.Status.TiDB.Members[name] = v1alpha1.TiDBMember{
			NodeName: node,
		}
	}
	return tc
}

func makePod(name string, component string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: v1.NamespaceDefault,
			Name:      name,
			Labels: map[string]string{
				label.ComponentLabelKey: component,
			},
			GenerateName: name[0 : len(name)-1],
		},
	}
}

func makeNode(name string) v1.Node {
	return v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func TestStableSchedulingFilter(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name           string
		instanceName   string
		pod            *v1.Pod
		candicateNodes []v1.Node
		tidbCluster    *v1alpha1.TidbCluster
		expectFn       func([]v1.Node, error, *record.FakeRecorder)
	}

	tests := []testcase{
		{
			name:         "cannot schedule to previous node because it's not a TiDB pod",
			instanceName: "demo",
			pod:          makePod("demo-tikv-0", label.TiKVLabelVal),
			tidbCluster:  nil,
			candicateNodes: []v1.Node{
				makeNode("node-1"),
				makeNode("node-2"),
				makeNode("node-3"),
			},
			expectFn: func(nodes []v1.Node, err error, recorder *record.FakeRecorder) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(3))
			},
		},
		{
			name:         "cannot schedule to previous node because tidb cluster does not exist",
			instanceName: "demo",
			pod:          makePod("demo-tidb-0", label.TiDBLabelVal),
			tidbCluster:  nil,
			candicateNodes: []v1.Node{
				makeNode("node-1"),
				makeNode("node-2"),
				makeNode("node-3"),
			},
			expectFn: func(nodes []v1.Node, err error, recorder *record.FakeRecorder) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(3))
			},
		},
		{
			name:         "cannot schedule to previous node because no previous node exist in tidb cluster",
			instanceName: "demo",
			pod:          makePod("demo-tidb-0", label.TiDBLabelVal),
			tidbCluster:  makeTidbCluster("", ""),
			candicateNodes: []v1.Node{
				makeNode("node-1"),
				makeNode("node-2"),
				makeNode("node-3"),
			},
			expectFn: func(nodes []v1.Node, err error, recorder *record.FakeRecorder) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring("no previous node exists for pod \"demo-tidb-0\" in TiDB cluster default/demo"))
				g.Expect(len(nodes)).To(Equal(3))
			},
		},
		{
			name:         "cannot schedule to previous node because previous node does not exist in candicates",
			instanceName: "demo",
			pod:          makePod("demo-tidb-0", label.TiDBLabelVal),
			tidbCluster:  makeTidbCluster("demo-tidb-0", "node-4"),
			candicateNodes: []v1.Node{
				makeNode("node-1"),
				makeNode("node-2"),
				makeNode("node-3"),
			},
			expectFn: func(nodes []v1.Node, err error, recorder *record.FakeRecorder) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring("cannot run default/demo-tidb-0 on its previous node \"node-4\""))
				g.Expect(len(nodes)).To(Equal(3))
			},
		},
		{
			name:         "schedule the pod to previous node",
			instanceName: "demo",
			pod:          makePod("demo-tidb-0", label.TiDBLabelVal),
			tidbCluster:  makeTidbCluster("demo-tidb-0", "node-2"),
			candicateNodes: []v1.Node{
				makeNode("node-1"),
				makeNode("node-2"),
				makeNode("node-3"),
			},
			expectFn: func(nodes []v1.Node, err error, recorder *record.FakeRecorder) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"node-2"}))
			},
		},
	}

	for _, tc := range tests {
		t.Log(tc.name)
		recorder := record.NewFakeRecorder(10)
		kubeCli := fake.NewSimpleClientset()
		if tc.pod != nil {
			_, err := kubeCli.CoreV1().Pods(v1.NamespaceDefault).Create(tc.pod)
			g.Expect(err).NotTo(HaveOccurred())
		}
		cli := pingcapfake.NewSimpleClientset()
		if tc.tidbCluster != nil {
			_, err := cli.PingcapV1alpha1().TidbClusters(v1.NamespaceDefault).Create(tc.tidbCluster)
			g.Expect(err).NotTo(HaveOccurred())
		}
		p := stableScheduling{
			kubeCli: kubeCli,
			cli:     cli,
		}
		nodes, err := p.Filter(tc.instanceName, tc.pod, tc.candicateNodes)
		tc.expectFn(nodes, err, recorder)
	}
}
