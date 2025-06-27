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

package coreutil

import "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"

func TiProxyGroupClientPort(proxyg *v1alpha1.TiProxyGroup) int32 {
	if proxyg.Spec.Template.Spec.Server.Ports.Client != nil {
		return proxyg.Spec.Template.Spec.Server.Ports.Client.Port
	}
	return v1alpha1.DefaultTiProxyPortClient
}

func TiProxyGroupAPIPort(proxyg *v1alpha1.TiProxyGroup) int32 {
	if proxyg.Spec.Template.Spec.Server.Ports.API != nil {
		return proxyg.Spec.Template.Spec.Server.Ports.API.Port
	}
	return v1alpha1.DefaultTiProxyPortAPI
}

func TiProxyGroupPeerPort(proxyg *v1alpha1.TiProxyGroup) int32 {
	if proxyg.Spec.Template.Spec.Server.Ports.Peer != nil {
		return proxyg.Spec.Template.Spec.Server.Ports.Peer.Port
	}
	return v1alpha1.DefaultTiProxyPortPeer
}

func TiProxyClientPort(tiproxy *v1alpha1.TiProxy) int32 {
	if tiproxy.Spec.Server.Ports.Client != nil {
		return tiproxy.Spec.Server.Ports.Client.Port
	}
	return v1alpha1.DefaultTiProxyPortClient
}

func TiProxyAPIPort(tiproxy *v1alpha1.TiProxy) int32 {
	if tiproxy.Spec.Server.Ports.API != nil {
		return tiproxy.Spec.Server.Ports.API.Port
	}
	return v1alpha1.DefaultTiProxyPortAPI
}

func TiProxyPeerPort(tiproxy *v1alpha1.TiProxy) int32 {
	if tiproxy.Spec.Server.Ports.Peer != nil {
		return tiproxy.Spec.Server.Ports.Peer.Port
	}
	return v1alpha1.DefaultTiProxyPortPeer
}

// TiProxyMySQLTLSSecretName returns the secret name used in TiProxy server for the TLS between TiProxy server and MySQL client.
func TiProxyMySQLTLSSecretName(tiproxy *v1alpha1.TiProxy) string {
	prefix, _ := NamePrefixAndSuffix(tiproxy)
	return prefix + "-tiproxy-server-secret"
}

func TiProxyTiDBTLSSecretName(tiproxy *v1alpha1.TiProxy) string {
	prefix, _ := NamePrefixAndSuffix(tiproxy)
	return prefix + "-tiproxy-tidb-secret"
}

// IsTiProxyMySQLTLSEnabled returns whether the TLS between TiProxy server and MySQL client is enabled.
func IsTiProxyMySQLTLSEnabled(tiproxy *v1alpha1.TiProxy) bool {
	return tiproxy.Spec.Security != nil &&
		tiproxy.Spec.Security.TLS != nil &&
		tiproxy.Spec.Security.TLS.MySQL != nil &&
		tiproxy.Spec.Security.TLS.MySQL.Enabled
}

func IsTiProxyTiDBTLSEnabled(tiproxy *v1alpha1.TiProxy) bool {
	return tiproxy.Spec.Security != nil &&
		tiproxy.Spec.Security.TLS != nil &&
		tiproxy.Spec.Security.TLS.Backend != nil &&
		tiproxy.Spec.Security.TLS.Backend.Enabled
}
