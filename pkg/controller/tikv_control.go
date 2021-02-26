// Copyright 2021 PingCAP, Inc.
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
	"github.com/pingcap/tidb-operator/pkg/tikvapi"
)

// NewFakeTiKVClient creates a fake tikvclient that is set as the tikv client
func NewFakeTiKVClient(tikvControl *tikvapi.FakeTiKVControl, tc *v1alpha1.TidbCluster, podName string) *tikvapi.FakeTiKVClient {
	tikvClient := tikvapi.NewFakeTiKVClient()
	tikvControl.SetTiKVPodClient(tc.Namespace, tc.Name, podName, tikvClient)
	return tikvClient
}
