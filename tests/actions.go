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

package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/pingcap/errors"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/tests/pkg/blockwriter"
	"github.com/pingcap/tidb-operator/tests/pkg/util"
	"k8s.io/api/apps/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultTableNum    int = 64
	defaultConcurrency     = 512
	defaultBatchSize       = 100
	defaultRawSize         = 100
)

func NewOperatorActions(cli versioned.Interface, kubeCli kubernetes.Interface, logDir string) OperatorActions {
	return &operatorActions{
		cli:       cli,
		kubeCli:   kubeCli,
		pdControl: controller.NewDefaultPDControl(),
		logDir:    logDir,
	}
}

const (
	DefaultPollTimeout  time.Duration = 10 * time.Minute
	DefaultPollInterval time.Duration = 10 * time.Second
)

const (
	grafanaUsername = "admin"
	grafanaPassword = "admin"
)

type OperatorActions interface {
	DeployOperator(info *OperatorInfo) error
	CleanOperator(info *OperatorInfo) error
	UpgradeOperator(info *OperatorInfo) error
	DumpAllLogs(info *OperatorInfo, clusterInfos []*TidbClusterInfo) error
	DeployTidbCluster(info *TidbClusterInfo) error
	CleanTidbCluster(info *TidbClusterInfo) error
	CheckTidbClusterStatus(info *TidbClusterInfo) error
	BeginInsertDataTo(info *TidbClusterInfo) error
	StopInsertDataTo(info *TidbClusterInfo) error
	ScaleTidbCluster(info *TidbClusterInfo) error
	UpgradeTidbCluster(info *TidbClusterInfo) error
	DeployAdHocBackup(info *TidbClusterInfo) error
	CheckAdHocBackup(info *TidbClusterInfo) error
	DeployScheduledBackup(info *TidbClusterInfo) error
	CheckScheduledBackup(info *TidbClusterInfo) error
	DeployIncrementalBackup(from *TidbClusterInfo, to *TidbClusterInfo) error
	CheckIncrementalBackup(info *TidbClusterInfo) error
	Restore(from *TidbClusterInfo, to *TidbClusterInfo) error
	CheckRestore(from *TidbClusterInfo, to *TidbClusterInfo) error
	ForceDeploy(info *TidbClusterInfo) error
	CreateSecret(info *TidbClusterInfo) error
}

type FaultTriggerActions interface {
	StopNode(nodeName string) error
	StartNode(nodeName string) error
	StopEtcd() error
	StartEtcd() error
	StopKubeAPIServer() error
	StartKubeAPIServer() error
	StopKubeControllerManager() error
	StartKubeControllerManager() error
	StopKubeScheduler() error
	StartKubeScheduler() error
	StopKubelet(nodeName string) error
	StartKubelet(nodeName string) error
	StopKubeProxy(nodeName string) error
	StartKubeProxy(nodeName string) error
	DiskCorruption(nodeName string) error
	NetworkPartition(fromNode, toNode string) error
	NetworkDelay(fromNode, toNode string) error
	DockerCrash(nodeName string) error
}

type operatorActions struct {
	cli       versioned.Interface
	kubeCli   kubernetes.Interface
	pdControl controller.PDControlInterface
	logDir    string
}

var _ = OperatorActions(&operatorActions{})

type OperatorInfo struct {
	Namespace      string
	ReleaseName    string
	Image          string
	Tag            string
	SchedulerImage string
	LogLevel       string
}

type TidbClusterInfo struct {
	Namespace        string
	ClusterName      string
	OperatorTag      string
	PDImage          string
	TiKVImage        string
	TiDBImage        string
	StorageClassName string
	Password         string
	RecordCount      string
	InsertBatchSize  string
	Resources        map[string]string
	Args             map[string]string
	blockWriter      *blockwriter.BlockWriterCase
	Monitor          bool
}

func (tc *TidbClusterInfo) HelmSetString() string {

	// add a database and table for test
	initSql := `"create database record;use record;create table test(t char(32));"`

	set := map[string]string{
		"clusterName":             tc.ClusterName,
		"pd.storageClassName":     tc.StorageClassName,
		"tikv.storageClassName":   tc.StorageClassName,
		"tidb.storageClassName":   tc.StorageClassName,
		"tidb.password":           tc.Password,
		"pd.maxStoreDownTime":     "5m",
		"pd.image":                tc.PDImage,
		"tikv.image":              tc.TiKVImage,
		"tidb.image":              tc.TiDBImage,
		"tidb.passwordSecretName": "set-secret",
		"tidb.initSql":            initSql,
		"monitor.create":          strconv.FormatBool(tc.Monitor),
	}

	for k, v := range tc.Resources {
		set[k] = v
	}
	for k, v := range tc.Args {
		set[k] = v
	}

	arr := make([]string, 0, len(set))
	for k, v := range set {
		arr = append(arr, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(arr, ",")
}

func (oa *operatorActions) DeployOperator(info *OperatorInfo) error {
	if err := cloneOperatorRepo(); err != nil {
		return err
	}
	if err := checkoutTag(info.Tag); err != nil {
		return err
	}

	cmd := fmt.Sprintf(`helm install /charts/%s/tidb-operator \
		--name %s \
		--namespace %s \
		--set operatorImage=%s \
		--set controllerManager.autoFailover=true \
		--set scheduler.kubeSchedulerImage=%s \
		--set controllerManager.logLevel=%s \
		--set scheduler.logLevel=2`,
		info.Tag,
		info.ReleaseName,
		info.Namespace,
		info.Image,
		info.SchedulerImage,
		info.LogLevel)
	glog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to deploy operator: %v, %s", err, string(res))
	}

	return nil
}

func (oa *operatorActions) CleanOperator(info *OperatorInfo) error {
	res, err := exec.Command("helm", "del", "--purge", info.ReleaseName).CombinedOutput()
	if err == nil || !releaseIsNotFound(err) {
		return nil
	}
	return fmt.Errorf("failed to clear operator: %v, %s", err, string(res))
}

func (oa *operatorActions) UpgradeOperator(info *OperatorInfo) error {
	if err := checkoutTag(info.Tag); err != nil {
		return err
	}

	cmd := fmt.Sprintf(`helm upgrade %s /charts/%s/tidb-operator
		--set operatorImage=%s`,
		info.ReleaseName, info.Tag,
		info.Image)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to upgrade operator to: %s, %v, %s", info.Image, err, string(res))
	}
	return nil
}

