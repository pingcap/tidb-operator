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

package tidb

import (
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	stringutil "github.com/pingcap/tidb-operator/pkg/utils/string"
)

type Proxy struct {
	Address          string `toml:"addr"`
	AdvertiseAddress string `toml:"advertise-addr"`
	PDAddress        string `toml:"pd-addrs"`
}

type API struct {
	Address string `toml:"addr"`
}

type TLSConfig struct {
	Cert   string `toml:"cert,omitempty"`
	Key    string `toml:"key,omitempty"`
	CA     string `toml:"ca,omitempty"`
	SkipCA bool   `toml:"skip-ca,omitempty"`
}

type Security struct {
	// ServerSQLTLS is used to provide TLS between TiProxy and MySQL client.
	ServerSQLTLS TLSConfig `toml:"server-tls,omitempty"`
	// ServerHTTPTLS is used to provide TLS on HTTP status port.
	ServerHTTPTLS TLSConfig `toml:"server-http-tls,omitempty"`
	// ClusterTLS is used to access TiDB or PD.
	ClusterTLS TLSConfig `toml:"cluster-tls,omitempty"`
	// SQLTLS is used to access TiDB SQL port.
	SQLTLS TLSConfig `toml:"sql-tls,omitempty"`
	// RequireBackendTLS determines whether requires TLS between TiProxy and TiDB servers.
	// If the TiDB server does not support TLS, clients will report an error when connecting to TiProxy.
	RequireBackendTLS bool `toml:"require-backend-tls,omitempty"`
}

// Config is a subset config of TiProxy.
// Only TiDB Operator managed fields are defined here.
// ref: https://docs.pingcap.com/tidb/stable/tiproxy-configuration/
type Config struct {
	Proxy    Proxy             `toml:"proxy"`
	API      API               `toml:"api"`
	Security Security          `toml:"security"`
	Labels   map[string]string `toml:"labels"`
}

// Ref: https://github.com/pingcap/tidb-operator/blob/f13aaaa40cfd9f1b7c068922e9391a48d21a6528/pkg/manager/member/tiproxy_member_manager.go#L704
func (c *Config) Overlay(cluster *v1alpha1.Cluster, tiproxy *v1alpha1.TiProxy) error {
	if err := c.Validate(); err != nil {
		return err
	}

	c.Proxy.Address = fmt.Sprintf("0.0.0.0:%d", coreutil.TiProxyClientPort(tiproxy))
	c.Proxy.AdvertiseAddress = getAdvertiseAddress(tiproxy)
	c.Proxy.PDAddress = stringutil.RemoveHTTPPrefix(cluster.Status.PD)
	c.API.Address = fmt.Sprintf("0.0.0.0:%d", coreutil.TiProxyStatusPort(tiproxy))

	if coreutil.IsTLSClusterEnabled(cluster) {
		c.Security.ClusterTLS.CA = path.Join(v1alpha1.DirPathClusterTLSTiProxy, corev1.ServiceAccountRootCAKey)
		c.Security.ClusterTLS.Cert = path.Join(v1alpha1.DirPathClusterTLSTiProxy, corev1.TLSCertKey)
		c.Security.ClusterTLS.Key = path.Join(v1alpha1.DirPathClusterTLSTiProxy, corev1.TLSPrivateKeyKey)

		// Use the same cert as cluster TLS to simplify the cert management.
		c.Security.ServerHTTPTLS.CA = path.Join(v1alpha1.DirPathClusterTLSTiProxy, corev1.ServiceAccountRootCAKey)
		c.Security.ServerHTTPTLS.Cert = path.Join(v1alpha1.DirPathClusterTLSTiProxy, corev1.TLSCertKey)
		c.Security.ServerHTTPTLS.Key = path.Join(v1alpha1.DirPathClusterTLSTiProxy, corev1.TLSPrivateKeyKey)
		c.Security.ServerHTTPTLS.SkipCA = true
	}

	if coreutil.IsTiProxyMySQLTLSEnabled(tiproxy) {
		c.Security.ServerSQLTLS.CA = path.Join(v1alpha1.DirPathTiProxySQLTLS, corev1.ServiceAccountRootCAKey)
		c.Security.ServerSQLTLS.Cert = path.Join(v1alpha1.DirPathTiProxySQLTLS, corev1.TLSCertKey)
		c.Security.ServerSQLTLS.Key = path.Join(v1alpha1.DirPathTiProxySQLTLS, corev1.TLSPrivateKeyKey)
		c.Security.ServerSQLTLS.SkipCA = coreutil.IsTiProxyMySQLTLSSkipCA(tiproxy)
	}

	if coreutil.IsTiProxyTiDBTLSEnabled(tiproxy) {
		c.Security.SQLTLS.CA = path.Join(v1alpha1.DirPathTiProxyTiDBTLS, corev1.ServiceAccountRootCAKey)
		c.Security.SQLTLS.Cert = path.Join(v1alpha1.DirPathTiProxyTiDBTLS, corev1.TLSCertKey)
		c.Security.SQLTLS.Key = path.Join(v1alpha1.DirPathTiProxyTiDBTLS, corev1.TLSPrivateKeyKey)
		c.Security.SQLTLS.SkipCA = coreutil.IsTiProxyTiDBTLSSkipCA(tiproxy)
	}

	return nil
}

//nolint:gocyclo // refactor if possible
func (c *Config) Validate() error {
	var fields []string

	if c.Proxy.Address != "" {
		fields = append(fields, "proxy.address")
	}

	if c.Proxy.AdvertiseAddress != "" {
		fields = append(fields, "proxy.advertise-address")
	}

	if c.Proxy.PDAddress != "" {
		fields = append(fields, "proxy.pd-address")
	}

	if c.API.Address != "" {
		fields = append(fields, "api.address")
	}

	if c.Security.ServerSQLTLS.Cert != "" {
		fields = append(fields, "security.server-sql-tls.cert")
	}

	if c.Security.ServerSQLTLS.Key != "" {
		fields = append(fields, "security.server-sql-tls.key")
	}

	if c.Security.ServerSQLTLS.CA != "" {
		fields = append(fields, "security.server-sql-tls.ca")
	}

	if c.Security.ServerHTTPTLS.Cert != "" {
		fields = append(fields, "security.server-http-tls.cert")
	}

	if c.Security.ServerHTTPTLS.Key != "" {
		fields = append(fields, "security.server-http-tls.key")
	}

	if c.Security.ServerHTTPTLS.CA != "" {
		fields = append(fields, "security.server-http-tls.ca")
	}

	if c.Security.ClusterTLS.Cert != "" {
		fields = append(fields, "security.cluster-tls.cert")
	}

	if c.Security.ClusterTLS.Key != "" {
		fields = append(fields, "security.cluster-tls.key")
	}

	if c.Security.ClusterTLS.CA != "" {
		fields = append(fields, "security.cluster-tls.ca")
	}

	if c.Security.SQLTLS.Cert != "" {
		fields = append(fields, "security.sql-tls.cert")
	}

	if c.Security.SQLTLS.Key != "" {
		fields = append(fields, "security.sql-tls.key")
	}

	if c.Security.SQLTLS.CA != "" {
		fields = append(fields, "security.sql-tls.ca")
	}

	if len(fields) == 0 {
		return nil
	}

	return fmt.Errorf("%v: %w", fields, v1alpha1.ErrFieldIsManagedByOperator)
}

func getAdvertiseAddress(tiproxy *v1alpha1.TiProxy) string {
	ns := tiproxy.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	return coreutil.PodName[scope.TiProxy](tiproxy) + "." + tiproxy.Spec.Subdomain + "." + ns + ".svc"
}
