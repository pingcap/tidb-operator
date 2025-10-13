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

func TiDBGroupClientPort(dbg *v1alpha1.TiDBGroup) int32 {
	if dbg.Spec.Template.Spec.Server.Ports.Client != nil {
		return dbg.Spec.Template.Spec.Server.Ports.Client.Port
	}
	return v1alpha1.DefaultTiDBPortClient
}

func TiDBGroupStatusPort(dbg *v1alpha1.TiDBGroup) int32 {
	if dbg.Spec.Template.Spec.Server.Ports.Status != nil {
		return dbg.Spec.Template.Spec.Server.Ports.Status.Port
	}
	return v1alpha1.DefaultTiDBPortStatus
}

func TiDBClientPort(db *v1alpha1.TiDB) int32 {
	if db.Spec.Server.Ports.Client != nil {
		return db.Spec.Server.Ports.Client.Port
	}
	return v1alpha1.DefaultTiDBPortClient
}

func TiDBStatusPort(db *v1alpha1.TiDB) int32 {
	if db.Spec.Server.Ports.Status != nil {
		return db.Spec.Server.Ports.Status.Port
	}
	return v1alpha1.DefaultTiDBPortStatus
}

func TiDBGroupMySQLTLS(dbg *v1alpha1.TiDBGroup) *v1alpha1.TLS {
	sec := dbg.Spec.Template.Spec.Security
	if sec != nil && sec.TLS != nil && sec.TLS.MySQL != nil {
		return sec.TLS.MySQL
	}

	return nil
}

func TiDBGroupMySQLCertKeyPairSecretName(dbg *v1alpha1.TiDBGroup) string {
	tls := TiDBGroupMySQLTLS(dbg)
	if tls != nil && tls.CertKeyPair != nil {
		return tls.CertKeyPair.Name
	}
	return dbg.GetName() + "-tidb-server-secret"
}

func TiDBGroupMySQLCASecretName(dbg *v1alpha1.TiDBGroup) string {
	tls := TiDBGroupMySQLTLS(dbg)
	if tls != nil && tls.CA != nil {
		return tls.CA.Name
	}
	return dbg.GetName() + "-tidb-server-secret"
}

func IsTiDBGroupMySQLTLSEnabled(dbg *v1alpha1.TiDBGroup) bool {
	tls := TiDBGroupMySQLTLS(dbg)
	return tls != nil && tls.Enabled
}

func TiDBMySQLTLS(db *v1alpha1.TiDB) *v1alpha1.TLS {
	sec := db.Spec.Security
	if sec != nil && sec.TLS != nil && sec.TLS.MySQL != nil {
		return sec.TLS.MySQL
	}

	return nil
}

// TiDBMySQLCertKeyPairSecretName returns the secret name used in TiDB server
// for the TLS between TiDB server and MySQL client.
func TiDBMySQLCertKeyPairSecretName(db *v1alpha1.TiDB) string {
	tls := TiDBMySQLTLS(db)
	if tls != nil && tls.CertKeyPair != nil {
		return tls.CertKeyPair.Name
	}
	prefix, _ := NamePrefixAndSuffix(db)
	return prefix + "-tidb-server-secret"
}

// TiDBMySQLCASecretName returns the secret name for TiDB server
// to authenticate MySQL client.
func TiDBMySQLCASecretName(db *v1alpha1.TiDB) string {
	tls := TiDBMySQLTLS(db)
	if tls != nil && tls.CA != nil {
		return tls.CA.Name
	}
	prefix, _ := NamePrefixAndSuffix(db)
	return prefix + "-tidb-server-secret"
}

func TiDBMySQLTLSVolume(db *v1alpha1.TiDB) *corev1.Volume {
	tls := TiDBMySQLTLS(db)
	certKeyPair := TiDBMySQLCertKeyPairSecretName(db)

	// tls is disabled
	if tls == nil {
		return nil
	}

	if tls.ClientAuth == v1alpha1.ClientAuthTypeNoClientCert {
		return &corev1.Volume{
			Name: v1alpha1.VolumeNameMySQLTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: certKeyPair,
				},
			},
		}
	}

	ca := TiDBMySQLCASecretName(db)

	if ca == certKeyPair {
		return &corev1.Volume{
			Name: v1alpha1.VolumeNameMySQLTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: ca,
				},
			},
		}
	}

	return &corev1.Volume{
		Name: v1alpha1.VolumeNameMySQLTLS,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: ca,
							},
							Items: []corev1.KeyToPath{
								{
									Key:  corev1.ServiceAccountRootCAKey,
									Path: corev1.ServiceAccountRootCAKey,
								},
							},
						},
					},
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: certKeyPair,
							},
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

// IsTiDBMySQLTLSEnabled returns whether the TLS between TiDB server and MySQL client is enabled.
func IsTiDBMySQLTLSEnabled(db *v1alpha1.TiDB) bool {
	tls := TiDBMySQLTLS(db)
	return tls != nil && tls.Enabled
}

func IsTiDBMySQLTLSNoClientCert(db *v1alpha1.TiDB) bool {
	tls := TiDBMySQLTLS(db)
	return tls != nil && tls.ClientAuth == v1alpha1.ClientAuthTypeNoClientCert
}

func IsTokenBasedAuthEnabled(db *v1alpha1.TiDB) bool {
	return db.Spec.Security != nil && db.Spec.Security.AuthToken != nil
}

func AuthTokenJWKSSecretName(db *v1alpha1.TiDB) string {
	if IsTokenBasedAuthEnabled(db) {
		return db.Spec.Security.AuthToken.JWKs.Name
	}
	return ""
}

func IsSEMEnabled(db *v1alpha1.TiDB) bool {
	return db.Spec.Security != nil && db.Spec.Security.SEM != nil
}

func SEMConfigMapName(db *v1alpha1.TiDB) string {
	if IsSEMEnabled(db) {
		return db.Spec.Security.SEM.Config.Name
	}
	return ""
}

func IsSeparateSlowLogEnabled(db *v1alpha1.TiDB) bool {
	return db.Spec.SlowLog != nil
}

// SessionTokenSigningCertSecretName returns the secret name used in TiDB server for the session token signing cert.
// If the session token signing cert is not specified, it will return the TLS cluster secret name if TLS is enabled.
// If TLS is not enabled, it will return an empty string.
func SessionTokenSigningCertSecretName(cluster *v1alpha1.Cluster, db *v1alpha1.TiDB) string {
	if cluster.Spec.Security != nil && cluster.Spec.Security.SessionTokenSigningCertKeyPair != nil {
		return cluster.Spec.Security.SessionTokenSigningCertKeyPair.Name
	}
	if IsTLSClusterEnabled(cluster) {
		return ClusterCertKeyPairSecretName[scope.TiDB](db)
	}
	return ""
}