func (oa *operatorActions) DeployTidbCluster(info *TidbClusterInfo) error {
	glog.Infof("begin to deploy tidb cluster cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)
	defer func() {
		glog.Infof("deploy tidb cluster end cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)
	}()
	cmd := fmt.Sprintf("helm install /charts/%s/tidb-cluster  --name %s --namespace %s --set-string %s",
		info.OperatorTag, info.ClusterName, info.Namespace, info.HelmSetString())
	if res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to deploy tidbcluster: %s/%s, %v, %s",
			info.Namespace, info.ClusterName, err, string(res))
	}

	// init blockWriter case
	info.blockWriter = blockwriter.NewBlockWriterCase(blockwriter.Config{
		TableNum:    defaultTableNum,
		Concurrency: defaultConcurrency,
		BatchSize:   defaultBatchSize,
		RawSize:     defaultRawSize,
	})

	return nil
}

func (oa *operatorActions) CleanTidbCluster(info *TidbClusterInfo) error {
	glog.Infof("begin to clean tidb cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)
	defer func() {
		glog.Infof("clean tidb cluster end cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)
	}()
	charts := []string{
		info.ClusterName,
		fmt.Sprintf("%s-backup", info.ClusterName),
		fmt.Sprintf("%s-restore", info.ClusterName),
	}
	for _, chartName := range charts {
		res, err := exec.Command("helm", "del", "--purge", chartName).CombinedOutput()
		if err != nil && releaseIsNotFound(err) {
			return fmt.Errorf("failed to delete chart: %s/%s, %v, %s",
				info.Namespace, chartName, err, string(res))
		}
	}

	setStr := label.New().Instance(info.ClusterName).String()

	resources := []string{"pvc"}
	for _, resource := range resources {
		if res, err := exec.Command("kubectl", "delete", resource, "-n", info.Namespace, "-l",
			setStr).CombinedOutput(); err != nil {
			return fmt.Errorf("failed to delete %s: %v, %s", resource, err, string(res))
		}
	}

	patchPVCmd := fmt.Sprintf(`kubectl get pv -l %s=%s,%s=%s --output=name | xargs -I {} \
		kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'`,
		label.NamespaceLabelKey, info.Namespace, label.InstanceLabelKey, info.ClusterName)
	glog.Info(patchPVCmd)
	if res, err := exec.Command("/bin/sh", "-c", patchPVCmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to patch pv: %v, %s", err, string(res))
	}

	pollFn := func() (bool, error) {
		if res, err := exec.Command("kubectl", "get", "po", "--output=name", "-n", info.Namespace, "-l", setStr).
			CombinedOutput(); err != nil || len(res) != 0 {
			glog.Infof("waiting for tidbcluster: %s/%s pods deleting, %v, [%s]",
				info.Namespace, info.ClusterName, err, string(res))
			return false, nil
		}

		pvCmd := fmt.Sprintf("kubectl get pv -l %s=%s,%s=%s 2>/dev/null|grep Released",
			label.NamespaceLabelKey, info.Namespace, label.InstanceLabelKey, info.ClusterName)
		glog.Info(pvCmd)
		if res, err := exec.Command("/bin/sh", "-c", pvCmd).
			CombinedOutput(); len(res) == 0 {
		} else if err != nil {
			glog.Infof("waiting for tidbcluster: %s/%s pv deleting, %v, %s",
				info.Namespace, info.ClusterName, err, string(res))
			return false, nil
		}
		return true, nil
	}
	return wait.PollImmediate(DefaultPollInterval, DefaultPollTimeout, pollFn)
}

func (oa *operatorActions) CheckTidbClusterStatus(info *TidbClusterInfo) error {
	glog.Infof("begin to check tidb cluster cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)
	defer func() {
		glog.Infof("check tidb cluster end cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)
	}()
	ns := info.Namespace
	tcName := info.ClusterName
	if err := wait.PollImmediate(DefaultPollInterval, DefaultPollTimeout, func() (bool, error) {
		var tc *v1alpha1.TidbCluster
		var err error
		if tc, err = oa.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{}); err != nil {
			glog.Errorf("failed to get tidbcluster: %s/%s, %v", ns, tcName, err)
			return false, nil
		}

		if b, err := oa.pdMembersReadyFn(tc); !b && err == nil {
			return false, nil
		}
		if b, err := oa.tikvMembersReadyFn(tc); !b && err == nil {
			return false, nil
		}

		glog.Infof("check tidb cluster begin tidbMembersReadyFn")
		if b, err := oa.tidbMembersReadyFn(tc); !b && err == nil {
			return false, nil
		}

		glog.Infof("check tidb cluster begin reclaimPolicySyncFn")
		if b, err := oa.reclaimPolicySyncFn(tc); !b && err == nil {
			return false, nil
		}

		glog.Infof("check tidb cluster begin metaSyncFn")
		if b, err := oa.metaSyncFn(tc); err != nil {
			return false, err
		} else if !b && err == nil {
			return false, nil
		}

		glog.Infof("check tidb cluster begin schedulerHAFn")
		if b, err := oa.schedulerHAFn(tc); !b && err == nil {
			return false, nil
		}

		glog.Infof("check tidb cluster begin passwordIsSet")
		if b, err := oa.passwordIsSet(info); !b && err == nil {
			return false, nil
		}

		if info.Monitor {
			glog.Infof("check tidb monitor normal")
			if b, err := oa.monitorNormal(info); !b && err == nil {
				return false, nil
			}
		}
		return true, nil
	}); err != nil {
		glog.Infof("check tidb cluster status failed: %s", err.Error())
		return fmt.Errorf("failed to waiting for tidbcluster %s/%s ready in 10 minutes", ns, tcName)
	}

	return nil
}

func (oa *operatorActions) BeginInsertDataTo(info *TidbClusterInfo) error {
	dsn := getDSN(info.Namespace, info.ClusterName, "test", info.Password)
	db, err := util.OpenDB(dsn, defaultConcurrency)
	if err != nil {
		return err
	}

	return info.blockWriter.Start(db)
}

func (oa *operatorActions) StopInsertDataTo(info *TidbClusterInfo) error {
	info.blockWriter.Stop()
	return nil
}

func chartPath(name string, tag string) string {
	return "/charts/" + tag + "/" + name
}

func (oa *operatorActions) ScaleTidbCluster(info *TidbClusterInfo) error {
	cmd := fmt.Sprintf("helm upgrade %s %s --set-string %s",
		info.ClusterName, chartPath("tidb-cluster", info.OperatorTag), info.HelmSetString())
	glog.Info("[SCALE] " + cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to scale tidb cluster: %s", string(res))
	}
	return nil
}

func (oa *operatorActions) UpgradeTidbCluster(info *TidbClusterInfo) error {
	cmd := fmt.Sprintf("helm upgrade %s %s --set-string %s",
		info.ClusterName, chartPath("tidb-cluster", info.OperatorTag), info.HelmSetString())
	glog.Info("[UPGRADE] " + cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to upgrade tidb cluster: %s", string(res))
	}
	return nil
}

func (oa *operatorActions) DeployMonitor(info *TidbClusterInfo) error { return nil }
func (oa *operatorActions) CleanMonitor(info *TidbClusterInfo) error  { return nil }

func getComponentContainer(set *v1beta1.StatefulSet) (corev1.Container, bool) {
	name := set.Labels[label.ComponentLabelKey]
	for _, c := range set.Spec.Template.Spec.Containers {
		if c.Name == name {
			return c, true
		}
	}
	return corev1.Container{}, false
}

func (oa *operatorActions) pdMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	pdSetName := controller.PDMemberName(tcName)

	pdSet, err := oa.kubeCli.AppsV1beta1().StatefulSets(ns).Get(pdSetName, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("failed to get statefulset: %s/%s, %v", ns, pdSetName, err)
		return false, nil
	}

	if tc.Status.PD.StatefulSet == nil {
		glog.Infof("tidbcluster: %s/%s .status.PD.StatefulSet is nil", ns, tcName)
		return false, nil
	}
	failureCount := len(tc.Status.PD.FailureMembers)
	replicas := tc.Spec.PD.Replicas + int32(failureCount)
	if *pdSet.Spec.Replicas != replicas {
		glog.Infof("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, pdSetName, *pdSet.Spec.Replicas, replicas)
		return false, nil
	}
	if pdSet.Status.ReadyReplicas != tc.Spec.PD.Replicas {
		glog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, pdSetName, pdSet.Status.ReadyReplicas, tc.Spec.PD.Replicas)
		return false, nil
	}
	if len(tc.Status.PD.Members) != int(tc.Spec.PD.Replicas) {
		glog.Infof("tidbcluster: %s/%s .status.PD.Members count(%d) != %d",
			ns, tcName, len(tc.Status.PD.Members), tc.Spec.PD.Replicas)
		return false, nil
	}
	if pdSet.Status.ReadyReplicas != pdSet.Status.Replicas {
		glog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, pdSetName, pdSet.Status.ReadyReplicas, pdSet.Status.Replicas)
		return false, nil
	}
	if c, ok := getComponentContainer(pdSet); !ok || tc.Spec.PD.Image != c.Image {
		glog.Infof("statefulset: %s/%s .spec.template.spec.containers[name=pd].image(%s) != %s",
			ns, pdSetName, c.Image, tc.Spec.PD.Image)
		return false, nil
	}

	for _, member := range tc.Status.PD.Members {
		if !member.Health {
			glog.Infof("tidbcluster: %s/%s pd member(%s/%s) is not health",
				ns, tcName, member.ID, member.Name)
			return false, nil
		}
	}

	pdServiceName := controller.PDMemberName(tcName)
	pdPeerServiceName := controller.PDPeerMemberName(tcName)
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(pdServiceName, metav1.GetOptions{}); err != nil {
		glog.Errorf("failed to get service: %s/%s", ns, pdServiceName)
		return false, nil
	}
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(pdPeerServiceName, metav1.GetOptions{}); err != nil {
		glog.Errorf("failed to get peer service: %s/%s", ns, pdPeerServiceName)
		return false, nil
	}

	return true, nil
}

func (oa *operatorActions) tikvMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	tikvSetName := controller.TiKVMemberName(tcName)

	tikvSet, err := oa.kubeCli.AppsV1beta1().StatefulSets(ns).Get(tikvSetName, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("failed to get statefulset: %s/%s, %v", ns, tikvSetName, err)
		return false, nil
	}

	if tc.Status.TiKV.StatefulSet == nil {
		glog.Infof("tidbcluster: %s/%s .status.TiKV.StatefulSet is nil", ns, tcName)
		return false, nil
	}
	failureCount := len(tc.Status.TiKV.FailureStores)
	replicas := tc.Spec.TiKV.Replicas + int32(failureCount)
	if *tikvSet.Spec.Replicas != replicas {
		glog.Infof("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, tikvSetName, *tikvSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tikvSet.Status.ReadyReplicas != replicas {
		glog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, tikvSetName, tikvSet.Status.ReadyReplicas, replicas)
		return false, nil
	}
	if len(tc.Status.TiKV.Stores) != int(replicas) {
		glog.Infof("tidbcluster: %s/%s .status.TiKV.Stores.count(%d) != %d",
			ns, tcName, len(tc.Status.TiKV.Stores), tc.Spec.TiKV.Replicas)
		return false, nil
	}
	if tikvSet.Status.ReadyReplicas != tikvSet.Status.Replicas {
		glog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, tikvSetName, tikvSet.Status.ReadyReplicas, tikvSet.Status.Replicas)
		return false, nil
	}
	if c, ok := getComponentContainer(tikvSet); !ok || tc.Spec.TiKV.Image != c.Image {
		glog.Infof("statefulset: %s/%s .spec.template.spec.containers[name=tikv].image(%s) != %s",
			ns, tikvSetName, c.Image, tc.Spec.TiKV.Image)
		return false, nil
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.State != v1alpha1.TiKVStateUp {
			glog.Infof("tidbcluster: %s/%s's store(%s) state != %s", ns, tcName, store.ID, v1alpha1.TiKVStateUp)
			return false, nil
		}
	}

	tikvPeerServiceName := controller.TiKVPeerMemberName(tcName)
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(tikvPeerServiceName, metav1.GetOptions{}); err != nil {
		glog.Errorf("failed to get peer service: %s/%s", ns, tikvPeerServiceName)
		return false, nil
	}

	return true, nil
}

