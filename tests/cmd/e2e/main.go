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

	oa := tests.NewOperatorActions(cli, kubeCli)

	operatorInfo := &tests.OperatorInfo{
		Namespace:      "pingcap",
		ReleaseName:    "operator",
		Image:          "pingcap/tidb-operator:v1.0.0-beta.1-p2",
		Tag:            "v1.0.0-beta.1-p2",
		SchedulerImage: "gcr.io/google-containers/hyperkube:v1.12.1",
		LogLevel:       "2",
	}
	perror(oa.CleanOperator(operatorInfo))
	perror(oa.DeployOperator(operatorInfo))

	clusterInfo := &tests.TidbClusterInfo{
		Namespace:        "tidb",
		ClusterName:      "demo",
		OperatorTag:      "v1.0.0-beta.1-p2",
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

	perror(oa.CleanTidbCluster(clusterInfo))
	perror(oa.DeployTidbCluster(clusterInfo))
	perror(oa.CheckTidbClusterStatus(clusterInfo))

	clusterInfo = clusterInfo.ScaleTiDB(3)
	perror(oa.ScaleTidbCluster(clusterInfo))
	perror(oa.CheckTidbClusterStatus(clusterInfo))

	clusterInfo = clusterInfo.UpgradeAll("v2.1.4")
	perror(oa.UpgradeTidbCluster(clusterInfo))
	perror(oa.CheckTidbClusterStatus(clusterInfo))

	restoreClusterInfo := &tests.TidbClusterInfo{
		Namespace:        "tidb",
		ClusterName:      "demo2",
		OperatorTag:      "v1.0.0-beta.1-p2",
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

	perror(oa.CleanTidbCluster(restoreClusterInfo))
	perror(oa.DeployTidbCluster(restoreClusterInfo))
	perror(oa.CheckTidbClusterStatus(restoreClusterInfo))

}
