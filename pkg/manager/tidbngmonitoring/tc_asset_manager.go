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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type tcAssetManager struct {
	deps *controller.Dependencies
}

func NewTCAssetManager(deps *controller.Dependencies) *tcAssetManager {
	return &tcAssetManager{
		deps: deps,
	}
}

func (m *tcAssetManager) Sync(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) error {
	err := m.syncClientTLSSecret(tngm, tc)
	if err != nil {
		klog.Errorf("sync client tls of tc %s/%s for tidb ng monitoring %s/%s failed: %s",
			tc.Namespace, tc.Name, tngm.Namespace, tngm.Name, err)
		return err
	}

	return nil
}

func (m *tcAssetManager) syncClientTLSSecret(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) error {
	name := tngm.Name
	tcName := tc.GetName()
	tcNS := tc.GetNamespace()

	// collect secret of tc
	tcClientSecretName := util.ClusterClientTLSSecretName(tcName)
	tcSecret, err := m.deps.SecretLister.Secrets(tcNS).Get(tcClientSecretName)
	if err != nil {
		return err
	}

	// generate data of secret
	data := map[string][]byte{}
	for k, v := range tcSecret.Data {
		data[assetKey(tcName, tcNS, k)] = v
	}

	// create/update secret
	meta, _ := GenerateNGMonitoringMeta(tngm, TCClientTLSSecretName(name))
	secret := &corev1.Secret{
		ObjectMeta: meta,
		Data:       data,
	}
	_, err = m.deps.TypedControl.CreateOrUpdateSecret(tngm, secret)
	if err != nil {
		return err
	}

	return nil
}