func (oa *operatorActions) tidbMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	tidbSetName := controller.TiDBMemberName(tcName)

	tidbSet, err := oa.kubeCli.AppsV1beta1().StatefulSets(ns).Get(tidbSetName, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("failed to get statefulset: %s/%s, %v", ns, tidbSetName, err)
		return false, nil
	}

	if tc.Status.TiDB.StatefulSet == nil {
		glog.Infof("tidbcluster: %s/%s .status.TiDB.StatefulSet is nil", ns, tcName)
		return false, nil
	}
	failureCount := len(tc.Status.TiDB.FailureMembers)
	replicas := tc.Spec.TiDB.Replicas + int32(failureCount)
	if *tidbSet.Spec.Replicas != replicas {
		glog.Infof("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, tidbSetName, *tidbSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tidbSet.Status.ReadyReplicas != tc.Spec.TiDB.Replicas {
		glog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, tidbSetName, tidbSet.Status.ReadyReplicas, replicas)
		return false, nil
	}
	if tidbSet.Status.ReadyReplicas != tidbSet.Status.Replicas {
		glog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, tidbSetName, tidbSet.Status.ReadyReplicas, tidbSet.Status.Replicas)
		return false, nil
	}
	if c, ok := getComponentContainer(tidbSet); !ok || tc.Spec.TiDB.Image != c.Image {
		glog.Infof("statefulset: %s/%s .spec.template.spec.containers[name=tikv].image(%s) != %s",
			ns, tidbSetName, c.Image, tc.Spec.TiDB.Image)
		return false, nil
	}

	_, err = oa.kubeCli.CoreV1().Services(ns).Get(tidbSetName, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("failed to get service: %s/%s", ns, tidbSetName)
		return false, nil
	}

	return true, nil
}

