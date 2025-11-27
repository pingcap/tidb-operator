// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package data

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func NewTiProxyGroup(ns string, patches ...GroupPatch[*v1alpha1.TiProxyGroup]) *v1alpha1.TiProxyGroup {
	proxyg := &v1alpha1.TiProxyGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      defaultTiProxyGroupName,
		},
		Spec: v1alpha1.TiProxyGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: defaultClusterName},
			Replicas: ptr.To[int32](1),
			Template: v1alpha1.TiProxyTemplate{
				Spec: v1alpha1.TiProxyTemplateSpec{
					Image:   ptr.To(defaultImageRegistry + "tiproxy"),
					Version: defaultTiProxyVersion,
					Probes: v1alpha1.TiProxyProbes{
						Readiness: &v1alpha1.TiProxyProb{
							// Currently the e2e env does not support command probe.
							// Remove this after the env is fixed.
							Type: ptr.To(v1alpha1.TCPProbeType),
						},
					},
				},
			},
		},
	}

	for _, p := range patches {
		p.Patch(proxyg)
	}

	return proxyg
}

// Deprecated: use WithTiProxyMySQLTLS
func WithTLSForTiProxy() GroupPatch[*v1alpha1.TiProxyGroup] {
	return GroupPatchFunc[*v1alpha1.TiProxyGroup](func(obj *v1alpha1.TiProxyGroup) {
		if obj.Spec.Template.Spec.Security == nil {
			obj.Spec.Template.Spec.Security = &v1alpha1.TiProxySecurity{}
		}

		obj.Spec.Template.Spec.Security.TLS = &v1alpha1.TiProxyTLSConfig{
			MySQL: &v1alpha1.TLS{
				Enabled: true,
			},
		}
	})
}

func WithTiProxyMySQLTLS(ca, certKeyPair string) GroupPatch[*v1alpha1.TiProxyGroup] {
	return GroupPatchFunc[*v1alpha1.TiProxyGroup](func(obj *v1alpha1.TiProxyGroup) {
		if obj.Spec.Template.Spec.Security == nil {
			obj.Spec.Template.Spec.Security = &v1alpha1.TiProxySecurity{}
		}
		if obj.Spec.Template.Spec.Security.TLS == nil {
			obj.Spec.Template.Spec.Security.TLS = &v1alpha1.TiProxyTLSConfig{}
		}

		obj.Spec.Template.Spec.Security.TLS.MySQL = &v1alpha1.TLS{
			Enabled: true,
		}
		if ca != "" {
			obj.Spec.Template.Spec.Security.TLS.MySQL.CA = &v1alpha1.CAReference{
				Name: ca,
			}
		}
		if certKeyPair != "" {
			obj.Spec.Template.Spec.Security.TLS.MySQL.CertKeyPair = &v1alpha1.CertKeyPairReference{
				Name: certKeyPair,
			}
		}
	})
}

func WithHotReloadPolicyForTiProxy() GroupPatch[*v1alpha1.TiProxyGroup] {
	return GroupPatchFunc[*v1alpha1.TiProxyGroup](func(obj *v1alpha1.TiProxyGroup) {
		obj.Spec.Template.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyHotReload
	})
}

func WithTiProxyNextGen() GroupPatch[*v1alpha1.TiProxyGroup] {
	return GroupPatchFunc[*v1alpha1.TiProxyGroup](func(obj *v1alpha1.TiProxyGroup) {
		obj.Spec.Template.Spec.Version = "v1.4.0"
		obj.Spec.Template.Spec.Image = ptr.To(defaultImageRegistry + "tiproxy:v1.4.0-beta.1-26-g02d15d8")
		obj.Spec.Template.Spec.Config = `[balance]
label-name = "keyspace"`
	})
}
