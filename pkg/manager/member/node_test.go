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

package member

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetNodeLabels(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		nodeLabels  map[string]string
		labels      []string
		result      map[string]string
		errExpectFn func(*GomegaWithT, error)
	}

	testNodeName := "test-node"

	testFn := func(c *testcase, t *testing.T) {
		fakeDeps := controller.NewFakeDependencies()
		nodeIndexer := fakeDeps.KubeInformerFactory.Core().V1().Nodes().Informer().GetIndexer()

		nodeIndexer.Add(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testNodeName,
				Labels: c.nodeLabels,
			},
		})

		res, err := getNodeLabels(fakeDeps.NodeLister, testNodeName, c.labels)
		if c.errExpectFn != nil {
			c.errExpectFn(g, err)
		} else {
			g.Expect(err).To(BeNil())
			g.Expect(res).To(Equal(c.result))
		}
	}

	tests := []*testcase{
		{
			nodeLabels: map[string]string{
				"region": "us-west-1",
				"zone":   "us-west-1a",
				"host":   "172.16.0.1",
			},
			labels: []string{"region", "zone", "host"},
			result: map[string]string{
				"region": "us-west-1",
				"zone":   "us-west-1a",
				"host":   "172.16.0.1",
			},
		},
		{
			nodeLabels: map[string]string{
				"kubernetes.io/os": "Linux",
				"region":           "us-west-1",
				"zone":             "us-west-1a",
				"host":             "172.16.0.1",
			},
			labels: []string{"zone", "host"},
			result: map[string]string{
				"zone": "us-west-1a",
				"host": "172.16.0.1",
			},
		},
		{
			nodeLabels: map[string]string{
				"kubernetes.io/os": "Linux",
				"region":           "us-west-1",
				"zone":             "us-west-1a",
			},
			labels: []string{"region", "zone", "host"},
			result: map[string]string{
				"region": "us-west-1",
				"zone":   "us-west-1a",
			},
		},
		{
			nodeLabels: map[string]string{
				"kubernetes.io/os":              "Linux",
				"topology.kubernetes.io/region": "us-west-1",
				"topology.kubernetes.io/zone":   "us-west-1a",
				"kubernetes.io/hostname":        "172.16.0.1",
			},
			labels: []string{"topology.kubernetes.io/region", "topology.kubernetes.io/zone", "kubernetes.io/hostname"},
			result: map[string]string{
				"topology.kubernetes.io/region": "us-west-1",
				"topology.kubernetes.io/zone":   "us-west-1a",
				"kubernetes.io/hostname":        "172.16.0.1",
			},
		},
		{
			nodeLabels: map[string]string{
				"kubernetes.io/os":              "Linux",
				"topology.kubernetes.io/region": "us-west-1",
				"topology.kubernetes.io/zone":   "us-west-1a",
				"kubernetes.io/hostname":        "172.16.0.1",
			},
			labels: []string{"region", "zone", "host"},
			result: map[string]string{
				"region": "us-west-1",
				"zone":   "us-west-1a",
				"host":   "172.16.0.1",
			},
		},
		{
			nodeLabels: map[string]string{
				"kubernetes.io/os":                         "Linux",
				"failure-domain.beta.kubernetes.io/region": "us-west-1",
				"failure-domain.beta.kubernetes.io/zone":   "us-west-1a",
				"kubernetes.io/hostname":                   "172.16.0.1",
			},
			labels: []string{"region", "zone", "host"},
			result: map[string]string{
				"region": "us-west-1",
				"zone":   "us-west-1a",
				"host":   "172.16.0.1",
			},
		},
		{
			nodeLabels: map[string]string{
				"kubernetes.io/os":                         "Linux",
				"failure-domain.beta.kubernetes.io/region": "us-west-1",
				"failure-domain.beta.kubernetes.io/zone":   "us-west-1a",
				"kubernetes.io/hostname":                   "172.16.0.1",
				"region":                                   "test-region",
				"zone":                                     "test-zone",
				"host":                                     "test-host",
			},
			labels: []string{"region", "zone", "host"},
			result: map[string]string{
				"region": "test-region",
				"zone":   "test-zone",
				"host":   "test-host",
			},
		},
	}

	for _, test := range tests {
		testFn(test, t)
	}
}
