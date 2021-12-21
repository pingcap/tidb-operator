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

package tidbngmonitoring

import (
	"fmt"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/controller"

	. "github.com/onsi/gomega"
)

func TestNGMonitoringStartScriptModel(t *testing.T) {

	t.Run("FormatClusterDomain", func(t *testing.T) {
		g := NewGomegaWithT(t)

		model := &NGMonitoringStartScriptModel{}
		g.Expect(model.FormatClusterDomain()).Should(Equal(""))

		model.TCClusterDomain = "cluster.local"
		g.Expect(model.FormatClusterDomain()).Should(Equal(".cluster.local"))
	})

	t.Run("PDAddress", func(t *testing.T) {
		g := NewGomegaWithT(t)

		type testcase struct {
			model *NGMonitoringStartScriptModel

			expectAddr string
		}

		cases := []testcase{
			{
				model: &NGMonitoringStartScriptModel{
					TCName:          "tc",
					TCNamespace:     "default",
					TCClusterDomain: "",
				},
				expectAddr: fmt.Sprintf("%s.default:2379", controller.PDMemberName("tc")),
			},
		}

		for _, testcase := range cases {
			addr := testcase.model.PDAddress()
			g.Expect(addr).Should(Equal(testcase.expectAddr))
		}
	})

	t.Run("NGMPeerAddress", func(t *testing.T) {
		g := NewGomegaWithT(t)

		type testcase struct {
			model *NGMonitoringStartScriptModel

			expectAddr string
		}

		cases := []testcase{
			{
				model: &NGMonitoringStartScriptModel{
					TNGMName:          "ng-monitoring",
					TNGMNamespace:     "default",
					TNGMClusterDomain: "",
				},
				expectAddr: fmt.Sprintf("${POD_NAME}.%s.default:12020", NGMonitoringHeadlessServiceName("ng-monitoring")),
			},
			{
				model: &NGMonitoringStartScriptModel{
					TNGMName:          "ng-monitoring",
					TNGMNamespace:     "default",
					TNGMClusterDomain: "cluster.local",
				},
				expectAddr: fmt.Sprintf("${POD_NAME}.%s.default.svc.cluster.local:12020", NGMonitoringHeadlessServiceName("ng-monitoring")),
			},
		}

		for _, testcase := range cases {
			addr := testcase.model.NGMPeerAddress()
			g.Expect(addr).Should(Equal(testcase.expectAddr))
		}
	})
}
