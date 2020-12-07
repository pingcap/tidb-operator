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

// GetPDClientBasic gets the pd client from the TidbCluster
func GetPDClientBasic(pdControl pdapi.PDControlInterface, tc *v1alpha1.TidbCluster) pdapi.PDClient {
	if tc.IsHeterogeneous() {
		if len(tc.Spec.ClusterDomain) > 0 {
			return pdControl.GetClusterRefPDClient(pdapi.Namespace(tc.GetNamespace()), tc.Spec.Cluster.Name, tc.Spec.ClusterDomain, tc.IsTLSClusterEnabled())
		}
		return pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.Spec.Cluster.Name, tc.IsTLSClusterEnabled())
	}
	if len(tc.Spec.ClusterDomain) > 0 {
		return pdControl.GetClusterRefPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.Spec.ClusterDomain, tc.IsTLSClusterEnabled())
	}
	return pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled())
}

// Retry to GetPDClient for multi-cluster
func GetPDClient(pdControl pdapi.PDControlInterface, tc *v1alpha1.TidbCluster) pdapi.PDClient {
	pdClient := GetPDClientBasic(pdControl, tc)
	// Add health check for
	_, err := pdClient.GetHealth()
	if err != nil && len(tc.Status.PD.PeerMembers) > 0 {
		for _, pdMember := range tc.Status.PD.PeerMembers {
			pdClient = pdControl.GetClusterRefPDClientMultiClusterRetry(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.Spec.ClusterDomain, tc.IsTLSClusterEnabled(), pdMember.ClientURL, pdMember.Name)
			_, err := pdClient.GetHealth()
			if err == nil {
				return pdClient
			}
		}
	}
	return pdClient
}

// NewFakePDClient creates a fake pdclient that is set as the pd client
func NewFakePDClient(pdControl *pdapi.FakePDControl, tc *v1alpha1.TidbCluster) *pdapi.FakePDClient {
	pdClient := pdapi.NewFakePDClient()
	if len(tc.Spec.ClusterDomain) > 0 {
		pdControl.SetPDClientWithClusterDomain(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.Spec.ClusterDomain, pdClient)
	} else {
		pdControl.SetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), pdClient)
	}
	return pdClient
}
