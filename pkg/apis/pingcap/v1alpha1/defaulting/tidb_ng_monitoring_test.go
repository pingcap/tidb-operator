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

package defaulting

import (
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

	. "github.com/onsi/gomega"
)

func TestSetTidbNGMonitoringDefault(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		setTNGM  func(*v1alpha1.TidbNGMonitoring)
		expectFn func(*v1alpha1.TidbNGMonitoring)
	}

	cases := []testcase{
		{
			name: "should set ns if ns of tc isn't setted",
			setTNGM: func(tngm *v1alpha1.TidbNGMonitoring) {
				tngm.Spec.Clusters = []v1alpha1.TidbClusterRef{
					{
						Name: "tc",
					},
					{
						Name:      "tc-2",
						Namespace: "tc-2-ns",
					},
				}
			},
			expectFn: func(tngm *v1alpha1.TidbNGMonitoring) {
				g.Expect(tngm.Spec.Clusters[0].Namespace).Should(Equal(tngm.Namespace))
				g.Expect(tngm.Spec.Clusters[1].Namespace).Should(Equal("tc-2-ns"))
			},
		},
	}

	for _, testcase := range cases {
		t.Logf("testcase: %s", testcase.name)

		tngm := &v1alpha1.TidbNGMonitoring{}
		tngm.Name = "tngm"
		tngm.Namespace = "tngm-ns"
		if testcase.setTNGM != nil {
			testcase.setTNGM(tngm)
		}

		SetTidbNGMonitoringDefault(tngm)

		testcase.expectFn(tngm)
	}
}
