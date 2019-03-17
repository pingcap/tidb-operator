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
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/backup"
	"github.com/pingcap/tidb-operator/tests/pkg/util"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

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

	gcli, err := util.NewClientFor(cfg)
	if err != nil {
		zap.L().Fatal("new generic client", zap.Error(err))
	}

	src := util.NewOperatorSource("master")
	if err := src.Fetch(false); err != nil {
		zap.L().Fatal("download tidb-operator", zap.Error(err))
	}

	oa := tests.NewOperatorActions(cli, kubeCli)

	operatorInfo := &tests.OperatorInfo{
		Namespace:      "pingcap",
		ReleaseName:    "operator",
		Image:          "pingcap/tidb-operator:latest",
		SchedulerImage: "gcr.io/google-containers/hyperkube:v1.12.1",
		LogLevel:       "2",

		Source: src,
	}

	if err := oa.CleanOperator(operatorInfo); err != nil {
		glog.Fatal(err)
	}
	if err := oa.DeployOperator(operatorInfo); err != nil {
		glog.Fatal(err)
	}

	clusterInfo := &tests.TidbClusterInfo{
		Namespace:        "tidb",
		ClusterName:      "demo",
		PDImage:          "pingcap/pd:v2.1.3",
		TiKVImage:        "pingcap/tikv:v2.1.3",
		TiDBImage:        "pingcap/tidb:v2.1.3",
		StorageClassName: "local-storage",
		Password:         "admin",
		Args:             map[string]string{},

		Source: src,
	}

	if err := oa.CreateSecret(clusterInfo); err != nil {
		glog.Fatal(err)
	}

	if err := oa.CleanTidbCluster(clusterInfo); err != nil {
		glog.Fatal(err)
	}
	if err := oa.DeployTidbCluster(clusterInfo); err != nil {
		glog.Fatal(err)
	}

	// just for example
	err = wait.PollImmediate(15*time.Second, 10*time.Minute, util.CheckAllMembersReady(
		gcli, "tidb", "demo",
		util.ClusterStatus{
			PD:   util.MemberStatus{Replicas: 3, Image: clusterInfo.PDImage},
			TiKV: util.MemberStatus{Replicas: 3, Image: clusterInfo.TiKVImage},
			TiDB: util.MemberStatus{Replicas: 2, Image: clusterInfo.TiDBImage},
		},
	))
	if err != nil {
		zap.L().Fatal("wait for all members ready", zap.Error(err))
	}

	if err := oa.CheckTidbClusterStatus(clusterInfo); err != nil {
		glog.Fatal(err)
	}

	restoreClusterInfo := &tests.TidbClusterInfo{
		Namespace:        "tidb",
		ClusterName:      "demo2",
		PDImage:          "pingcap/pd:v2.1.3",
		TiKVImage:        "pingcap/tikv:v2.1.3",
		TiDBImage:        "pingcap/tidb:v2.1.3",
		StorageClassName: "local-storage",
		Password:         "admin",
		Args:             map[string]string{},

		Source: src,
	}

	if err := oa.CleanTidbCluster(restoreClusterInfo); err != nil {
		glog.Fatal(err)
	}
	if err := oa.DeployTidbCluster(restoreClusterInfo); err != nil {
		glog.Fatal(err)
	}
	if err := oa.CheckTidbClusterStatus(restoreClusterInfo); err != nil {
		glog.Fatal(err)
	}

	backupCase := backup.NewBackupCase(oa, clusterInfo, restoreClusterInfo)

	if err := backupCase.Run(); err != nil {
		glog.Fatal(err)
	}
}

func init() {
	logger, err := util.NewGLogDev()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
}
