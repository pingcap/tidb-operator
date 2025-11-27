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

package coreutil

import (
	"fmt"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

func PDGroupClientPort(pdg *v1alpha1.PDGroup) int32 {
	if pdg.Spec.Template.Spec.Server.Ports.Client != nil {
		return pdg.Spec.Template.Spec.Server.Ports.Client.Port
	}
	return v1alpha1.DefaultPDPortClient
}

func PDGroupPeerPort(pdg *v1alpha1.PDGroup) int32 {
	if pdg.Spec.Template.Spec.Server.Ports.Peer != nil {
		return pdg.Spec.Template.Spec.Server.Ports.Peer.Port
	}
	return v1alpha1.DefaultPDPortPeer
}

func PDClientPort(pd *v1alpha1.PD) int32 {
	if pd.Spec.Server.Ports.Client != nil {
		return pd.Spec.Server.Ports.Client.Port
	}
	return v1alpha1.DefaultPDPortClient
}

func PDPeerPort(pd *v1alpha1.PD) int32 {
	if pd.Spec.Server.Ports.Peer != nil {
		return pd.Spec.Server.Ports.Peer.Port
	}
	return v1alpha1.DefaultPDPortPeer
}

// PDServiceURL returns the service url of PD
func PDServiceURL(c *v1alpha1.Cluster, pdg *v1alpha1.PDGroup) string {
	svc := ClusterPD(c)
	ns := c.Namespace
	port := int32(v1alpha1.DefaultPDPortClient)
	if !IsFeatureEnabled(c, metav1alpha1.MultiPDGroup) {
		svc = pdg.Name + "-pd"
		port = PDGroupClientPort(pdg)
	}
	host := fmt.Sprintf("%s.%s:%d", svc, ns, port)

	return hostToURL(host, IsTLSClusterEnabled(c))
}
