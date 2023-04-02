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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// TcTlsManager manages the tls secrets, which are borrowed from TiDBCluster, for TiDBDashboard.
type TcTlsManager struct {
	deps *controller.Dependencies
}

func NewTcTlsManager(deps *controller.Dependencies) *TcTlsManager {
	return &TcTlsManager{
		deps: deps,
	}
}

func (m *TcTlsManager) Sync(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) error {
	err := m.synTLSSecret(td, tc)
	if err != nil {
		klog.Errorf("sync client tls of tc %s/%s for tidb dashboard %s/%s failed: %s",
			tc.Namespace, tc.Name, td.Namespace, td.Name, err)
		return err
	}

	return nil
}

func (m *TcTlsManager) synTLSSecret(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) error {
	tcName := tc.GetName()
	tcNS := tc.GetNamespace()

	// Sync cluster client tls secret.
	if tc.IsTLSClusterEnabled() {
		// Get the secret from tc.
		tcClientSecretName := util.ClusterClientTLSSecretName(tcName)
		tcSecret, err := m.deps.SecretLister.Secrets(tcNS).Get(tcClientSecretName)
		if err != nil {
			return err
		}

		// Build the secret for tidb-dashboard.
		clusterClientTLSMeta, _ := generateTiDBDashboardMeta(td, TCClusterClientTLSSecretName(td.Name))
		clusterClientTLSSecret := &corev1.Secret{
			ObjectMeta: clusterClientTLSMeta,
			Data:       tcSecret.DeepCopy().Data,
		}

		// Create or update clusterClientTLSSecret.
		_, err = m.deps.TypedControl.CreateOrUpdateSecret(td, clusterClientTLSSecret)
		if err != nil {
			return err
		}
	}

	// Sync mysql client tls secret.
	if tc.Spec.TiDB != nil && tc.Spec.TiDB.IsTLSClientEnabled() && !tc.SkipTLSWhenConnectTiDB() {
		tcSecret, err := m.deps.SecretLister.Secrets(tcNS).Get(util.TiDBClientTLSSecretName(tcName, nil))
		if err != nil {
			return err
		}

		mysqlClientTLSMeta, _ := generateTiDBDashboardMeta(td, TCMySQLClientTLSSecretName(td.Name))
		mysqlClientTLSSecret := &corev1.Secret{
			ObjectMeta: mysqlClientTLSMeta,
			Data:       tcSecret.DeepCopy().Data,
		}

		_, err = m.deps.TypedControl.CreateOrUpdateSecret(td, mysqlClientTLSSecret)
		if err != nil {
			return err
		}
	}

	return nil
}
