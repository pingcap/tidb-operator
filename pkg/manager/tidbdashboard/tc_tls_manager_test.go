// Copyright 2022 PingCAP, Inc.
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

package tidbdashboard

import (
	"fmt"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func TestTLSManager(t *testing.T) {
	t.Run("synTLSSecret", func(t *testing.T) {
		g := NewGomegaWithT(t)

		type testcase struct {
			name string

			setInputs               func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster)
			getTCSecret             func() *corev1.Secret
			createOrUpdateSecretErr error
			expectFn                func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster, err error)
		}

		cases := []testcase{
			{
				name: "should create or update secret",
				getTCSecret: func() *corev1.Secret {
					secret := &corev1.Secret{}
					secret.Name = util.ClusterClientTLSSecretName("tc")
					secret.Namespace = "default"
					return secret
				},
				createOrUpdateSecretErr: nil,
				expectFn: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster, err error) {
					g.Expect(err).Should(Succeed())
				},
			},
			{
				name: "should return err if create or update failed",
				getTCSecret: func() *corev1.Secret {
					secret := &corev1.Secret{}
					secret.Name = util.ClusterClientTLSSecretName("tc")
					secret.Namespace = "default"
					return secret
				},
				createOrUpdateSecretErr: fmt.Errorf("test failed"),
				expectFn: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster, err error) {
					g.Expect(err).Should(HaveOccurred())
					g.Expect(err.Error()).Should(ContainSubstring("test failed"))
				},
			},
			{
				name:                    "should return err if tc secret is not found",
				getTCSecret:             nil,
				createOrUpdateSecretErr: nil,
				expectFn: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster, err error) {
					g.Expect(err).Should(HaveOccurred())
					g.Expect(err.Error()).Should(ContainSubstring("not found"))
				},
			},
			{
				name: "should create a empty secret if tls is disable",
				setInputs: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) {
					tc.Spec.TLSCluster = nil
				},
				createOrUpdateSecretErr: nil,
				expectFn: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster, err error) {
					g.Expect(err).Should(Succeed())
				},
			},
		}

		for _, testcase := range cases {
			t.Logf("testcase: %s", testcase.name)

			deps := controller.NewFakeDependencies()
			indexer := deps.KubeInformerFactory.Core().V1().Secrets().Informer().GetIndexer()

			manager := NewTcTlsManager(deps)

			td := &v1alpha1.TidbDashboard{}
			td.Name = "td"
			td.Namespace = "default"
			tc := &v1alpha1.TidbCluster{}
			tc.Name = "tc"
			tc.Namespace = "default"
			tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			if testcase.setInputs != nil {
				testcase.setInputs(td, tc)
			}

			if testcase.getTCSecret != nil {
				secret := testcase.getTCSecret()
				indexer.Add(secret)
			}

			manager.deps.GenericControl.(*controller.FakeGenericControl).SetCreateOrUpdateError(testcase.createOrUpdateSecretErr, 0)

			err := manager.synTLSSecret(td, tc)
			testcase.expectFn(td, tc, err)
		}
	})
}
