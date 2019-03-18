// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package backup

import (
	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/tests"
)

type BackupCase struct {
	operator   tests.OperatorActions
	srcCluster *tests.TidbClusterInfo
	desCluster *tests.TidbClusterInfo
}

func NewBackupCase(operator tests.OperatorActions, srcCluster *tests.TidbClusterInfo, desCluster *tests.TidbClusterInfo) *BackupCase {
	return &BackupCase{
		operator:   operator,
		srcCluster: srcCluster,
		desCluster: desCluster,
	}
}

func (bc *BackupCase) Run() error {

	err := bc.operator.DeployAdHocBackup(bc.srcCluster)
	if err != nil {
		glog.Errorf("cluster:[%s] deploy happen error: %v", bc.srcCluster.ClusterName, err)
		return err
	}

	err = bc.operator.CheckAdHocBackup(bc.srcCluster)
	if err != nil {
		glog.Errorf("cluster:[%s] deploy happen error: %v", bc.srcCluster.ClusterName, err)
		return err
	}

	err = bc.operator.CheckTidbClusterStatus(bc.desCluster)
	if err != nil {
		glog.Errorf("cluster:[%s] deploy faild error: %v", bc.desCluster.ClusterName, err)
		return err
	}

	err = bc.operator.Restore(bc.srcCluster, bc.desCluster)
	if err != nil {
		glog.Errorf("from cluster:[%s] to cluster [%s] restore happen error: %v", bc.srcCluster.ClusterName, bc.desCluster.ClusterName, err)
		return err
	}

	err = bc.operator.CheckRestore(bc.srcCluster, bc.desCluster)
	if err != nil {
		glog.Errorf("from cluster:[%s] to cluster [%s] restore failed error: %v", bc.srcCluster.ClusterName, bc.desCluster.ClusterName, err)
		return err
	}

	bc.srcCluster.Name = "demo-scheduled-backup"

	err = bc.operator.DeployScheduledBackup(bc.srcCluster)
	if err != nil {
		glog.Errorf("cluster:[%s] scheduler happen error: %v", bc.srcCluster.ClusterName, err)
		return err
	}

	err = bc.operator.CheckScheduledBackup(bc.srcCluster)
	if err != nil {
		glog.Errorf("cluster:[%s] scheduler failed error: %v", bc.srcCluster.ClusterName, err)
		return err
	}

	return nil
}
