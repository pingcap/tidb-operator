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

package scheduler

import (
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

// Config is the configuration for the scheduler.
// TODO: reduce duplicated code
type Config struct {
	Name                string   `toml:"name"`
	ListenAddr          string   `toml:"listen-addr"`
	AdvertiseListenAddr string   `toml:"advertise-listen-addr"`
	BackendEndpoints    string   `toml:"backend-endpoints"`
	Security            Security `toml:"security"`
}

type Security struct {
	// CACertPath is the path of file that contains list of trusted SSL CAs.
	CACertPath string `toml:"cacert-path"`
	// CertPath is the path of file that contains X509 certificate in PEM format.
	CertPath string `toml:"cert-path"`
	// KeyPath is the path of file that contains X509 key in PEM format.
	KeyPath string `toml:"key-path"`
}

func (c *Config) Overlay(cluster *v1alpha1.Cluster, s *v1alpha1.Scheduler) error {
	if err := c.Validate(); err != nil {
		return err
	}

	scheme := "http"
	if coreutil.IsTLSClusterEnabled(cluster) {
		scheme = "https"
		c.Security.CACertPath = path.Join(v1alpha1.DirPathClusterTLSScheduler, corev1.ServiceAccountRootCAKey)
		c.Security.CertPath = path.Join(v1alpha1.DirPathClusterTLSScheduler, corev1.TLSCertKey)
		c.Security.KeyPath = path.Join(v1alpha1.DirPathClusterTLSScheduler, corev1.TLSPrivateKeyKey)
	}

	c.ListenAddr = getClientURLs(s, scheme)
	c.AdvertiseListenAddr = getAdvertiseClientURLs(s, scheme)
	c.BackendEndpoints = cluster.Status.PD

	return nil
}

func (c *Config) Validate() error {
	var fields []string

	if c.ListenAddr != "" {
		fields = append(fields, "listen-addr")
	}
	if c.AdvertiseListenAddr != "" {
		fields = append(fields, "advertise-listen-addr")
	}
	if c.BackendEndpoints != "" {
		fields = append(fields, "backend-endpoints")
	}

	if c.Security.CACertPath != "" {
		fields = append(fields, "security.cacert-path")
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

func getClientURLs(s *v1alpha1.Scheduler, scheme string) string {
	return fmt.Sprintf("%s://[::]:%d", scheme, coreutil.SchedulerClientPort(s))
}

func getAdvertiseClientURLs(s *v1alpha1.Scheduler, scheme string) string {
	ns := s.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	return fmt.Sprintf("%s://%s.%s.%s:%d", scheme, coreutil.PodName[scope.Scheduler](s), s.Spec.Subdomain, ns, coreutil.SchedulerClientPort(s))
}

