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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPumpAdvertiseAddr(t *testing.T) {
	g := NewGomegaWithT(t)

	type Case struct {
		clusterName string
		domain      string
		result      string
	}

	tests := []Case{
		{
			clusterName: "cname",
			result:      "pod.cname-pump:8250",
		},
		{
			clusterName: "cname",
			domain:      "cluster.local",
			result:      "pod.cname-pump.ns.svc.cluster.local:8250",
		},
	}

	for _, test := range tests {
		model := &PumpStartScriptModel{
			ClusterName:   test.clusterName,
			ClusterDomain: test.domain,
			Namespace:     "ns",
		}

		data, err := RenderPumpStartScript(model)
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
