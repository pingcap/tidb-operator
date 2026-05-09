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

package dm

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

// Config is a subset config of dm-master.
// Only operator-managed fields are defined here.
type Config struct {
	Name    string `toml:"name"`
	DataDir string `toml:"data-dir"`

	MasterAddr    string `toml:"master-addr"`
	AdvertiseAddr string `toml:"advertise-addr"`

	PeerUrls          string `toml:"peer-urls"`
	AdvertisePeerUrls string `toml:"advertise-peer-urls"`

	InitialCluster      string `toml:"initial-cluster"`
	InitialClusterState string `toml:"initial-cluster-state"`
	Join                string `toml:"join"`

	// SSL fields are top-level in DM master config (not nested under [security])
	SSLCA   string `toml:"ssl-ca"`
	SSLCert string `toml:"ssl-cert"`
	SSLKey  string `toml:"ssl-key"`
}

// Overlay fills in operator-managed config fields.
// peers is the list of all dm-master instances in the same group (used for initial-cluster).
// join is the existing dm-master client address list for scale-out; empty means bootstrap.
func (c *Config) Overlay(cluster *v1alpha1.Cluster, dm *v1alpha1.DM, peers []*v1alpha1.DM, join string) error {
	if err := c.Validate(); err != nil {
		return err
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		c.SSLCA = path.Join(v1alpha1.DirPathClusterTLSDMMaster, corev1.ServiceAccountRootCAKey)
		c.SSLCert = path.Join(v1alpha1.DirPathClusterTLSDMMaster, corev1.TLSCertKey)
		c.SSLKey = path.Join(v1alpha1.DirPathClusterTLSDMMaster, corev1.TLSPrivateKeyKey)
	}

	c.Name = dm.Name
	c.MasterAddr = coreutil.ListenAddress(coreutil.DMPort(dm))
	c.AdvertiseAddr = coreutil.InstanceAdvertiseAddress[scope.DM](cluster, dm, coreutil.DMPort(dm))
	c.PeerUrls = coreutil.ListenURL(cluster, coreutil.DMPeerPort(dm))
	c.AdvertisePeerUrls = coreutil.InstanceAdvertiseURL[scope.DM](cluster, dm, coreutil.DMPeerPort(dm))

	// data dir from DataVolume mounts
	for k := range dm.Spec.DataVolume.Mounts {
		mount := &dm.Spec.DataVolume.Mounts[k]
		if mount.Type == v1alpha1.VolumeMountTypeDMData {
			c.DataDir = mount.MountPath
		}
	}
	if c.DataDir == "" {
		c.DataDir = v1alpha1.VolumeMountDMDataDefaultPath
	}

	initialClusterNum, ok := dm.Annotations[v1alpha1.AnnoKeyInitialClusterNum]
	if !ok {
		// joining an existing cluster
		c.Join = join
	}

	if c.Join == "" {
		num, err := strconv.ParseInt(initialClusterNum, 10, 32)
		if err != nil {
			return fmt.Errorf("cannot parse initial cluster num %v: %w", initialClusterNum, err)
		}
		peers = filterBootstrappingPeers(peers)
		if num != int64(len(peers)) {
			return fmt.Errorf("unexpected number of replicas, expected is %v, current is %v", num, len(peers))
		}
		c.InitialCluster = getInitialCluster(cluster, peers)
		c.InitialClusterState = InitialClusterStateNew
	}

	return nil
}

func filterBootstrappingPeers(peers []*v1alpha1.DM) []*v1alpha1.DM {
	var boot []*v1alpha1.DM
	for _, peer := range peers {
		if _, ok := peer.Annotations[v1alpha1.AnnoKeyInitialClusterNum]; ok {
			boot = append(boot, peer)
		}
	}
	return boot
}

func (c *Config) Validate() error {
	var fields []string

	if c.Name != "" {
		fields = append(fields, "name")
	}
	if c.DataDir != "" {
		fields = append(fields, "data-dir")
	}
	if c.MasterAddr != "" {
		fields = append(fields, "master-addr")
	}
	if c.AdvertiseAddr != "" {
		fields = append(fields, "advertise-addr")
	}
	if c.PeerUrls != "" {
		fields = append(fields, "peer-urls")
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
	if c.Join != "" {
		fields = append(fields, "join")
	}
	if c.SSLCA != "" {
		fields = append(fields, "ssl-ca")
	}
	if c.SSLCert != "" {
		fields = append(fields, "ssl-cert")
	}
	if c.SSLKey != "" {
		fields = append(fields, "ssl-key")
	}

	if len(fields) == 0 {
		return nil
	}

	return fmt.Errorf("%v: %w", fields, v1alpha1.ErrFieldIsManagedByOperator)
}

func getInitialCluster(c *v1alpha1.Cluster, peers []*v1alpha1.DM) string {
	var urls []string
	for _, peer := range peers {
		url := coreutil.InstanceAdvertiseURL[scope.DM](c, peer, coreutil.DMPeerPort(peer))
		urls = append(urls, peer.Name+"="+url)
	}
	return strings.Join(urls, ",")
}
