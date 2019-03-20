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
	"flag"
	"net/http"
	_ "net/http/pprof"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/backup"
	"github.com/pingcap/tidb-operator/tests/pkg/workload"
	"github.com/pingcap/tidb-operator/tests/pkg/workload/ddl"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func perror(err error) {
	if err != nil {
		glog.Fatal(err)
	}
}

func main() {
	flag.Parse()
	logs.InitLogs()
	defer logs.FlushLogs()

	go func() {
		glog.Info(http.ListenAndServe("localhost:6060", nil))
	}()

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

	oa := tests.NewOperatorActions(cli, kubeCli, "/logDir")

	operatorInfo := &tests.OperatorInfo{
		Namespace:      "pingcap",
		ReleaseName:    "operator",
		Image:          "pingcap/tidb-operator:latest",
		Tag:            "master",
		SchedulerImage: "gcr.io/google-containers/hyperkube:v1.12.1",
		LogLevel:       "2",
	}
	if err := oa.CleanOperator(operatorInfo); err != nil {
		oa.DumpAllLogs(operatorInfo, nil)
		glog.Fatal(err)
	}
	if err = oa.DeployOperator(operatorInfo); err != nil {
		oa.DumpAllLogs(operatorInfo, nil)
		glog.Fatal(err)
	}

	// create database and table and insert a column for test backup and restore
	initSql := `"create database record;use record;create table test(t char(32))"`

	clusterInfo := &tests.TidbClusterInfo{
		BackupPVC:        "test-backup",
		Namespace:        "tidb",
		ClusterName:      "demo",
		OperatorTag:      "master",
		PDImage:          "pingcap/pd:v2.1.0",
		TiKVImage:        "pingcap/tikv:v2.1.0",
		TiDBImage:        "pingcap/tidb:v2.1.0",
		StorageClassName: "local-storage",
		Password:         "admin",
		InitSql:          initSql,
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
		Args: map[string]string{},
	}

	if err = oa.CleanTidbCluster(clusterInfo); err != nil {
		oa.DumpAllLogs(operatorInfo, []*tests.TidbClusterInfo{clusterInfo})
		glog.Fatal(err)
	}
	if err = oa.DeployTidbCluster(clusterInfo); err != nil {
		oa.DumpAllLogs(operatorInfo, []*tests.TidbClusterInfo{clusterInfo})
		glog.Fatal(err)
	}
	if err = oa.CheckTidbClusterStatus(clusterInfo); err != nil {
		oa.DumpAllLogs(operatorInfo, []*tests.TidbClusterInfo{clusterInfo})
		glog.Fatal(err)
	}

	err = workload.Run(func() error {
		clusterInfo = clusterInfo.ScaleTiDB(3).ScaleTiKV(5).ScalePD(5)
		if err := oa.ScaleTidbCluster(clusterInfo); err != nil {
			return err
		}
		if err := oa.CheckTidbClusterStatus(clusterInfo); err != nil {
			return err
		}

		clusterInfo = clusterInfo.ScalePD(3)
		if err := oa.ScaleTidbCluster(clusterInfo); err != nil {
			return err
		}
		if err := oa.CheckTidbClusterStatus(clusterInfo); err != nil {
			return err
		}

		clusterInfo = clusterInfo.ScaleTiKV(3)
		if err := oa.ScaleTidbCluster(clusterInfo); err != nil {
			return err
		}
		if err := oa.CheckTidbClusterStatus(clusterInfo); err != nil {
			return err
		}

		clusterInfo = clusterInfo.ScaleTiDB(1)
		if err := oa.ScaleTidbCluster(clusterInfo); err != nil {
			return err
		}
		if err := oa.CheckTidbClusterStatus(clusterInfo); err != nil {
			return err
		}

		return nil
	}, ddl.New(clusterInfo.DSN("test"), 1, 1))

	if err != nil {
		oa.DumpAllLogs(operatorInfo, []*tests.TidbClusterInfo{clusterInfo})
		glog.Fatal(err)
	}

	clusterInfo = clusterInfo.UpgradeAll("v2.1.4")
	if err = oa.UpgradeTidbCluster(clusterInfo); err != nil {
		oa.DumpAllLogs(operatorInfo, []*tests.TidbClusterInfo{clusterInfo})
		glog.Fatal(err)
	}
	if err = oa.CheckTidbClusterStatus(clusterInfo); err != nil {
		oa.DumpAllLogs(operatorInfo, []*tests.TidbClusterInfo{clusterInfo})
		glog.Fatal(err)
	}

	restoreClusterInfo := &tests.TidbClusterInfo{
		BackupPVC:        "test-backup",
		Namespace:        "tidb",
		ClusterName:      "demo2",
		OperatorTag:      "master",
		PDImage:          "pingcap/pd:v2.1.0",
		TiKVImage:        "pingcap/tikv:v2.1.0",
		TiDBImage:        "pingcap/tidb:v2.1.0",
		StorageClassName: "local-storage",
		Password:         "admin",
		InitSql:          initSql,
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
		Args: map[string]string{},
	}

	if err = oa.CleanTidbCluster(restoreClusterInfo); err != nil {
		oa.DumpAllLogs(operatorInfo, []*tests.TidbClusterInfo{clusterInfo, restoreClusterInfo})
		glog.Fatal(err)
	}
	if err = oa.DeployTidbCluster(restoreClusterInfo); err != nil {
		oa.DumpAllLogs(operatorInfo, []*tests.TidbClusterInfo{clusterInfo, restoreClusterInfo})
		glog.Fatal(err)
	}
	if err = oa.CheckTidbClusterStatus(restoreClusterInfo); err != nil {
		oa.DumpAllLogs(operatorInfo, []*tests.TidbClusterInfo{clusterInfo, restoreClusterInfo})
		glog.Fatal(err)
	}

	backupCase := backup.NewBackupCase(oa, clusterInfo, restoreClusterInfo)

	if err := backupCase.Run(); err != nil {
		oa.DumpAllLogs(operatorInfo, []*tests.TidbClusterInfo{clusterInfo, restoreClusterInfo})
		glog.Fatal(err)
	}
}
