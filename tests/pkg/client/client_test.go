package client

import (
	"testing"

	glog "k8s.io/klog"
	fclient "github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/client"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/manager"
)

func TestClientConn(t *testing.T) {
	faultCli := fclient.NewClient(fclient.Config{
		Addr: "172.16.5.11:23332",
	})

	if err := faultCli.StopVM(&manager.VM{
		Name: "105",
	}); err != nil {
		glog.Errorf("failed to start node on physical node %v", err)
	}
}