func (oa *operatorActions) reclaimPolicySyncFn(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Labels(),
		).String(),
	}
	var pvcList *corev1.PersistentVolumeClaimList
	var err error
	if pvcList, err = oa.kubeCli.CoreV1().PersistentVolumeClaims(ns).List(listOptions); err != nil {
		glog.Errorf("failed to list pvs for tidbcluster %s/%s, %v", ns, tcName, err)
		return false, nil
	}

	for _, pvc := range pvcList.Items {
		pvName := pvc.Spec.VolumeName
		if pv, err := oa.kubeCli.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{}); err != nil {
			glog.Errorf("failed to get pv: %s, error: %v", pvName, err)
			return false, nil
		} else if pv.Spec.PersistentVolumeReclaimPolicy != tc.Spec.PVReclaimPolicy {
			glog.Errorf("pv: %s's reclaimPolicy is not Retain", pvName)
			return false, nil
		}
	}

	return true, nil
}

func (oa *operatorActions) metaSyncFn(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	pdCli := oa.pdControl.GetPDClient(tc)
	var cluster *metapb.Cluster
	var err error
	if cluster, err = pdCli.GetCluster(); err != nil {
		glog.Errorf("failed to get cluster from pdControl: %s/%s, error: %v", ns, tcName, err)
		return false, nil
	}

	clusterID := strconv.FormatUint(cluster.Id, 10)
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Labels(),
		).String(),
	}

	var podList *corev1.PodList
	if podList, err = oa.kubeCli.CoreV1().Pods(ns).List(listOptions); err != nil {
		glog.Errorf("failed to list pods for tidbcluster %s/%s, %v", ns, tcName, err)
		return false, nil
	}

