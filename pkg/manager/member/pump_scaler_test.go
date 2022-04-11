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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	startscriptv1 "github.com/pingcap/tidb-operator/pkg/manager/member/startscript/v1"
	startscriptv2 "github.com/pingcap/tidb-operator/pkg/manager/member/startscript/v2"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPumpAdvertiseAddr(t *testing.T) {
	g := NewGomegaWithT(t)

	type Case struct {
		name string

		tc     *v1alpha1.TidbCluster
		result string
	}

	tests := []Case{
		{
			name: "parse addr from basic startup script",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cname",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Pump: &v1alpha1.PumpSpec{},
				},
			},
			result: "pod.cname-pump:8250",
		},
		{
			name: "parse addr from heterogeneous startup script",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cname",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					AcrossK8s:     true,
					ClusterDomain: "cluster.local",
					Pump:          &v1alpha1.PumpSpec{},
				},
			},
			result: "pod.cname-pump.ns.svc.cluster.local:8250",
		},
	}

	for _, test := range tests {
		t.Logf("test case: %s", test.name)

		startScriptFns := []func(*v1alpha1.TidbCluster) (string, error){
			startscriptv1.RenderPumpStartScript,
			startscriptv2.RenderPumpStartScript,
		}

		for _, renderFn := range startScriptFns {
			data, err := renderFn(test.tc)
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
}
