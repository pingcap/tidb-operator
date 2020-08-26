// Copyright 2020 PingCAP, Inc.
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
	"github.com/pingcap/tidb-operator/pkg/dmapi"
)

// GetMasterClient gets the master client from the DMCluster
func GetMasterClient(dmControl dmapi.MasterControlInterface, dc *v1alpha1.DMCluster) dmapi.MasterClient {
	return dmControl.GetMasterClient(dmapi.Namespace(dc.GetNamespace()), dc.GetName(), dc.IsTLSClusterEnabled())
}

// GetMasterClient gets the master client from the DMCluster
func GetMasterPeerClient(dmControl dmapi.MasterControlInterface, dc *v1alpha1.DMCluster, podName string) dmapi.MasterClient {
	return dmControl.GetMasterPeerClient(dmapi.Namespace(dc.GetNamespace()), dc.GetName(), podName, dc.IsTLSClusterEnabled())
}

// NewFakeMasterClient creates a fake master client that is set as the master client
func NewFakeMasterClient(dmControl *dmapi.FakeMasterControl, dc *v1alpha1.DMCluster) *dmapi.FakeMasterClient {
	masterClient := dmapi.NewFakeMasterClient()
	dmControl.SetMasterClient(dmapi.Namespace(dc.GetNamespace()), dc.GetName(), masterClient)
	return masterClient
}