outerLoop:
	for _, pod := range podList.Items {
		podName := pod.GetName()
		if pod.Labels[label.ClusterIDLabelKey] != clusterID {
			glog.Infof("tidbcluster %s/%s's pod %s's label %s not equals %s ",
				ns, tcName, podName, label.ClusterIDLabelKey, clusterID)
			return false, nil
		}

		component := pod.Labels[label.ComponentLabelKey]
		switch component {
		case label.PDLabelVal:
			var memberID string
			members, err := pdCli.GetMembers()
			if err != nil {
				glog.Errorf("failed to get members for tidbcluster %s/%s, %v", ns, tcName, err)
				return false, nil
			}
			for _, member := range members.Members {
				if member.Name == podName {
					memberID = strconv.FormatUint(member.GetMemberId(), 10)
					break
				}
			}
			if memberID == "" {
				glog.Errorf("tidbcluster: %s/%s's pod %s label [%s] is empty",
					ns, tcName, podName, label.MemberIDLabelKey)
				return false, nil
			}
			if pod.Labels[label.MemberIDLabelKey] != memberID {
				return false, fmt.Errorf("tidbcluster: %s/%s's pod %s label [%s] not equals %s",
					ns, tcName, podName, label.MemberIDLabelKey, memberID)
			}
		case label.TiKVLabelVal:
			var storeID string
			stores, err := pdCli.GetStores()
			if err != nil {
				glog.Errorf("failed to get stores for tidbcluster %s/%s, %v", ns, tcName, err)
				return false, nil
			}
			for _, store := range stores.Stores {
				addr := store.Store.GetAddress()
				if strings.Split(addr, ".")[0] == podName {
					storeID = strconv.FormatUint(store.Store.GetId(), 10)
					break
				}
			}
			if storeID == "" {
				glog.Errorf("tidbcluster: %s/%s's pod %s label [%s] is empty",
					tc.GetNamespace(), tc.GetName(), podName, label.StoreIDLabelKey)
				return false, nil
			}
			if pod.Labels[label.StoreIDLabelKey] != storeID {
				return false, fmt.Errorf("tidbcluster: %s/%s's pod %s label [%s] not equals %s",
					ns, tcName, podName, label.StoreIDLabelKey, storeID)
			}
		case label.TiDBLabelVal:
			continue outerLoop
		default:
			continue outerLoop
		}

		var pvcName string
		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				pvcName = vol.PersistentVolumeClaim.ClaimName
				break
			}
		}
		if pvcName == "" {
			return false, fmt.Errorf("pod: %s/%s's pvcName is empty", ns, podName)
		}

		var pvc *corev1.PersistentVolumeClaim
		if pvc, err = oa.kubeCli.CoreV1().PersistentVolumeClaims(ns).Get(pvcName, metav1.GetOptions{}); err != nil {
			glog.Errorf("failed to get pvc %s/%s for pod %s/%s", ns, pvcName, ns, podName)
			return false, nil
		}
		if pvc.Labels[label.ClusterIDLabelKey] != clusterID {
			return false, fmt.Errorf("tidbcluster: %s/%s's pvc %s label [%s] not equals %s ",
				ns, tcName, pvcName, label.ClusterIDLabelKey, clusterID)
		}
		if pvc.Labels[label.MemberIDLabelKey] != pod.Labels[label.MemberIDLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pvc %s label [%s=%s] not equals pod lablel [%s=%s]",
				ns, tcName, pvcName,
				label.MemberIDLabelKey, pvc.Labels[label.MemberIDLabelKey],
				label.MemberIDLabelKey, pod.Labels[label.MemberIDLabelKey])
		}
		if pvc.Labels[label.StoreIDLabelKey] != pod.Labels[label.StoreIDLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pvc %s label[%s=%s] not equals pod lable[%s=%s]",
				ns, tcName, pvcName,
				label.StoreIDLabelKey, pvc.Labels[label.StoreIDLabelKey],
				label.StoreIDLabelKey, pod.Labels[label.StoreIDLabelKey])
		}
		if pvc.Annotations[label.AnnPodNameKey] != podName {
			return false, fmt.Errorf("tidbcluster: %s/%s's pvc %s annotations [%s] not equals podName: %s",
				ns, tcName, pvcName, label.AnnPodNameKey, podName)
		}

		pvName := pvc.Spec.VolumeName
		var pv *corev1.PersistentVolume
		if pv, err = oa.kubeCli.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{}); err != nil {
			glog.Errorf("failed to get pv for pvc %s/%s, %v", ns, pvcName, err)
			return false, nil
		}
		if pv.Labels[label.NamespaceLabelKey] != ns {
			return false, fmt.Errorf("tidbcluster: %s/%s 's pv %s label [%s] not equals %s",
				ns, tcName, pvName, label.NamespaceLabelKey, ns)
		}
		if pv.Labels[label.ComponentLabelKey] != pod.Labels[label.ComponentLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s=%s] not equals pod label[%s=%s]",
				ns, tcName, pvName,
				label.ComponentLabelKey, pv.Labels[label.ComponentLabelKey],
				label.ComponentLabelKey, pod.Labels[label.ComponentLabelKey])
		}
		if pv.Labels[label.NameLabelKey] != pod.Labels[label.NameLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s=%s] not equals pod label [%s=%s]",
				ns, tcName, pvName,
				label.NameLabelKey, pv.Labels[label.NameLabelKey],
				label.NameLabelKey, pod.Labels[label.NameLabelKey])
		}
		if pv.Labels[label.ManagedByLabelKey] != pod.Labels[label.ManagedByLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s=%s] not equals pod label [%s=%s]",
				ns, tcName, pvName,
				label.ManagedByLabelKey, pv.Labels[label.ManagedByLabelKey],
				label.ManagedByLabelKey, pod.Labels[label.ManagedByLabelKey])
		}
		if pv.Labels[label.InstanceLabelKey] != pod.Labels[label.InstanceLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s=%s] not equals pod label [%s=%s]",
				ns, tcName, pvName,
				label.InstanceLabelKey, pv.Labels[label.InstanceLabelKey],
				label.InstanceLabelKey, pod.Labels[label.InstanceLabelKey])
		}
		if pv.Labels[label.ClusterIDLabelKey] != clusterID {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s] not equals %s",
				ns, tcName, pvName, label.ClusterIDLabelKey, clusterID)
		}
		if pv.Labels[label.MemberIDLabelKey] != pod.Labels[label.MemberIDLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s=%s] not equals pod label [%s=%s]",
				ns, tcName, pvName,
				label.MemberIDLabelKey, pv.Labels[label.MemberIDLabelKey],
				label.MemberIDLabelKey, pod.Labels[label.MemberIDLabelKey])
		}
		if pv.Labels[label.StoreIDLabelKey] != pod.Labels[label.StoreIDLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s=%s] not equals pod label [%s=%s]",
				ns, tcName, pvName,
				label.StoreIDLabelKey, pv.Labels[label.StoreIDLabelKey],
				label.StoreIDLabelKey, pod.Labels[label.StoreIDLabelKey])
		}
		if pv.Annotations[label.AnnPodNameKey] != podName {
			return false, fmt.Errorf("tidbcluster:[%s/%s's pv %s annotations [%s] not equals %s",
				ns, tcName, pvName, label.AnnPodNameKey, podName)
		}
	}

	return true, nil
}

