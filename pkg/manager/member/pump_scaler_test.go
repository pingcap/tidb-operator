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

package member

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPumpAdvertiseAddr(t *testing.T) {
	g := NewGomegaWithT(t)

	type Case struct {
		name string

		model  *PumpStartScriptModel
		result string
	}

	tests := []Case{
		{
			name: "parse addr from basic startup script",
			model: &PumpStartScriptModel{
				ClusterName: "cname",
				Namespace:   "ns",
			},
			result: "pod.cname-pump:8250",
		},
		{
			name: "parse addr from remote heterogeneous startup script",
			model: &PumpStartScriptModel{
				CommonModel: CommonModel{
					RefCluster: &v1alpha1.TidbClusterRef{
						Namespace:     "default",
						Name:          "cluster-2",
						ClusterDomain: "cluster.local",
					},
					ClusterDomain: "cluster.local",
				},
				ClusterName: "cname",
				Namespace:   "ns",
			},
			result: "pod.cname-pump.ns.svc.cluster.local:8250",
		},
	}

	for _, test := range tests {
		data, err := RenderPumpStartScript(test.model)
		g.Expect(err).Should(BeNil())

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod",
				Namespace: "ns",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:    "pump",
						Command: []string{data},
					},
				},
			},
		}

		addr := pumpAdvertiseAddr(pod)
		g.Expect(addr).Should(Equal(test.result))
	}
}
