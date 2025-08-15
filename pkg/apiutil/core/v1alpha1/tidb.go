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

// MySQLTLSSecretName returns the secret name used in TiDB server for the TLS between TiDB server and MySQL client.
func MySQLTLSSecretName(db *v1alpha1.TiDB) string {
	prefix, _ := NamePrefixAndSuffix(db)
	return prefix + "-tidb-server-secret"
}

// IsMySQLTLSEnabled returns whether the TLS between TiDB server and MySQL client is enabled.
func IsMySQLTLSEnabled(db *v1alpha1.TiDB) bool {
	return db.Spec.Security != nil && db.Spec.Security.TLS != nil && db.Spec.Security.TLS.MySQL != nil && db.Spec.Security.TLS.MySQL.Enabled
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
		return TLSClusterSecretName[scope.TiDB](db)
	}
	return ""
}