func (oa *operatorActions) schedulerHAFn(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	fn := func(component string) (bool, error) {
		nodeMap := make(map[string][]string)
		listOptions := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(
				label.New().Instance(tcName).Component(component).Labels()).String(),
		}
		var podList *corev1.PodList
		var err error
		if podList, err = oa.kubeCli.CoreV1().Pods(ns).List(listOptions); err != nil {
			glog.Errorf("failed to list pods for tidbcluster %s/%s, %v", ns, tcName, err)
			return false, nil
		}

		totalCount := len(podList.Items)
		for _, pod := range podList.Items {
			nodeName := pod.Spec.NodeName
			if len(nodeMap[nodeName]) == 0 {
				nodeMap[nodeName] = make([]string, 0)
			}
			nodeMap[nodeName] = append(nodeMap[nodeName], pod.GetName())
			if len(nodeMap[nodeName]) > totalCount/2 {
				return false, fmt.Errorf("node %s have %d pods, greater than %d/2",
					nodeName, len(nodeMap[nodeName]), totalCount)
			}
		}
		return true, nil
	}

	components := []string{label.PDLabelVal, label.TiKVLabelVal}
	for _, com := range components {
		if b, err := fn(com); err != nil {
			return false, err
		} else if !b && err == nil {
			return false, nil
		}
	}

	return true, nil
}

func (oa *operatorActions) passwordIsSet(clusterInfo *TidbClusterInfo) (bool, error) {
	ns := clusterInfo.Namespace
	tcName := clusterInfo.ClusterName
	jobName := tcName + "-tidb-initializer"

	var job *batchv1.Job
	var err error
	if job, err = oa.kubeCli.BatchV1().Jobs(ns).Get(jobName, metav1.GetOptions{}); err != nil {
		glog.Errorf("failed to get job %s/%s, %v", ns, jobName, err)
		return false, nil
	}
	if job.Status.Succeeded < 1 {
		glog.Errorf("tidbcluster: %s/%s password setter job not finished", ns, tcName)
		return false, nil
	}

	var db *sql.DB
	dsn := getDSN(ns, tcName, "test", clusterInfo.Password)
	if db, err = sql.Open("mysql", dsn); err != nil {
		glog.Errorf("can't open connection to mysql: %s, %v", dsn, err)
		return false, nil
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		glog.Errorf("can't connect to mysql: %s with password %s, %v", dsn, clusterInfo.Password, err)
		return false, nil
	}

	return true, nil
}

func (oa *operatorActions) monitorNormal(clusterInfo *TidbClusterInfo) (bool, error) {
	ns := clusterInfo.Namespace
	tcName := clusterInfo.ClusterName
	monitorDeploymentName := fmt.Sprintf("%s-monitor", tcName)
	monitorDeployment, err := oa.kubeCli.AppsV1().Deployments(ns).Get(monitorDeploymentName, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("get monitor deployment: [%s/%s] failed", ns, monitorDeploymentName)
		return false, nil
	}
	if monitorDeployment.Status.ReadyReplicas < 1 {
		glog.Info("monitor ready replicas %d < 1", monitorDeployment.Status.ReadyReplicas)
		return false, nil
	}
	configuratorJobName := fmt.Sprintf("%s-monitor-configurator", tcName)
	monitorJob, err := oa.kubeCli.BatchV1().Jobs(ns).Get(configuratorJobName, metav1.GetOptions{})
	if err != nil {
		glog.Info("get monitor configurator job: [%s/%s] failed", ns, configuratorJobName)
		return false, nil
	}
	if monitorJob.Status.Succeeded == 0 {
		glog.Info("the monitor configurator job: [%s/%s] had not success", ns, configuratorJobName)
		return false, nil
	}

	if err := oa.checkPrometheus(clusterInfo); err != nil {
		glog.Info("check [%s/%s]'s prometheus data failed: %v", ns, monitorDeploymentName, err)
		return false, nil
	}

	if err := oa.checkGrafanaData(clusterInfo); err != nil {
		glog.Info("check [%s/%s]'s grafana data failed: %v", ns, monitorDeploymentName, err)
		return false, nil
	}
	return true, nil
}

