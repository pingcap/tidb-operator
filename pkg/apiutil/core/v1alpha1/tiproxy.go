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

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

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

func TiProxyMySQLTLS(db *v1alpha1.TiProxy) *v1alpha1.TLS {
	sec := db.Spec.Security
	if sec != nil && sec.TLS != nil && sec.TLS.MySQL != nil {
		return sec.TLS.MySQL
	}

	return nil
}

// TiProxyMySQLCertKeyPairSecretName returns the secret name used in TiProxy server for the TLS between TiProxy server and MySQL client.
func TiProxyMySQLCertKeyPairSecretName(tiproxy *v1alpha1.TiProxy) string {
	tls := TiProxyMySQLTLS(tiproxy)
	if tls != nil && tls.CertKeyPair != nil {
		return tls.CertKeyPair.Name
	}
	prefix, _ := NamePrefixAndSuffix(tiproxy)
	return prefix + "-tiproxy-server-secret"
}

// TiProxyMySQLCASecretName returns the secret name for TiProxy server to authenticate MySQL client.
func TiProxyMySQLCASecretName(tiproxy *v1alpha1.TiProxy) string {
	tls := TiProxyMySQLTLS(tiproxy)
	if tls != nil && tls.CA != nil {
		return tls.CA.Name
	}
	prefix, _ := NamePrefixAndSuffix(tiproxy)
	return prefix + "-tiproxy-server-secret"
}

func TiProxyMySQLTLSVolume(tiproxy *v1alpha1.TiProxy) *corev1.Volume {
	tls := TiProxyMySQLTLS(tiproxy)
	certKeyPair := TiProxyMySQLCertKeyPairSecretName(tiproxy)

	if tls != nil && tls.ClientAuth == v1alpha1.ClientAuthTypeNoClientCert {
		return &corev1.Volume{
			Name: v1alpha1.VolumeNameTiProxyMySQLTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: certKeyPair,
				},
			},
		}
	}

	ca := TiProxyMySQLCASecretName(tiproxy)

	if ca == certKeyPair {
		return &corev1.Volume{
			Name: v1alpha1.VolumeNameTiProxyMySQLTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: ca,
				},
			},
		}
	}

	return &corev1.Volume{
		Name: v1alpha1.VolumeNameTiProxyMySQLTLS,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: ca,
							},
						},
					},
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: certKeyPair,
							},
							// avoid mounting injected ca.crt
							Items: []corev1.KeyToPath{
								{
									Key:  corev1.TLSCertKey,
									Path: corev1.TLSCertKey,
								},
								{
									Key:  corev1.TLSPrivateKeyKey,
									Path: corev1.TLSPrivateKeyKey,
								},
							},
						},
					},
				},
			},
		},
	}
}

// IsTiProxyMySQLTLSEnabled returns whether the TLS between TiProxy server and MySQL client is enabled.
func IsTiProxyMySQLTLSEnabled(tiproxy *v1alpha1.TiProxy) bool {
	tls := TiProxyMySQLTLS(tiproxy)
	return tls != nil && tls.Enabled
}

func IsTiProxyMySQLNoClientCert(tiproxy *v1alpha1.TiProxy) bool {
	tls := TiProxyMySQLTLS(tiproxy)
	return tls != nil && tls.ClientAuth == v1alpha1.ClientAuthTypeNoClientCert
}

func TiProxyHTTPServerTLS(db *v1alpha1.TiProxy) *v1alpha1.TLS {
	sec := db.Spec.Security
	if sec != nil && sec.TLS != nil && sec.TLS.Server != nil {
		return sec.TLS.Server
	}

	return nil
}

// TiProxyHTTPServerCertKeyPairSecretName returns the secret name used for tiproxy http server.
func TiProxyHTTPServerCertKeyPairSecretName(tiproxy *v1alpha1.TiProxy) string {
	tls := TiProxyHTTPServerTLS(tiproxy)
	if tls != nil && tls.CertKeyPair != nil {
		return tls.CertKeyPair.Name
	}
	return ClusterCertKeyPairSecretName[scope.TiProxy](tiproxy)
}

// TiProxyHTTPServerCASecretName returns the secret name for TiProxy http server to authenticate clients.
func TiProxyHTTPServerCASecretName(tiproxy *v1alpha1.TiProxy) string {
	tls := TiProxyHTTPServerTLS(tiproxy)
	if tls != nil && tls.CA != nil {
		return tls.CA.Name
	}
	return ClusterCASecretName[scope.TiProxy](tiproxy)
}

