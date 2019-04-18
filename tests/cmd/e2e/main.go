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
// limitations under the License.package spec

package main

import (
	"fmt"
	_ "net/http/pprof"
	"time"

	"github.com/golang/glog"
	"github.com/jinzhu/copier"
	"k8s.io/apiserver/pkg/util/logs"

	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/backup"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	conf := tests.ParseConfigOrDie()
	conf.ChartDir = "/charts"

	cli, kubeCli := client.NewCliOrDie()
	oa := tests.NewOperatorActions(cli, kubeCli, 5*time.Second, conf)

	// start a http server in goruntine
	go oa.StartValidatingAdmissionWebhookServerOrDie()

	operatorInfo := &tests.OperatorConfig{
		Namespace:          "pingcap",
		ReleaseName:        "operator",
		Image:              conf.OperatorImage,
		Tag:                conf.OperatorTag,
		SchedulerImage:     "mirantis/hypokube",
		SchedulerTag:       "final",
		LogLevel:           "2",
		WebhookServiceName: "webhook-service",
		WebhookSecretName:  "webhook-secret",
		WebhookConfigName:  "webhook-config",
	}

	initTidbVersion, err := conf.GetTiDBVersion()
	if err != nil {
		glog.Fatal(err)
	}
	// create database and table and insert a column for test backup and restore
	initSql := `"create database record;use record;create table test(t char(32))"`

	name1 := "e2e-cluster1"
	name2 := "e2e-cluster2"
	clusterInfos := []*tests.TidbClusterConfig{
		{
			Namespace:        name1,
			ClusterName:      name1,
			OperatorTag:      conf.OperatorTag,
			PDImage:          fmt.Sprintf("pingcap/pd:%s", initTidbVersion),
			TiKVImage:        fmt.Sprintf("pingcap/tikv:%s", initTidbVersion),
			TiDBImage:        fmt.Sprintf("pingcap/tidb:%s", initTidbVersion),
			StorageClassName: "local-storage",
			Password:         "admin",
			InitSql:          initSql,
			UserName:         "root",
			InitSecretName:   fmt.Sprintf("%s-set-secret", name1),
			BackupSecretName: fmt.Sprintf("%s-backup-secret", name1),
			BackupPVC:        "backup-pvc",
			Resources: map[string]string{
				"pd.resources.limits.cpu":        "1000m",
				"pd.resources.limits.memory":     "2Gi",
				"pd.resources.requests.cpu":      "200m",
				"pd.resources.requests.memory":   "1Gi",
				"tikv.resources.limits.cpu":      "2000m",
				"tikv.resources.limits.memory":   "4Gi",
				"tikv.resources.requests.cpu":    "200m",
				"tikv.resources.requests.memory": "1Gi",
				"tidb.resources.limits.cpu":      "2000m",
				"tidb.resources.limits.memory":   "4Gi",
				"tidb.resources.requests.cpu":    "200m",
				"tidb.resources.requests.memory": "1Gi",
			},
			Args:    map[string]string{},
			Monitor: true,
		},
		{
			Namespace:        name2,
			ClusterName:      name2,
			OperatorTag:      conf.OperatorTag,
			PDImage:          fmt.Sprintf("pingcap/pd:%s", initTidbVersion),
			TiKVImage:        fmt.Sprintf("pingcap/tikv:%s", initTidbVersion),
			TiDBImage:        fmt.Sprintf("pingcap/tidb:%s", initTidbVersion),
			StorageClassName: "local-storage",
			Password:         "admin",
			InitSql:          initSql,
			UserName:         "root",
			InitSecretName:   fmt.Sprintf("%s-set-secret", name2),
			BackupSecretName: fmt.Sprintf("%s-backup-secret", name2),
			BackupPVC:        "backup-pvc",
			Resources: map[string]string{
				"pd.resources.limits.cpu":        "1000m",
				"pd.resources.limits.memory":     "2Gi",
				"pd.resources.requests.cpu":      "200m",
				"pd.resources.requests.memory":   "1Gi",
				"tikv.resources.limits.cpu":      "2000m",
				"tikv.resources.limits.memory":   "4Gi",
				"tikv.resources.requests.cpu":    "200m",
				"tikv.resources.requests.memory": "1Gi",
				"tidb.resources.limits.cpu":      "2000m",
				"tidb.resources.limits.memory":   "4Gi",
				"tidb.resources.requests.cpu":    "200m",
				"tidb.resources.requests.memory": "1Gi",
			},
			Args:    map[string]string{},
			Monitor: true,
		},
	}

	defer func() {
		oa.DumpAllLogs(operatorInfo, clusterInfos)
	}()

	// deploy operator
	if err := oa.CleanOperator(operatorInfo); err != nil {
		oa.DumpAllLogs(operatorInfo, nil)
		glog.Fatal(err)
	}
	if err = oa.DeployOperator(operatorInfo); err != nil {
		oa.DumpAllLogs(operatorInfo, nil)
		glog.Fatal(err)
	}

	// deploy tidbclusters
	for _, clusterInfo := range clusterInfos {
		if err = oa.CleanTidbCluster(clusterInfo); err != nil {
			glog.Fatal(err)
		}
		if err = oa.DeployTidbCluster(clusterInfo); err != nil {
			glog.Fatal(err)
		}
	}

	for _, clusterInfo := range clusterInfos {
		if err = oa.CheckTidbClusterStatus(clusterInfo); err != nil {
			glog.Fatal(err)
		}
	}

	// before upgrade cluster, register webhook first
	oa.RegisterWebHookAndServiceOrDie(operatorInfo)

	// upgrade test
	upgradeTidbVersions := conf.GetUpgradeTidbVersions()
	for _, upgradeTidbVersion := range upgradeTidbVersions {
		for _, clusterInfo := range clusterInfos {
			clusterInfo = clusterInfo.UpgradeAll(upgradeTidbVersion)
			if err = oa.UpgradeTidbCluster(clusterInfo); err != nil {
				glog.Fatal(err)
			}
		}
		for _, clusterInfo := range clusterInfos {
			if err = oa.CheckTidbClusterStatus(clusterInfo); err != nil {
				glog.Fatal(err)
			}
		}
	}

	// after upgrade cluster, clean webhook
	oa.CleanWebHookAndService(operatorInfo)

	for _, clusterInfo := range clusterInfos {
		clusterInfo = clusterInfo.ScaleTiDB(3).ScaleTiKV(5).ScalePD(5)
		if err := oa.ScaleTidbCluster(clusterInfo); err != nil {
			glog.Fatal(err)
		}
	}
	for _, clusterInfo := range clusterInfos {
		if err := oa.CheckTidbClusterStatus(clusterInfo); err != nil {
			glog.Fatal(err)
		}
	}

	for _, clusterInfo := range clusterInfos {
		clusterInfo = clusterInfo.ScalePD(3)
		if err := oa.ScaleTidbCluster(clusterInfo); err != nil {
			glog.Fatal(err)
		}
	}
	for _, clusterInfo := range clusterInfos {
		if err := oa.CheckTidbClusterStatus(clusterInfo); err != nil {
			glog.Fatal(err)
		}
	}

	for _, clusterInfo := range clusterInfos {
		clusterInfo = clusterInfo.ScaleTiKV(3)
		if err := oa.ScaleTidbCluster(clusterInfo); err != nil {
			glog.Fatal(err)
		}
	}
	for _, clusterInfo := range clusterInfos {
		if err := oa.CheckTidbClusterStatus(clusterInfo); err != nil {
			glog.Fatal(err)
		}
	}

	for _, clusterInfo := range clusterInfos {
		clusterInfo = clusterInfo.ScaleTiDB(1)
		if err := oa.ScaleTidbCluster(clusterInfo); err != nil {
			glog.Fatal(err)
		}
	}
	for _, clusterInfo := range clusterInfos {
		if err := oa.CheckTidbClusterStatus(clusterInfo); err != nil {
			glog.Fatal(err)
		}
	}

	// backup and restore
	backupClusterInfo := clusterInfos[0]
	restoreClusterInfo := &tests.TidbClusterConfig{}
	copier.Copy(restoreClusterInfo, backupClusterInfo)
	restoreClusterInfo.ClusterName = restoreClusterInfo.ClusterName + "-other"
	restoreClusterInfo.InitSecretName = fmt.Sprintf("%s-set-secret", restoreClusterInfo.ClusterName)
	restoreClusterInfo.BackupSecretName = fmt.Sprintf("%s-backup-secret", restoreClusterInfo.ClusterName)

	if err = oa.CleanTidbCluster(restoreClusterInfo); err != nil {
		glog.Fatal(err)
	}
	if err = oa.DeployTidbCluster(restoreClusterInfo); err != nil {
		glog.Fatal(err)
	}
	if err = oa.CheckTidbClusterStatus(restoreClusterInfo); err != nil {
		glog.Fatal(err)
	}

	backupCase := backup.NewBackupCase(oa, backupClusterInfo, restoreClusterInfo)

	if err := backupCase.Run(); err != nil {
		glog.Fatal(err)
	}

	//clean temp dirs when e2e success
	err = conf.CleanTempDirs()
	if err != nil {
		glog.Errorf("failed to clean temp dirs, this error can be ignored.")
	}
	glog.Infof("\nFinished.")
}
