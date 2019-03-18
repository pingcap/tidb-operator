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

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/backup"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func perror(err error, dumplogs func() error) {
	if err != nil {
		if dumplogs != nil {
			err := dumplogs()
			if err != nil {
				glog.Errorf("failed to dump logs: %s", err.Error())
			}
		}
		glog.FatalDepth(1, err)
	}
}

func main() {
	flag.Parse()
	logs.InitLogs()
	defer logs.FlushLogs()

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

	clusterInfo := &tests.TidbClusterInfo{
		Namespace:        "tidb",
		ClusterName:      "demo",
		OperatorTag:      "master",
		PDImage:          "pingcap/pd:v2.1.3",
		TiKVImage:        "pingcap/tikv:v2.1.3",
		TiDBImage:        "pingcap/tidb:v2.1.3",
		StorageClassName: "local-storage",
		Password:         "admin",
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

	dumplogs := func() error { return oa.DumpAllLogs(operatorInfo, []*tests.TidbClusterInfo{clusterInfo}) }

	// scale out: tidb 2 -> 3, tikv 3 -> 5
	podUIDsBeforeScale, err := oa.GetPodUIDMap(clusterInfo)
	perror(err, dumplogs)
	clusterInfo = clusterInfo.ScaleTiDB(3).ScaleTiKV(5)
	perror(oa.ScaleTidbCluster(clusterInfo), dumplogs)
	perror(oa.CheckTidbClusterStatus(clusterInfo), dumplogs)
	perror(oa.CheckScaledCorrectly(clusterInfo, podUIDsBeforeScale), dumplogs)

	// scale in: tikv 5 -> 3
	podUIDsBeforeScale, err = oa.GetPodUIDMap(clusterInfo)
	perror(err, dumplogs)
	clusterInfo = clusterInfo.ScaleTiKV(3)
	perror(oa.ScaleTidbCluster(clusterInfo), dumplogs)
	perror(oa.CheckScaleInSafely(clusterInfo), dumplogs)
	perror(oa.CheckTidbClusterStatus(clusterInfo), dumplogs)
	perror(oa.CheckScaledCorrectly(clusterInfo, podUIDsBeforeScale), dumplogs)

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
		Namespace:        "tidb",
		ClusterName:      "demo2",
		OperatorTag:      "master",
		PDImage:          "pingcap/pd:v2.1.3",
		TiKVImage:        "pingcap/tikv:v2.1.3",
		TiDBImage:        "pingcap/tidb:v2.1.3",
		StorageClassName: "local-storage",
		Password:         "admin",
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
