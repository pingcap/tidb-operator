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
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/jinzhu/copier"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/backup"
	"github.com/pingcap/tidb-operator/tests/pkg/workload"
	"github.com/pingcap/tidb-operator/tests/pkg/workload/ddl"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	go func() {
		glog.Info(http.ListenAndServe("localhost:6060", nil))
	}()

	conf := tests.NewConfig()
	conf.ParseOrDie()

	// TODO read these args from config
	beginTidbVersion := "v2.1.0"
	toTidbVersion := "v2.1.4"
	operatorTag := "master"
	operatorImage := "pingcap/tidb-operator:latest"

	cli, kubeCli, err := tests.CreateKubeClient()
	if err != nil {
		glog.Fatalf("failed to create kubernetes clientset: %v", err)
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

	fa := tests.NewFaultTriggerAction(cli, kubeCli, conf)
	if err := testFailover(kubeCli, oa, fa, conf, clusterInfos); err != nil {
		glog.Fatal(err)
	}
}

func testFailover(
	kubeCli kubernetes.Interface,
	oa tests.OperatorActions,
	fa tests.FaultTriggerActions,
	cfg *tests.Config,
	clusters []*tests.TidbClusterInfo,
) error {
	var faultPoint time.Time
	faultNode, err := getFaultNode(kubeCli)
	if err != nil {
		return err
	}

	physicalNode := getPhysicalNode(faultNode, cfg)

	if physicalNode == "" {
		err = errors.New("physical node is empty")
		glog.Error(err.Error())
		return err
	}

	glog.Infof("try to stop node [%s:%s]", physicalNode, faultNode)
	err = wait.Poll(2*time.Second, 10*time.Second, func() (bool, error) {
		err = fa.StopNode(physicalNode, faultNode)
		if err != nil {
			glog.Errorf("failed stop node [%s:%s]: %v", physicalNode, faultNode, err)
			return false, nil
		}
		faultPoint = time.Now()
		return true, nil
	})

	if err != nil {
		glog.Errorf("test failed when trigger a node [%s:%s] stop,error: %v", physicalNode, faultNode, err)
		return err
	}

	// defer start node
	defer func() {
		glog.Infof("defer: start node [%s:%s]", physicalNode, faultNode)
		if err = fa.StartNode(physicalNode, faultNode); err != nil {
			glog.Errorf("failed start node [%s:%s]: %v", physicalNode, faultNode, err)
		}
	}()

	glog.Info("the operator's failover feature should pending some time")
	if err = checkPendingFailover(oa, clusters, &faultPoint); err != nil {
		glog.Errorf("pending failover failed: %v", err)
		return err
	}

	glog.Info("the operator's failover feature should start.")
	if err = checkFailover(oa, clusters, faultNode); err != nil {
		glog.Errorf("check failover failed: %v", err)
		return err
	}

	glog.Info("sleep 3 minutes ......")
	time.Sleep(3 * time.Minute)

	glog.Infof("begin to start node %s %s", physicalNode, faultNode)
	err = wait.Poll(2*time.Second, 10*time.Second, func() (bool, error) {
		err = fa.StartNode(physicalNode, faultNode)
		if err != nil {
			glog.Errorf("failed start node [%s:%s]: %v", physicalNode, faultNode, err)
			return false, nil
		}
		return true, nil
	})

	glog.Infof("begin to check recover after node [%s:%s] start", physicalNode, faultNode)
	if err = checkRecover(oa, clusters); err != nil {
		glog.Infof("check recover failed: %v", err)
		return err
	}

	return nil
}

func checkRecover(oa tests.OperatorActions, clusters []*tests.TidbClusterInfo) error {
	return wait.Poll(5*time.Second, 10*time.Minute, func() (bool, error) {
		var passes []bool
		for i := range clusters {
			err := oa.CheckTidbClusterStatus(clusters[i])
			if err != nil {
				return false, err
			}
			passes = append(passes, true)
		}
		for _, pass := range passes {
			if !pass {
				return false, nil
			}
		}
		return true, nil
	})
}

func checkPendingFailover(oa tests.OperatorActions, clusters []*tests.TidbClusterInfo, faultPoint *time.Time) error {
	return wait.Poll(5*time.Second, 10*time.Minute, func() (bool, error) {
		var passes []bool
		for i := range clusters {
			pass, err := oa.PendingFailover(clusters[i], faultPoint)
			if err != nil {
				return pass, err
			}
			passes = append(passes, pass)
		}
		for _, pass := range passes {
			if !pass {
				return false, nil
			}
		}
		return true, nil
	})
}

func checkFailover(oa tests.OperatorActions, clusters []*tests.TidbClusterInfo, faultNode string) error {
	return wait.Poll(5*time.Second, 25*time.Minute, func() (bool, error) {
		var passes []bool
		for i := range clusters {
			pass, err := oa.CheckFailover(clusters[i], faultNode)
			if err != nil {
				return pass, err
			}
			passes = append(passes, pass)
		}
		for _, pass := range passes {
			if !pass {
				return false, nil
			}
		}
		return true, nil
	})

}

func getMyNodeName() string {
	return os.Getenv("MY_NODE_NAME")
}

func getFaultNode(kubeCli kubernetes.Interface) (string, error) {
	var err error
	var nodes *v1.NodeList
	err = wait.Poll(2*time.Second, 10*time.Second, func() (bool, error) {
		nodes, err = kubeCli.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			glog.Errorf("trigger node stop failed when get all nodes, error: %v", err)
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		glog.Errorf("failed to list nodes: %v", err)
		return "", err
	}

	if len(nodes.Items) <= 1 {
		err := errors.New("the number of nodes cannot be less than 1")
		glog.Error(err.Error())
		return "", err
	}

	myNode := getMyNodeName()
	if myNode == "" {
		err := errors.New("get own node name is empty")
		glog.Error(err.Error())
		return "", err
	}

	index := rand.Intn(len(nodes.Items))
	faultNode := nodes.Items[index].Name
	if faultNode != myNode {
		return faultNode, nil
	}

	if index == 0 {
		faultNode = nodes.Items[index+1].Name
	} else {
		faultNode = nodes.Items[index-1].Name
	}

	if faultNode == myNode {
		err := fmt.Errorf("there are at least two nodes with the name %s", myNode)
		glog.Error(err.Error())
		return "", err
	}

	return faultNode, nil
}

func getPhysicalNode(faultNode string, cfg *tests.Config) string {
	var physicalNode string
	for _, nodes := range cfg.Nodes {
		for _, node := range nodes.Nodes {
			if node == faultNode {
				physicalNode = nodes.PhysicalNode
			}
		}
	}

	return physicalNode
}
