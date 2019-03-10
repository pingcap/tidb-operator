// Copyright 2018 PingCAP, Inc.
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

package backup

import (
//	"fmt"
	"time"
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

	//begin to insert data
	bc.operator.BeginInsertDataTo(bc.srcCluster)
	time.Sleep(5 * time.Second)
	bc.operator.StopInsertDataTo(bc.srcCluster)

	//first is check adhoc backup case
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

	//rearrange the cluster
	err = bc.operator.ForceDeploy(bc.desCluster)
	if err != nil {
		glog.Errorf("cluster:[%s] deploy happen error: %v", bc.desCluster.ClusterName, err)
		return err
	}

	err = bc.operator.CheckTidbClusterStatus(bc.desCluster)
	if err != nil {
		glog.Errorf("cluster:[%s] deploy faild error: %v", bc.desCluster.ClusterName, err)
		return err
	}

	err = bc.operator.Restore(bc.srcCluster,bc.desCluster)
	if err != nil {
		glog.Errorf("from cluster:[%s] to cluster [%s] restore happen error: %v",bc.srcCluster.ClusterName, bc.desCluster.ClusterName, err)
		return err
	}

	err = bc.operator.CheckRestore(bc.srcCluster,bc.desCluster)
	if err != nil {
		glog.Errorf("from cluster:[%s] to cluster [%s] restore failed error: %v",bc.srcCluster.ClusterName, bc.desCluster.ClusterName, err)
		return err
	}

	//then check sechduler backup case
	bc.operator.BeginInsertDataTo(bc.srcCluster)
	time.Sleep(5 * time.Second)
	bc.operator.StopInsertDataTo(bc.srcCluster)

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

	//rearrange the cluster
	err = bc.operator.ForceDeploy(bc.desCluster)
	if err != nil {
		glog.Errorf("cluster:[%s] deploy happen error: %v", bc.desCluster.ClusterName, err)
		return err
	}

	err = bc.operator.CheckTidbClusterStatus(bc.desCluster)
	if err != nil {
		glog.Errorf("cluster:[%s] deploy faild error: %v", bc.desCluster.ClusterName, err)
		return err
	}

	err = bc.operator.Restore(bc.srcCluster,bc.desCluster)
	if err != nil {
		glog.Errorf("from cluster:[%s] to cluster [%s] restore happen error: %v",bc.srcCluster.ClusterName, bc.desCluster.ClusterName, err)
		return err
	}

	err = bc.operator.CheckRestore(bc.srcCluster,bc.desCluster)
	if err != nil {
		glog.Errorf("from cluster:[%s] to cluster [%s] restore failed error: %v",bc.srcCluster.ClusterName, bc.desCluster.ClusterName, err)
		return err
	}

/*	//at last check incremental backup case
	err = bc.operator.DeployIncrementalBackup(bc.srcCluster, bc.desCluster)
	if err != nil {
		return err
	}

	err = bc.operator.CheckIncrementalBackup(bc.srcCluster)
	if err != nil {
		return err
	}

	glog.Infof("waiting 60 second for binlog to work")
	time.Sleep(60 * time.Second)

	glog.Infof("cluster[%s] begin insert data")
	go bc.operator.BeginInsertDataTo(bc.srcCluster)

	time.Sleep(30 * time.Second)

	glog.Infof("cluster[%s] stop insert data")
	bc.operator.StopInsertDataTo(bc.srcCluster)

	time.Sleep(5 * time.Second)

	srcCount, err := bc.srcCluster.QueryCount()
	if err != nil {
		return err
	}
	desCount, err := bc.desCluster.QueryCount()
	if err != nil {
		return err
	}
	if srcCount != desCount {
		return fmt.Errorf("cluster:[%s] the src cluster data[%d] is not equals des cluster data[%d]", bc.srcCluster.FullName(), srcCount, desCount)
	}*/

	return nil
}
