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
	"net/http"
	_ "net/http/pprof"

	"github.com/golang/glog"
	"github.com/jinzhu/copier"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/backup"
	"github.com/pingcap/tidb-operator/tests/pkg/workload"
	"github.com/pingcap/tidb-operator/tests/pkg/workload/ddl"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	conf := tests.NewConfig()
	err := conf.Parse()
	if err != nil {
		glog.Fatalf("failed to parse config: %v", err)
	}

	go func() {
		glog.Info(http.ListenAndServe("localhost:6060", nil))
	}()

	// TODO read these args from config
	beginTidbVersion := "v2.1.0"
	toTidbVersion := "v2.1.4"
	operatorTag := "master"
	operatorImage := "pingcap/tidb-operator:latest"

	cfg, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("failed to get config: %v", err)
	}
	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("failed to create Clientset: %v", err)
	}
	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}

	oa := tests.NewOperatorActions(cli, kubeCli, conf)

	operatorInfo := &tests.OperatorInfo{
		Namespace:      "pingcap",
		ReleaseName:    "operator",
		Image:          operatorImage,
		Tag:            operatorTag,
		SchedulerImage: "gcr.io/google-containers/hyperkube:v1.12.1",
		LogLevel:       "2",
	}

	// create database and table and insert a column for test backup and restore
	initSql := `"create database record;use record;create table test(t char(32))"`

	clusterInfos := []*tests.TidbClusterInfo{
		{
			Namespace:        "e2e-cluster1",
			ClusterName:      "e2e-cluster1",
			OperatorTag:      operatorTag,
			PDImage:          fmt.Sprintf("pingcap/pd:%s", beginTidbVersion),
			TiKVImage:        fmt.Sprintf("pingcap/tikv:%s", beginTidbVersion),
			TiDBImage:        fmt.Sprintf("pingcap/tidb:%s", beginTidbVersion),
			StorageClassName: "local-storage",
			Password:         "admin",
			InitSql:          initSql,
			UserName:         "root",
			InitSecretName:   "demo-set-secret",
			BackupSecretName: "demo-backup-secret",
			BackupPVC:        "test-backup",
			Resources: map[string]string{
				"pd.resources.limits.cpu":        "1000m",
				"pd.resources.limits.memory":     "2Gi",
				"pd.resources.requests.cpu":      "200m",
				"pd.resources.requests.memory":   "1Gi",
				"tikv.resources.limits.cpu":      "2000m",
				"tikv.resources.limits.memory":   "4Gi",
				"tikv.resources.requests.cpu":    "1000m",
				"tikv.resources.requests.memory": "2Gi",
				"tidb.resources.limits.cpu":      "2000m",
				"tidb.resources.limits.memory":   "4Gi",
				"tidb.resources.requests.cpu":    "500m",
				"tidb.resources.requests.memory": "1Gi",
			},
			Args:    map[string]string{},
			Monitor: true,
		},
		{
			Namespace:        "e2e-cluster2",
			ClusterName:      "e2e-cluster2",
			OperatorTag:      "master",
			PDImage:          fmt.Sprintf("pingcap/pd:%s", beginTidbVersion),
			TiKVImage:        fmt.Sprintf("pingcap/tikv:%s", beginTidbVersion),
			TiDBImage:        fmt.Sprintf("pingcap/tidb:%s", beginTidbVersion),
			StorageClassName: "local-storage",
			Password:         "admin",
			InitSql:          initSql,
			UserName:         "root",
			InitSecretName:   "demo-set-secret",
			BackupSecretName: "demo-backup-secret",
			BackupPVC:        "test-backup",
			Resources: map[string]string{
				"pd.resources.limits.cpu":        "1000m",
				"pd.resources.limits.memory":     "2Gi",
				"pd.resources.requests.cpu":      "200m",
				"pd.resources.requests.memory":   "1Gi",
				"tikv.resources.limits.cpu":      "2000m",
				"tikv.resources.limits.memory":   "4Gi",
				"tikv.resources.requests.cpu":    "1000m",
				"tikv.resources.requests.memory": "2Gi",
				"tidb.resources.limits.cpu":      "2000m",
				"tidb.resources.limits.memory":   "4Gi",
				"tidb.resources.requests.cpu":    "500m",
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

	var workloads []workload.Workload
	for _, clusterInfo := range clusterInfos {
		workload := ddl.New(clusterInfo.DSN("test"), 1, 1)
		workloads = append(workloads, workload)
	}

	err = workload.Run(func() error {

		for _, clusterInfo := range clusterInfos {
			clusterInfo = clusterInfo.ScaleTiDB(3).ScaleTiKV(5).ScalePD(5)
			if err := oa.ScaleTidbCluster(clusterInfo); err != nil {
				return err
			}
		}
		for _, clusterInfo := range clusterInfos {
			if err := oa.CheckTidbClusterStatus(clusterInfo); err != nil {
				return err
			}
		}

		for _, clusterInfo := range clusterInfos {
			clusterInfo = clusterInfo.ScalePD(3)
			if err := oa.ScaleTidbCluster(clusterInfo); err != nil {
				return err
			}
		}
		for _, clusterInfo := range clusterInfos {
			if err := oa.CheckTidbClusterStatus(clusterInfo); err != nil {
				return err
			}
		}

		for _, clusterInfo := range clusterInfos {
			clusterInfo = clusterInfo.ScaleTiKV(3)
			if err := oa.ScaleTidbCluster(clusterInfo); err != nil {
				return err
			}
		}
		for _, clusterInfo := range clusterInfos {
			if err := oa.CheckTidbClusterStatus(clusterInfo); err != nil {
				return err
			}
		}

		for _, clusterInfo := range clusterInfos {
			clusterInfo = clusterInfo.ScaleTiDB(1)
			if err := oa.ScaleTidbCluster(clusterInfo); err != nil {
				return err
			}
		}
		for _, clusterInfo := range clusterInfos {
			if err := oa.CheckTidbClusterStatus(clusterInfo); err != nil {
				return err
			}
		}

		return nil
	}, workloads...)

	if err != nil {
		glog.Fatal(err)
	}

	for _, clusterInfo := range clusterInfos {
		if err = oa.CheckTidbClusterStatus(clusterInfo); err != nil {
			glog.Fatal(err)
		}
	}

	for _, clusterInfo := range clusterInfos {
		clusterInfo = clusterInfo.UpgradeAll(toTidbVersion)
		if err = oa.UpgradeTidbCluster(clusterInfo); err != nil {
			glog.Fatal(err)
		}
	}

	for _, clusterInfo := range clusterInfos {
		if err = oa.CheckTidbClusterStatus(clusterInfo); err != nil {
			glog.Fatal(err)
		}
	}

	// backup and restore
	backupClusterInfo := clusterInfos[0]
	restoreClusterInfo := &tests.TidbClusterInfo{}
	copier.Copy(restoreClusterInfo, backupClusterInfo)
	restoreClusterInfo.ClusterName = restoreClusterInfo.ClusterName + "-restore"

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
}
