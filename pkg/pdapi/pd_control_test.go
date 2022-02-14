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

package pdapi

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestPDControl(t *testing.T) {
	t.Run("clientConfig", func(t *testing.T) {

		g := NewGomegaWithT(t)

		type testcase struct {
			name         string
			options      []Option
			tcName       string
			tcNS         Namespace
			tlsEnable    bool
			expectConfig func(pdClient *clientConfig, etcdClient *clientConfig)
		}

		cases := []testcase{
			{
				name:    "http default",
				options: []Option{},
				tcName:  "target-cluster",
				tcNS:    "target-namespace",
				expectConfig: func(pdClient *clientConfig, etcdClient *clientConfig) {
					g.Expect(pdClient.clientURL).To(Equal("http://target-cluster-pd.target-namespace:2379"))
					g.Expect(pdClient.clientKey).To(Equal("http.target-cluster.target-namespace"))
					g.Expect(pdClient.tlsEnable).To(BeFalse())

					g.Expect(etcdClient.clientURL).To(Equal("target-cluster-pd.target-namespace:2379"))
					g.Expect(etcdClient.clientKey).To(Equal("target-cluster.target-namespace.false"))
					g.Expect(etcdClient.tlsEnable).To(BeFalse())
				},
			},
			{
				name:      "https default",
				options:   []Option{},
				tcName:    "target-cluster",
				tcNS:      "target-namespace",
				tlsEnable: true,
				expectConfig: func(pdClient *clientConfig, etcdClient *clientConfig) {
					g.Expect(pdClient.clientURL).To(Equal("https://target-cluster-pd.target-namespace:2379"))
					g.Expect(pdClient.clientKey).To(Equal("https.target-cluster.target-namespace"))
					g.Expect(pdClient.tlsEnable).To(BeTrue())

					g.Expect(etcdClient.clientURL).To(Equal("target-cluster-pd.target-namespace:2379"))
					g.Expect(etcdClient.clientKey).To(Equal("target-cluster.target-namespace.true"))
					g.Expect(etcdClient.tlsEnable).To(BeTrue())
				},
			},
			{
				name: "set cluster domain",
				options: []Option{
					ClusterRef("cluster.local"),
				},
				tcName: "target-cluster",
				tcNS:   "target-namespace",
				expectConfig: func(pdClient *clientConfig, etcdClient *clientConfig) {
					g.Expect(pdClient.clientURL).To(Equal("http://target-cluster-pd.target-namespace.svc.cluster.local:2379"))
					g.Expect(pdClient.clientKey).To(Equal("http.target-cluster.target-namespace.cluster.local"))

					g.Expect(etcdClient.clientURL).To(Equal("target-cluster-pd.target-namespace.svc.cluster.local:2379"))
					g.Expect(etcdClient.clientKey).To(Equal("target-cluster.target-namespace.cluster.local.false"))

					g.Expect(pdClient.clusterDomain).To(Equal("cluster.local"))
					g.Expect(etcdClient.clusterDomain).To(Equal("cluster.local"))
				},
			},
			{
				name: "use headless service",
				options: []Option{
					UseHeadlessService(true),
				},
				tcName: "target-cluster",
				tcNS:   "target-namespace",
				expectConfig: func(pdClient *clientConfig, etcdClient *clientConfig) {
					g.Expect(pdClient.clientURL).To(Equal("http://target-cluster-pd-peer.target-namespace:2379"))
					g.Expect(pdClient.clientKey).To(Equal("http.target-cluster.target-namespace"))

					g.Expect(etcdClient.clientURL).To(Equal("target-cluster-pd-peer.target-namespace:2379"))
					g.Expect(etcdClient.clientKey).To(Equal("target-cluster.target-namespace.false"))

					g.Expect(pdClient.headlessSvc).To(BeTrue())
					g.Expect(etcdClient.headlessSvc).To(BeTrue())
				},
			},
			{
				name: "use headless service and cluster domain",
				options: []Option{
					UseHeadlessService(true),
					ClusterRef("cluster.local"),
				},
				tcName: "target-cluster",
				tcNS:   "target-namespace",
				expectConfig: func(pdClient *clientConfig, etcdClient *clientConfig) {
					g.Expect(pdClient.clientURL).To(Equal("http://target-cluster-pd-peer.target-namespace.svc.cluster.local:2379"))
					g.Expect(pdClient.clientKey).To(Equal("http.target-cluster.target-namespace.cluster.local"))

					g.Expect(etcdClient.clientURL).To(Equal("target-cluster-pd-peer.target-namespace.svc.cluster.local:2379"))
					g.Expect(etcdClient.clientKey).To(Equal("target-cluster.target-namespace.cluster.local.false"))

					g.Expect(pdClient.clusterDomain).To(Equal("cluster.local"))
					g.Expect(etcdClient.clusterDomain).To(Equal("cluster.local"))

					g.Expect(pdClient.headlessSvc).To(BeTrue())
					g.Expect(etcdClient.headlessSvc).To(BeTrue())
				},
			},
			{
				name: "specify client",
				options: []Option{
					SpecifyClient("http://test.cluster", "test.cluster"),
					UseHeadlessService(true),    // should be useless
					ClusterRef("cluster.local"), // should be useless
				},
				tcName:    "target-cluster",
				tcNS:      "target-namespace",
				tlsEnable: true,
				expectConfig: func(pdClient *clientConfig, etcdClient *clientConfig) {
					g.Expect(pdClient.clientURL).To(Equal("http://test.cluster"))
					g.Expect(pdClient.clientKey).To(Equal("test.cluster"))

					g.Expect(etcdClient.clientURL).To(Equal("http://test.cluster"))
					g.Expect(etcdClient.clientKey).To(Equal("test.cluster"))
				},
			},
			{
				name: "cert from tc",
				options: []Option{
					TLSCertFromTC("test-namespace", "test-cluster"),
				},
				tcName:    "target-cluster",
				tcNS:      "target-namespace",
				tlsEnable: true,
				expectConfig: func(pdClient *clientConfig, etcdClient *clientConfig) {
					g.Expect(pdClient.clientURL).To(Equal("https://target-cluster-pd.target-namespace:2379"))
					g.Expect(pdClient.clientKey).To(Equal("https.target-cluster.target-namespace"))
					g.Expect(pdClient.tlsEnable).To(BeTrue())
					g.Expect(pdClient.tlsSecretName).To(Equal("test-cluster-cluster-client-secret"))
					g.Expect(string(pdClient.tlsSecretNamespace)).To(Equal("test-namespace"))

					g.Expect(etcdClient.clientURL).To(Equal("target-cluster-pd.target-namespace:2379"))
					g.Expect(etcdClient.clientKey).To(Equal("target-cluster.target-namespace.true"))
					g.Expect(etcdClient.tlsEnable).To(BeTrue())
					g.Expect(etcdClient.tlsSecretName).To(Equal("test-cluster-cluster-client-secret"))
					g.Expect(string(etcdClient.tlsSecretNamespace)).To(Equal("test-namespace"))
				},
			},
			{
				name: "cert from secret",
				options: []Option{
					TLSCertFromSecret("test-namespace", "test-secret"),
				},
				tcName:    "target-cluster",
				tcNS:      "target-namespace",
				tlsEnable: true,
				expectConfig: func(pdClient *clientConfig, etcdClient *clientConfig) {
					g.Expect(pdClient.clientURL).To(Equal("https://target-cluster-pd.target-namespace:2379"))
					g.Expect(pdClient.clientKey).To(Equal("https.target-cluster.target-namespace"))
					g.Expect(pdClient.tlsEnable).To(BeTrue())
					g.Expect(pdClient.tlsSecretName).To(Equal("test-secret"))
					g.Expect(string(pdClient.tlsSecretNamespace)).To(Equal("test-namespace"))

					g.Expect(etcdClient.clientURL).To(Equal("target-cluster-pd.target-namespace:2379"))
					g.Expect(etcdClient.clientKey).To(Equal("target-cluster.target-namespace.true"))
					g.Expect(etcdClient.tlsEnable).To(BeTrue())
					g.Expect(etcdClient.tlsSecretName).To(Equal("test-secret"))
					g.Expect(string(etcdClient.tlsSecretNamespace)).To(Equal("test-namespace"))
				},
			},
		}

		for _, c := range cases {
			t.Logf("test case: %s", c.name)

			pdClient := &clientConfig{}
			etcdClient := &clientConfig{}
			pdClient.tlsEnable = c.tlsEnable
			etcdClient.tlsEnable = c.tlsEnable

			pdClient.applyOptions(c.options...)
			etcdClient.applyOptions(c.options...)

			pdClient.completeForPDClient(c.tcNS, c.tcName)
			etcdClient.completeForEtcdClient(c.tcNS, c.tcName)

			c.expectConfig(pdClient, etcdClient)
		}
	})
}
