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
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	stringutil "github.com/pingcap/tidb-operator/pkg/utils/string"
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

	InitializeSQLFile          string            `toml:"initialize-sql-file"`
	GracefulWaitBeforeShutdown int               `toml:"graceful-wait-before-shutdown"`
	ServerLabels               map[string]string `toml:"labels"`
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

	// If tiproxy is enabled, need to set these configs.
	// https://docs.pingcap.com/tidb/stable/tidb-configuration-file/#session-token-signing-cert-new-in-v640
	SessionTokenSigningKey  string `toml:"session-token-signing-key"`
	SessionTokenSigningCert string `toml:"session-token-signing-cert"`
}

type Log struct {
	SlowQueryFile string `toml:"slow-query-file"`
}

func (c *Config) Overlay(cluster *v1alpha1.Cluster, tidb *v1alpha1.TiDB, fg features.Gates) error {
	if err := c.Validate(coreutil.IsSeparateSlowLogEnabled(tidb)); err != nil {
		return err
	}

	c.Store = "tikv" // always use tikv
	c.AdvertiseAddress = getAdvertiseAddress(tidb)
	c.Host = "::"
	c.Path = stringutil.RemoveHTTPPrefix(cluster.Status.PD)

	if coreutil.IsTiDBMySQLTLSEnabled(tidb) {
		c.Security.SSLCert = path.Join(v1alpha1.DirPathMySQLTLS, corev1.TLSCertKey)
		c.Security.SSLKey = path.Join(v1alpha1.DirPathMySQLTLS, corev1.TLSPrivateKeyKey)
		if !coreutil.IsTiDBMySQLTLSNoClientCert(tidb) {
			c.Security.SSLCA = path.Join(v1alpha1.DirPathMySQLTLS, corev1.ServiceAccountRootCAKey)
		}
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		c.Security.ClusterSSLCA = path.Join(v1alpha1.DirPathClusterTLSTiDB, corev1.ServiceAccountRootCAKey)
		c.Security.ClusterSSLCert = path.Join(v1alpha1.DirPathClusterTLSTiDB, corev1.TLSCertKey)
		c.Security.ClusterSSLKey = path.Join(v1alpha1.DirPathClusterTLSTiDB, corev1.TLSPrivateKeyKey)
	}

	if fg.Enabled(metav1alpha1.SessionTokenSigning) {
		c.Security.SessionTokenSigningKey = path.Join(v1alpha1.DirPathTiDBSessionTokenSigningTLS, corev1.TLSPrivateKeyKey)
		c.Security.SessionTokenSigningCert = path.Join(v1alpha1.DirPathTiDBSessionTokenSigningTLS, corev1.TLSCertKey)
	}

	c.Log.SlowQueryFile = getSlowQueryFile(tidb)

	if cluster.Spec.BootstrapSQL != nil {
		c.InitializeSQLFile = path.Join(v1alpha1.DirPathBootstrapSQL, v1alpha1.FileNameBootstrapSQL)
	}

	if coreutil.IsTokenBasedAuthEnabled(tidb) {
		c.Security.AuthTokenJwks = path.Join(v1alpha1.DirPathTiDBAuthToken, v1alpha1.FileNameTiDBAuthTokenJWKS)
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

	if len(c.ServerLabels) > 0 {
		fields = append(fields, "labels")
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
	return coreutil.PodName[scope.TiDB](tidb) + "." + tidb.Spec.Subdomain + "." + ns + ".svc"
}

func getSlowQueryFile(tidb *v1alpha1.TiDB) string {
	if !coreutil.IsSeparateSlowLogEnabled(tidb) {
		return ""
	}

	var dir string
	for i := range tidb.Spec.Volumes {
		vol := &tidb.Spec.Volumes[i]
		for k := range vol.Mounts {
			mount := &vol.Mounts[k]

			if mount.Type != v1alpha1.VolumeMountTypeTiDBSlowLog {
				continue
			}

			dir = mount.MountPath
		}
	}

	if dir == "" {
		dir = v1alpha1.DirPathTiDBSlowLogDefault
	}

	return path.Join(dir, v1alpha1.FileNameTiDBSlowLog)
}