func (oa *operatorActions) checkPrometheus(clusterInfo *TidbClusterInfo) error {
	ns := clusterInfo.Namespace
	tcName := clusterInfo.ClusterName
	prometheusSvc := fmt.Sprintf("http://%s-prometheus.%s:9090/api/v1/query?query=up", tcName, ns)
	resp, err := http.Get(prometheusSvc)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	response := &struct {
		Status string `json:"status"`
	}{}
	err = json.Unmarshal(body, response)
	if err != nil {
		return err
	}
	if response.Status != "success" {
		return fmt.Errorf("the prometheus's api[%s] has not ready", prometheusSvc)
	}
	return nil
}

func (oa *operatorActions) checkGrafanaData(clusterInfo *TidbClusterInfo) error {
	ns := clusterInfo.Namespace
	tcName := clusterInfo.ClusterName
	svcName := fmt.Sprintf("%s-grafana", tcName)
	end := time.Now()
	start := end.Add(-time.Minute)
	values := url.Values{}
	values.Set("query", `sum(tikv_pd_heartbeat_tick_total{type="leader"}) by (job)`)
	values.Set("start", fmt.Sprintf("%d", start.Unix()))
	values.Set("end", fmt.Sprintf("%d", end.Unix()))
	values.Set("step", "30")
	u := fmt.Sprintf("http://%s.%s.svc.cluster.local:3000/api/datasources/proxy/1/api/v1/query_range?%s", svcName, ns, values.Encode())
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(grafanaUsername, grafanaPassword)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	data := struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric struct {
					Job string `json:"job"`
				} `json:"metric"`
				Values []interface{} `json:"values"`
			} `json:"result"`
		}
	}{}
	if err := json.Unmarshal(buf, &data); err != nil {
		return err
	}
	if data.Status != "success" || len(data.Data.Result) < 1 {
		return fmt.Errorf("invalid response: status: %s, result: %v", data.Status, data.Data.Result)
	}
	return nil
}

func getDSN(ns, tcName, databaseName, password string) string {
	return fmt.Sprintf("root:%s@(%s-tidb.%s:4000)/%s?charset=utf8", password, tcName, ns, databaseName)
}

func releaseIsNotFound(err error) bool {
	return strings.Contains(err.Error(), "not found")
}

func cloneOperatorRepo() error {
	cmd := fmt.Sprintf("git clone https://github.com/pingcap/tidb-operator.git /tidb-operator")
	glog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone tidb-operator repository: %v, %s", err, string(res))
	}

	return nil
}

func checkoutTag(tagName string) error {
	cmd := fmt.Sprintf(`cd /tidb-operator &&
		git stash -u &&
		git checkout %s &&
		mkdir -p /charts/%s &&
		cp -rf charts/tidb-operator /charts/%s/tidb-operator &&
		cp -rf charts/tidb-cluster /charts/%s/tidb-cluster &&
		cp -rf charts/tidb-backup /charts/%s/tidb-backup`,
		tagName, tagName, tagName, tagName, tagName)
	glog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check tag: %s, %v, %s", tagName, err, string(res))
	}

	return nil
}

