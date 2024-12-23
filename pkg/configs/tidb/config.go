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
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

const (
	// defaultGracefulWaitBeforeShutdownInSeconds is the default value of the tidb config `graceful-wait-before-shutdown`,
	// which is set by the operator if not set by the user, for graceful shutdown.
	// Note that the default value is zero in tidb-server.
	defaultGracefulWaitBeforeShutdownInSeconds = 30
)

// Config is a subset config of tidb
// Only our managed fields are defined in here.
// ref: https://github.com/pingcap/tidb/blob/master/pkg/config/config.go
type Config struct {
	Store            string `toml:"store"`
	AdvertiseAddress string `toml:"advertise-address"`
	Host             string `toml:"host"`
	Path             string `toml:"path"`

	Security Security `toml:"security"`

	Log Log `toml:"log"`

	InitializeSQLFile          string `toml:"initialize-sql-file"`
	GracefulWaitBeforeShutdown int    `toml:"graceful-wait-before-shutdown"`
}

type Security struct {
	// TLS config for communication between TiDB server and MySQL client
	SSLCA   string `toml:"ssl-ca"`
	SSLCert string `toml:"ssl-cert"`
	SSLKey  string `toml:"ssl-key"`

	// mTLS config
	ClusterSSLCA   string `toml:"cluster-ssl-ca"`
	ClusterSSLCert string `toml:"cluster-ssl-cert"`
	ClusterSSLKey  string `toml:"cluster-ssl-key"`

	// tidb_auth_token
	AuthTokenJwks string `toml:"auth-token-jwks"`
}

type Log struct {
	SlowQueryFile string `toml:"slow-query-file"`
}

func (c *Config) Overlay(cluster *v1alpha1.Cluster, tidb *v1alpha1.TiDB) error {
	if err := c.Validate(tidb.IsSeparateSlowLogEnabled()); err != nil {
		return err
	}

	c.Store = "tikv" // always use tikv
	c.AdvertiseAddress = getAdvertiseAddress(tidb)
	c.Host = "::"
	c.Path = removeHTTPPrefix(cluster.Status.PD)

	if tidb.IsMySQLTLSEnabled() {
		// TODO(csuzhangxc): disable Client Authn
		c.Security.SSLCA = path.Join(v1alpha1.TiDBSQLTLSMountPath, corev1.ServiceAccountRootCAKey)
		c.Security.SSLCert = path.Join(v1alpha1.TiDBSQLTLSMountPath, corev1.TLSCertKey)
		c.Security.SSLKey = path.Join(v1alpha1.TiDBSQLTLSMountPath, corev1.TLSPrivateKeyKey)
	}

	if cluster.IsTLSClusterEnabled() {
		c.Security.ClusterSSLCA = path.Join(v1alpha1.TiDBClusterTLSMountPath, corev1.ServiceAccountRootCAKey)
		c.Security.ClusterSSLCert = path.Join(v1alpha1.TiDBClusterTLSMountPath, corev1.TLSCertKey)
		c.Security.ClusterSSLKey = path.Join(v1alpha1.TiDBClusterTLSMountPath, corev1.TLSPrivateKeyKey)
	}

	c.Log.SlowQueryFile = getSlowQueryFile(tidb)

	if tidb.IsBootstrapSQLEnabled() {
		c.InitializeSQLFile = path.Join(v1alpha1.BootstrapSQLFilePath, v1alpha1.BootstrapSQLFileName)
	}

	if tidb.IsTokenBasedAuthEnabled() {
		c.Security.AuthTokenJwks = path.Join(v1alpha1.TiDBAuthTokenPath, v1alpha1.TiDBAuthTokenJWKS)
	}

	// If not set, use default value.
	if c.GracefulWaitBeforeShutdown == 0 {
		c.GracefulWaitBeforeShutdown = defaultGracefulWaitBeforeShutdownInSeconds
	}

	return nil
}

//nolint:gocyclo // refactor if possible
func (c *Config) Validate(separateSlowLog bool) error {
	var fields []string

	if c.Store != "" {
		fields = append(fields, "store")
	}
	if c.AdvertiseAddress != "" {
		fields = append(fields, "advertise-address")
	}
	if c.Host != "" {
		fields = append(fields, "host")
	}
	if c.Path != "" {
		fields = append(fields, "path")
	}

	// valid security fields
	if c.Security.SSLCA != "" {
		fields = append(fields, "security.ssl-ca")
	}
	if c.Security.SSLCert != "" {
		fields = append(fields, "security.ssl-cert")
	}
	if c.Security.SSLKey != "" {
		fields = append(fields, "security.ssl-key")
	}
	if c.Security.ClusterSSLCA != "" {
		fields = append(fields, "security.cluster-ssl-ca")
	}
	if c.Security.ClusterSSLCert != "" {
		fields = append(fields, "security.cluster-ssl-cert")
	}
	if c.Security.ClusterSSLKey != "" {
		fields = append(fields, "security.cluster-ssl-key")
	}
	if c.Security.AuthTokenJwks != "" {
		fields = append(fields, "security.auth-token-jwks")
	}

	if separateSlowLog && c.Log.SlowQueryFile != "" {
		fields = append(fields, "log.slow-query-file")
	}

	if c.InitializeSQLFile != "" {
		fields = append(fields, "initialize-sql-file")
	}

	if len(fields) == 0 {
		return nil
	}

	return fmt.Errorf("%v: %w", fields, v1alpha1.ErrFieldIsManagedByOperator)
}

func getAdvertiseAddress(tidb *v1alpha1.TiDB) string {
	ns := tidb.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	return tidb.PodName() + "." + tidb.Spec.Subdomain + "." + ns + ".svc"
}

func removeHTTPPrefix(url string) string {
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	return url
}

func getSlowQueryFile(tidb *v1alpha1.TiDB) string {
	if !tidb.IsSeparateSlowLogEnabled() {
		return ""
	} else if tidb.Spec.SlowLog == nil || tidb.Spec.SlowLog.VolumeName == "" {
		return path.Join(v1alpha1.TiDBDefaultSlowLogDir, v1alpha1.TiDBSlowLogFileName)
	}

	for i := range tidb.Spec.Volumes {
		vol := &tidb.Spec.Volumes[i]
		if vol.Name == tidb.Spec.SlowLog.VolumeName {
			return path.Join(vol.Path, v1alpha1.TiDBSlowLogFileName)
		}
	}

	return "" // should not reach here
}
