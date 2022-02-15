// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
)

// GetPDClientFromService gets the pd client from the TidbCluster
func GetPDClientFromService(pdControl pdapi.PDControlInterface, tc *v1alpha1.TidbCluster) pdapi.PDClient {
	if tc.Heterogeneous() && tc.WithoutLocalPD() {
		return pdControl.GetPDClient(pdapi.Namespace(tc.Spec.Cluster.Namespace), tc.Spec.Cluster.Name, tc.IsTLSClusterEnabled(),
			pdapi.TLSCertFromTC(pdapi.Namespace(tc.GetNamespace()), tc.GetName()),
			pdapi.ClusterRef(tc.Spec.Cluster.ClusterDomain),
			pdapi.UseHeadlessService(tc.Spec.AcrossK8s),
		)
	}
	return pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled())
}

// GetPDClient tries to return an available PDClient
// If the pdClient built from the PD service name is unavailable, try to
// build another one with the ClientURL in the PeerMembers.
// ClientURL example:
// ClientURL: https://cluster2-pd-0.cluster2-pd-peer.pingcap.svc.cluster2.local
func GetPDClient(pdControl pdapi.PDControlInterface, tc *v1alpha1.TidbCluster) pdapi.PDClient {
	pdClient := GetPDClientFromService(pdControl, tc)

	if len(tc.Status.PD.PeerMembers) == 0 {
		return pdClient
	}

	_, err := pdClient.GetHealth()
	if err == nil {
		return pdClient
	}

	for _, pdMember := range tc.Status.PD.PeerMembers {
		pdPeerClient := pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled(), pdapi.SpecifyClient(pdMember.ClientURL, pdMember.Name))
		_, err := pdPeerClient.GetHealth()
		if err == nil {
			return pdPeerClient
		}
	}

	return pdClient
}

// NewFakePDClient creates a fake pdclient that is set as the pd client
func NewFakePDClient(pdControl *pdapi.FakePDControl, tc *v1alpha1.TidbCluster) *pdapi.FakePDClient {
	pdClient := pdapi.NewFakePDClient()
	if tc.Spec.Cluster != nil {
		pdControl.SetPDClientWithClusterDomain(pdapi.Namespace(tc.Spec.Cluster.Namespace), tc.Spec.Cluster.Name, tc.Spec.Cluster.ClusterDomain, pdClient)
	}
	pdControl.SetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), pdClient)

	return pdClient
}

// NewFakePDClient creates a fake pdclient that is set as the pd client
func NewFakePDClientWithAddress(pdControl *pdapi.FakePDControl, peerURL string) *pdapi.FakePDClient {
	pdClient := pdapi.NewFakePDClient()
	pdControl.SetPDClientWithAddress(peerURL, pdClient)
	return pdClient
}