func (oa *operatorActions) DeployAdHocBackup(info *TidbClusterInfo) error {
	glog.Infof("begin to deploy adhoc backup cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)
	defer func() {
		glog.Infof("deploy adhoc backup end cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)
	}()
	sets := map[string]string{
		"clusterName":  info.ClusterName,
		"name":         "test-backup",
		"mode":         "backup",
		"user":         "root",
		"password":     info.Password,
		"storage.size": "10Gi",
	}
	var buffer bytes.Buffer
	for k, v := range sets {
		set := fmt.Sprintf(" --set %s=%s", k, v)
		_, err := buffer.WriteString(set)
		if err != nil {
			return err
		}
	}

	setStr := buffer.String()
	fullbackupName := fmt.Sprintf("%s-backup", info.ClusterName)
	cmd := fmt.Sprintf("helm install -n %s --namespace %s /charts/%s/tidb-backup %s",
		fullbackupName, info.Namespace, info.OperatorTag, setStr)
	glog.Infof("install adhoc deployment [%s]", cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to launch adhoc backup job: %v, %s", err, string(res))
	}
	return nil
}

func (oa *operatorActions) CheckAdHocBackup(info *TidbClusterInfo) error {
	glog.Infof("begin to clean adhoc backup cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)
	defer func() {
		glog.Infof("deploy clean backup end cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)
	}()

	jobName := fmt.Sprintf("%s-%s", info.ClusterName, "test-backup")
	fn := func() (bool, error) {
		job, err := oa.kubeCli.BatchV1().Jobs(info.Namespace).Get(jobName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("failed to get jobs %s ,%v", jobName, err)
			return false, nil
		}
		if job.Status.Succeeded == 0 {
			glog.Errorf("cluster [%s] back up job is not completed, please wait! ", info.ClusterName)
			return false, nil
		}

		return true, nil
	}

	err := wait.Poll(DefaultPollInterval, DefaultPollTimeout, fn)
	if err != nil {
		return fmt.Errorf("failed to launch scheduler backup job: %v", err)
	}
	return nil
}

func (oa *operatorActions) Restore(from *TidbClusterInfo, to *TidbClusterInfo) error {
	glog.Infof("begin to deploy restore cluster[%s] namespace[%s]", from.ClusterName, from.Namespace)
	defer func() {
		glog.Infof("deploy restore end cluster[%s] namespace[%s]", to.ClusterName, to.Namespace)
	}()
	sets := map[string]string{
		"clusterName":  to.ClusterName,
		"name":         "test-backup",
		"mode":         "restore",
		"user":         "root",
		"password":     to.Password,
		"storage.size": "10Gi",
	}
	var buffer bytes.Buffer
	for k, v := range sets {
		set := fmt.Sprintf(" --set %s=%s", k, v)
		_, err := buffer.WriteString(set)
		if err != nil {
			return err
		}
	}

	setStr := buffer.String()
	restoreName := fmt.Sprintf("%s-restore", from.ClusterName)
	cmd := fmt.Sprintf("helm install -n %s --namespace %s /charts/%s/tidb-backup %s",
		restoreName, to.Namespace, to.OperatorTag, setStr)
	glog.Infof("install restore [%s]", cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to launch restore job: %v, %s", err, string(res))
	}

	return nil
}

func (oa *operatorActions) CheckRestore(from *TidbClusterInfo, to *TidbClusterInfo) error {
	glog.Infof("begin to check restore backup cluster[%s] namespace[%s]", from.ClusterName, from.Namespace)
	defer func() {
		glog.Infof("check restore end cluster[%s] namespace[%s]", to.ClusterName, to.Namespace)
	}()

	jobName := fmt.Sprintf("%s-restore-test-backup", to.ClusterName)
	fn := func() (bool, error) {
		job, err := oa.kubeCli.BatchV1().Jobs(to.Namespace).Get(jobName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("failed to get jobs %s ,%v", jobName, err)
			return false, nil
		}
		if job.Status.Succeeded == 0 {
			glog.Errorf("cluster [%s] back up job is not completed, please wait! ", to.ClusterName)
			return false, nil
		}

		fromCount, err := from.QueryCount()
		if err != nil {
			glog.Errorf("cluster [%s] count err ", from.ClusterName)
			return false, nil
		}

		toCount, err := to.QueryCount()
		if err != nil {
			glog.Errorf("cluster [%s] count err ", to.ClusterName)
			return false, nil
		}

		if fromCount != toCount {
			glog.Errorf("cluster [%s] count %d cluster [%s] count %d is not equal ",
				from.ClusterName, fromCount, to.ClusterName, toCount)
			return false, nil
		}
		return true, nil
	}

	err := wait.Poll(DefaultPollInterval, DefaultPollTimeout, fn)
	if err != nil {
		return fmt.Errorf("failed to launch scheduler backup job: %v", err)
	}
	return nil
}

func (oa *operatorActions) ForceDeploy(info *TidbClusterInfo) error {
	if err := oa.CleanTidbCluster(info); err != nil {
		return err
	}

	if err := oa.DeployTidbCluster(info); err != nil {
		return err
	}

	return nil
}

func (info *TidbClusterInfo) QueryCount() (int, error) {
	tableName := "test"
	db, err := sql.Open("mysql", getDSN(info.Namespace, info.ClusterName, "record", info.Password))
	if err != nil {
		return 0, err
	}
	defer db.Close()

	rows, err := db.Query(fmt.Sprintf("SELECT count(*) FROM %s", tableName))
	if err != nil {
		glog.Infof("cluster:[%s], error: %v", info.ClusterName, err)
		return 0, err
	}

	for rows.Next() {
		var count int
		err := rows.Scan(&count)
		if err != nil {
			glog.Infof("cluster:[%s], error :%v", info.ClusterName, err)
		}
		return count, nil
	}
	return 0, fmt.Errorf("can not find count of ")
}

func (oa *operatorActions) CreateSecret(info *TidbClusterInfo) error {
	initSecretName := "set-secret"
	backupSecretName := "backup-secret"
	initSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      initSecretName,
			Namespace: info.Namespace,
		},
		Data: map[string][]byte{
			"root": []byte(info.Password),
		},
		Type: corev1.SecretTypeOpaque,
	}

	_, err := oa.kubeCli.CoreV1().Secrets(info.Namespace).Create(&initSecret)
	if err != nil && !releaseIsExist(err) {
		return err
	}

	backupSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupSecretName,
			Namespace: info.Namespace,
		},
		Data: map[string][]byte{
			"user":     []byte("root"),
			"password": []byte(info.Password),
		},
		Type: corev1.SecretTypeOpaque,
	}

	_, err = oa.kubeCli.CoreV1().Secrets(info.Namespace).Create(&backupSecret)
	if err != nil && !releaseIsExist(err) {
		return err
	}

	return nil
}

func releaseIsExist(err error) bool {
	return strings.Contains(err.Error(), "already exists")
}

func (oa *operatorActions) DeployScheduledBackup(info *TidbClusterInfo) error {
	return nil
}

func (oa *operatorActions) CheckScheduledBackup(info *TidbClusterInfo) error {
	return nil
}

func (oa *operatorActions) DeployIncrementalBackup(from *TidbClusterInfo, to *TidbClusterInfo) error {
	return nil
}

func (oa *operatorActions) CheckIncrementalBackup(info *TidbClusterInfo) error {
	return nil
}