func TiProxyHTTPServerTLSVolume(tiproxy *v1alpha1.TiProxy) *corev1.Volume {
	tls := TiProxyHTTPServerTLS(tiproxy)
	certKeyPair := TiProxyHTTPServerCertKeyPairSecretName(tiproxy)

	if tls != nil && tls.ClientAuth == v1alpha1.ClientAuthTypeNoClientCert {
		return &corev1.Volume{
			Name: v1alpha1.VolumeNameTiProxyHTTPTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: certKeyPair,
				},
			},
		}
	}

	ca := TiProxyHTTPServerCASecretName(tiproxy)

	if ca == certKeyPair {
		return &corev1.Volume{
			Name: v1alpha1.VolumeNameTiProxyHTTPTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: ca,
				},
			},
		}
	}

	return &corev1.Volume{
		Name: v1alpha1.VolumeNameTiProxyHTTPTLS,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: ca,
							},
						},
					},
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: certKeyPair,
							},
							// avoid mounting injected ca.crt
							Items: []corev1.KeyToPath{
								{
									Key:  corev1.TLSCertKey,
									Path: corev1.TLSCertKey,
								},
								{
									Key:  corev1.TLSPrivateKeyKey,
									Path: corev1.TLSPrivateKeyKey,
								},
							},
						},
					},
				},
			},
		},
	}
}

func IsTiProxyHTTPServerTLSEnabled(c *v1alpha1.Cluster, tiproxy *v1alpha1.TiProxy) bool {
	tls := TiProxyHTTPServerTLS(tiproxy)
	if tls == nil {
		return IsTLSClusterEnabled(c)
	}
	return tls.Enabled
}

func IsTiProxyHTTPServerNoClientCert(tiproxy *v1alpha1.TiProxy) bool {
	tls := TiProxyHTTPServerTLS(tiproxy)
	return tls != nil && tls.ClientAuth == v1alpha1.ClientAuthTypeNoClientCert
}

func TiProxyBackendTLS(db *v1alpha1.TiProxy) *v1alpha1.ClientTLS {
	sec := db.Spec.Security
	if sec != nil && sec.TLS != nil && sec.TLS.Backend != nil {
		return sec.TLS.Backend
	}

	return nil
}

func TiProxyBackendCertKeyPairSecretName(tiproxy *v1alpha1.TiProxy) string {
	tls := TiProxyBackendTLS(tiproxy)
	if tls != nil && tls.CertKeyPair != nil {
		return tls.CertKeyPair.Name
	}
	prefix, _ := NamePrefixAndSuffix(tiproxy)
	return prefix + "-tiproxy-tidb-secret"
}

func TiProxyBackendCASecretName(tiproxy *v1alpha1.TiProxy) string {
	tls := TiProxyBackendTLS(tiproxy)
	if tls != nil && tls.CA != nil {
		return tls.CA.Name
	}
	prefix, _ := NamePrefixAndSuffix(tiproxy)
	return prefix + "-tiproxy-tidb-secret"
}

func TiProxyBackendTLSVolume(tiproxy *v1alpha1.TiProxy) *corev1.Volume {
	tls := TiProxyBackendTLS(tiproxy)
	certKeyPair := TiProxyBackendCertKeyPairSecretName(tiproxy)
	ca := TiProxyBackendCASecretName(tiproxy)

	// not mutual and skip tls verification
	if !tls.Mutual && tls.InsecureSkipTLSVerify {
		return nil
	}

	// only mount CA
	if !tls.Mutual && !tls.InsecureSkipTLSVerify {
		return &corev1.Volume{
			Name: v1alpha1.VolumeNameTiProxyTiDBTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: ca,
				},
			},
		}
	}

	// only mount cert key pair
	if tls.InsecureSkipTLSVerify && tls.Mutual {
		return &corev1.Volume{
			Name: v1alpha1.VolumeNameTiProxyTiDBTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: certKeyPair,
				},
			},
		}
	}

	// if ca and cert key pair use same secret
	if ca == certKeyPair {
		return &corev1.Volume{
			Name: v1alpha1.VolumeNameTiProxyTiDBTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: ca,
				},
			},
		}
	}

	return &corev1.Volume{
		Name: v1alpha1.VolumeNameTiProxyTiDBTLS,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: ca,
							},
						},
					},
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: certKeyPair,
							},
							// avoid mounting injected ca.crt
							Items: []corev1.KeyToPath{
								{
									Key:  corev1.TLSCertKey,
									Path: corev1.TLSCertKey,
								},
								{
									Key:  corev1.TLSPrivateKeyKey,
									Path: corev1.TLSPrivateKeyKey,
								},
							},
						},
					},
				},
			},
		},
	}
}

func IsTiProxyBackendTLSEnabled(tiproxy *v1alpha1.TiProxy) bool {
	tls := TiProxyBackendTLS(tiproxy)
	return tls != nil && tls.Enabled
}

func IsTiProxyBackendMutualTLSEnabled(tiproxy *v1alpha1.TiProxy) bool {
	tls := TiProxyBackendTLS(tiproxy)
	return tls != nil && tls.Mutual
}

func IsTiProxyBackendInsecureSkipTLSVerify(tiproxy *v1alpha1.TiProxy) bool {
	tls := TiProxyBackendTLS(tiproxy)
	return tls != nil && tls.InsecureSkipTLSVerify
}
