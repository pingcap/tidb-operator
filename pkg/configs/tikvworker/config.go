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

package tikvworker

import (
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
)

type Config struct {
	Addr     string   `toml:"addr"`
	PD       PD       `toml:"pd"`
	Security Security `toml:"security"`
}

type PD struct {
	Endpoints []string `toml:"endpoints"`
}

type Security struct {
	// CAPath is the path of file that contains list of trusted SSL CAs.
	CAPath string `toml:"ca-path"`
	// CertPath is the path of file that contains X509 certificate in PEM format.
	CertPath string `toml:"cert-path"`
	// KeyPath is the path of file that contains X509 key in PEM format.
	KeyPath string `toml:"key-path"`
}

func (c *Config) Overlay(cluster *v1alpha1.Cluster, w *v1alpha1.TiKVWorker) error {
	if err := c.Validate(); err != nil {
		return err
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		c.Security.CAPath = path.Join(v1alpha1.DirPathClusterTLSTiKVWorker, corev1.ServiceAccountRootCAKey)
		c.Security.CertPath = path.Join(v1alpha1.DirPathClusterTLSTiKVWorker, corev1.TLSCertKey)
		c.Security.KeyPath = path.Join(v1alpha1.DirPathClusterTLSTiKVWorker, corev1.TLSPrivateKeyKey)
	}

	c.Addr = coreutil.ListenAddress(coreutil.TiKVWorkerAPIPort(w))
	c.PD.Endpoints = []string{cluster.Status.PD}

	return nil
}

func (c *Config) Validate() error {
	fields := []string{}

	if c.Addr != "" {
		fields = append(fields, "addr")
	}

	if len(c.PD.Endpoints) != 0 {
		fields = append(fields, "pd.endpoints")
	}

	if c.Security.CAPath != "" {
		fields = append(fields, "security.ca-path")
	}
	if c.Security.CertPath != "" {
		fields = append(fields, "security.cert-path")
	}
	if c.Security.KeyPath != "" {
		fields = append(fields, "security.key-path")
	}

	if len(fields) == 0 {
		return nil
	}

	return fmt.Errorf("%v: %w", fields, v1alpha1.ErrFieldIsManagedByOperator)
}
