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

package ticdc

import (
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

// Config is a subset config of ticdc
// Only our managed fields are defined in here.
// ref: https://github.com/pingcap/tidb/blob/master/pkg/config/config.go
type Config struct {
	Addr          string `toml:"addr"`
	AdvertiseAddr string `toml:"advertise-addr"`

	Security Security `toml:"security"`
}

type Security struct {
	CAPath   string `toml:"ca-path"`
	CertPath string `toml:"cert-path"`
	KeyPath  string `toml:"key-path"`
}

func (c *Config) Overlay(cluster *v1alpha1.Cluster, ticdc *v1alpha1.TiCDC) error {
	if err := c.Validate(); err != nil {
		return err
	}

	c.Addr = getAddr(ticdc)
	c.AdvertiseAddr = getAdvertiseAdd(ticdc)

	if coreutil.IsTLSClusterEnabled(cluster) {
		c.Security.CAPath = path.Join(v1alpha1.DirPathClusterTLSTiCDC, corev1.ServiceAccountRootCAKey)
		c.Security.CertPath = path.Join(v1alpha1.DirPathClusterTLSTiCDC, corev1.TLSCertKey)
		c.Security.KeyPath = path.Join(v1alpha1.DirPathClusterTLSTiCDC, corev1.TLSPrivateKeyKey)
	}

	return nil
}

//nolint:gocyclo // refactor if possible
func (c *Config) Validate() error {
	var fields []string

	if c.Addr != "" {
		fields = append(fields, "addr")
	}
	if c.AdvertiseAddr != "" {
		fields = append(fields, "advertise-addr")
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

func getAddr(ticdc *v1alpha1.TiCDC) string {
	return fmt.Sprintf("[::]:%d", coreutil.TiCDCPort(ticdc))
}

func getAdvertiseAdd(ticdc *v1alpha1.TiCDC) string {
	ns := ticdc.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	return fmt.Sprintf("%s.%s.%s:%d", coreutil.PodName[scope.TiCDC](ticdc),
		ticdc.Spec.Subdomain, ns, coreutil.TiCDCPort(ticdc))
}
