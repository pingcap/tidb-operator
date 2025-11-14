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

package pd

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

const (
	InitialClusterStateNew      = "new"
	InitialClusterStateExisting = "existing"
)

// Config is a subset config of pd
// Only our managed fields are defined in here
type Config struct {
	Name    string `toml:"name"`
	DataDir string `toml:"data-dir"`

	ClientUrls          string `toml:"client-urls"`
	PeerUrls            string `toml:"peer-urls"`
	AdvertiseClientUrls string `toml:"advertise-client-urls"`
	AdvertisePeerUrls   string `toml:"advertise-peer-urls"`

	InitialCluster      string `toml:"initial-cluster"`
	InitialClusterState string `toml:"initial-cluster-state"`
	InitialClusterToken string `toml:"initial-cluster-token"`
	Join                string `toml:"join"`

	Security Security `toml:"security"`
}

type Security struct {
	// CAPath is the path of file that contains list of trusted SSL CAs.
	CAPath string `toml:"cacert-path"`
	// CertPath is the path of file that contains X509 certificate in PEM format.
	CertPath string `toml:"cert-path"`
	// KeyPath is the path of file that contains X509 key in PEM format.
	KeyPath string `toml:"key-path"`
}

func (c *Config) Overlay(cluster *v1alpha1.Cluster, pd *v1alpha1.PD, peers []*v1alpha1.PD) error {
	if err := c.Validate(); err != nil {
		return err
	}

	scheme := "http"
	if coreutil.IsTLSClusterEnabled(cluster) {
		scheme = "https"
		c.Security.CAPath = path.Join(v1alpha1.DirPathClusterTLSPD, corev1.ServiceAccountRootCAKey)
		c.Security.CertPath = path.Join(v1alpha1.DirPathClusterTLSPD, corev1.TLSCertKey)
		c.Security.KeyPath = path.Join(v1alpha1.DirPathClusterTLSPD, corev1.TLSPrivateKeyKey)
	}

	c.Name = pd.Name
	c.ClientUrls = getClientURLs(pd, scheme)
	c.AdvertiseClientUrls = GetAdvertiseClientURLs(pd, scheme)
	c.PeerUrls = getPeerURLs(pd, scheme)
	c.AdvertisePeerUrls = getAdvertisePeerURLs(pd, scheme)

	for i := range pd.Spec.Volumes {
		vol := &pd.Spec.Volumes[i]
		for k := range vol.Mounts {
			mount := &vol.Mounts[k]
			if mount.Type == v1alpha1.VolumeMountTypePDData {
				c.DataDir = mount.MountPath
			}
		}
	}

	if c.DataDir == "" {
		c.DataDir = v1alpha1.VolumeMountPDDataDefaultPath
	}

	c.InitialClusterToken = pd.Spec.Cluster.Name
	initialClusterNum, ok := pd.Annotations[v1alpha1.AnnoKeyInitialClusterNum]
	if !ok {
		c.Join = cluster.Status.PD
	}

	if c.Join == "" {
		num, err := strconv.ParseInt(initialClusterNum, 10, 32)
		if err != nil {
			return fmt.Errorf("cannot parse initial cluster num %v: %w", initialClusterNum, err)
		}
		if num != int64(len(peers)) {
			return fmt.Errorf("unexpected number of replicas, expected is %v, current is %v", num, len(peers))
		}
		c.InitialCluster = getInitialCluster(peers, scheme)
		c.InitialClusterState = InitialClusterStateNew
	}

	return nil
}

func (c *Config) Validate() error {
	fields := []string{}

	if c.Name != "" {
		fields = append(fields, "name")
	}
	if c.DataDir != "" {
		fields = append(fields, "data-dir")
	}
	if c.ClientUrls != "" {
		fields = append(fields, "client-urls")
	}
	if c.PeerUrls != "" {
		fields = append(fields, "peer-urls")
	}
	if c.AdvertiseClientUrls != "" {
		fields = append(fields, "advertise-client-urls")
	}
	if c.AdvertisePeerUrls != "" {
		fields = append(fields, "advertise-peer-urls")
	}

	if c.InitialCluster != "" {
		fields = append(fields, "initial-cluster")
	}
	if c.InitialClusterState != "" {
		fields = append(fields, "initial-cluster-state")
	}
	if c.InitialClusterToken != "" {
		fields = append(fields, "initial-cluster-token")
	}
	if c.Join != "" {
		fields = append(fields, "join")
	}

	if c.Security.CAPath != "" {
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

func getClientURLs(pd *v1alpha1.PD, scheme string) string {
	return fmt.Sprintf("%s://[::]:%d", scheme, coreutil.PDClientPort(pd))
}

func GetAdvertiseClientURLs(pd *v1alpha1.PD, scheme string) string {
	ns := pd.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	host := coreutil.PodName[scope.PD](pd) + "." + pd.Spec.Subdomain + "." + ns
	return fmt.Sprintf("%s://%s:%d", scheme, host, coreutil.PDClientPort(pd))
}

func getPeerURLs(pd *v1alpha1.PD, scheme string) string {
	return fmt.Sprintf("%s://[::]:%d", scheme, coreutil.PDPeerPort(pd))
}

func getAdvertisePeerURLs(pd *v1alpha1.PD, scheme string) string {
	ns := pd.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	host := coreutil.PodName[scope.PD](pd) + "." + pd.Spec.Subdomain + "." + ns
	return fmt.Sprintf("%s://%s:%d", scheme, host, coreutil.PDPeerPort(pd))
}

func getInitialCluster(peers []*v1alpha1.PD, scheme string) string {
	urls := []string{}
	for _, peer := range peers {
		url := getAdvertisePeerURLs(peer, scheme)
		urls = append(urls, peer.Name+"="+url)
	}

	return strings.Join(urls, ",")
}
