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

package client

import (
	"testing"

	fclient "github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/client"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/manager"
	"k8s.io/klog"
)

func TestClientConn(t *testing.T) {
	faultCli := fclient.NewClient(fclient.Config{
		Addr: "172.16.5.11:23332",
	})

	if err := faultCli.StopVM(&manager.VM{
		Name: "105",
	}); err != nil {
		klog.Errorf("failed to start node on physical node %v", err)
	}
}
