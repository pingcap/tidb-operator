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
	pingcapErrors "github.com/pingcap/errors"
	"github.com/pingcap/kvproto/pkg/metapb"
	"k8s.io/api/apps/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/tests/pkg/blockwriter"
	"github.com/pingcap/tidb-operator/tests/pkg/util"
)

const (
	defaultTableNum    int = 64
	defaultConcurrency     = 512
	defaultBatchSize       = 100
	defaultRawSize         = 100

	period = 5 * time.Minute
)

func NewOperatorActions(cli versioned.Interface, kubeCli kubernetes.Interface, cfg *Config) OperatorActions {
	return &operatorActions{
		cli:       cli,
		kubeCli:   kubeCli,
		pdControl: controller.NewDefaultPDControl(),
		cfg:       cfg,
	}
}

const (
	DefaultPollTimeout  time.Duration = 10 * time.Minute
	DefaultPollInterval time.Duration = 1 * time.Minute
	getBackupDirPodName               = "get-backup-dir"
	grafanaUsername                   = "admin"
	grafanaPassword                   = "admin"
)

type OperatorActions interface {
	DeployOperator(info *OperatorConfig) error
	DeployOperatorOrDie(info *OperatorConfig)
	CleanOperator(info *OperatorConfig) error
	CleanOperatorOrDie(info *OperatorConfig)
	UpgradeOperator(info *OperatorConfig) error
	DumpAllLogs(info *OperatorConfig, clusterInfos []*TidbClusterConfig) error
	DeployTidbCluster(info *TidbClusterConfig) error
	DeployTidbClusterOrDie(info *TidbClusterConfig)
	CleanTidbCluster(info *TidbClusterConfig) error
	CleanTidbClusterOrDie(info *TidbClusterConfig)
	CheckTidbClusterStatus(info *TidbClusterConfig) error
	CheckTidbClusterStatusOrDie(info *TidbClusterConfig)
	BeginInsertDataTo(info *TidbClusterConfig) error
	StopInsertDataTo(info *TidbClusterConfig) error
	ScaleTidbCluster(info *TidbClusterConfig) error
	ScaleTidbClusterOrDie(info *TidbClusterConfig)
	CheckScaleInSafely(info *TidbClusterConfig) error
	CheckScaledCorrectly(info *TidbClusterConfig, podUIDsBeforeScale map[string]types.UID) error
	UpgradeTidbCluster(info *TidbClusterConfig) error
	UpgradeTidbClusterOrDie(info *TidbClusterConfig)
	CheckUpgradeProgress(info *TidbClusterConfig) error
	DeployAdHocBackup(info *TidbClusterConfig) error
	CheckAdHocBackup(info *TidbClusterConfig) error
	DeployScheduledBackup(info *TidbClusterConfig) error
	CheckScheduledBackup(info *TidbClusterConfig) error
	DeployIncrementalBackup(from *TidbClusterConfig, to *TidbClusterConfig) error
	CheckIncrementalBackup(info *TidbClusterConfig) error
	Restore(from *TidbClusterConfig, to *TidbClusterConfig) error
	CheckRestore(from *TidbClusterConfig, to *TidbClusterConfig) error
	ForceDeploy(info *TidbClusterConfig) error
	CreateSecret(info *TidbClusterConfig) error
	GetPodUIDMap(info *TidbClusterConfig) (map[string]types.UID, error)
	GetNodeMap(info *TidbClusterConfig, component string) (map[string][]string, error)
	TruncateSSTFileThenCheckFailover(info *TidbClusterConfig, tikvFailoverPeriod time.Duration) error
	TruncateSSTFileThenCheckFailoverOrDie(info *TidbClusterConfig, tikvFailoverPeriod time.Duration)
	CheckFailoverPending(info *TidbClusterConfig, faultPoint *time.Time) (bool, error)
	CheckFailoverPendingOrDie(clusters []*TidbClusterConfig, faultPoint *time.Time)
	CheckFailover(info *TidbClusterConfig, faultNode string) (bool, error)
	CheckFailoverOrDie(clusters []*TidbClusterConfig, faultNode string)
	CheckRecover(cluster *TidbClusterConfig) (bool, error)
	CheckRecoverOrDie(clusters []*TidbClusterConfig)
}

type operatorActions struct {
	cli       versioned.Interface
	kubeCli   kubernetes.Interface
	pdControl controller.PDControlInterface
	cfg       *Config
}

var _ = OperatorActions(&operatorActions{})

type OperatorConfig struct {
	Namespace      string
	ReleaseName    string
	Image          string
	Tag            string
	SchedulerImage string
	SchedulerTag   string
	LogLevel       string
}

type TidbClusterConfig struct {
	BackupPVC        string
	Namespace        string
	ClusterName      string
	OperatorTag      string
	PDImage          string
	TiKVImage        string
	TiDBImage        string
	StorageClassName string
	Password         string
	InitSql          string
	RecordCount      string
	InsertBatchSize  string
	Resources        map[string]string
	Args             map[string]string
	blockWriter      *blockwriter.BlockWriterCase
	Monitor          bool
	UserName         string
	InitSecretName   string
	BackupSecretName string
}

func (tc *TidbClusterConfig) BackupHelmSetString(m map[string]string) string {

	set := map[string]string{
		"clusterName": tc.ClusterName,
		"secretName":  tc.BackupSecretName,
	}

	for k, v := range tc.Args {
		set[k] = v
	}
	for k, v := range m {
		set[k] = v
	}

	arr := make([]string, 0, len(set))
	for k, v := range set {
		arr = append(arr, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(arr, ",")
}

func (tc *TidbClusterConfig) TidbClusterHelmSetString(m map[string]string) string {

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
		"tidb.passwordSecretName": tc.InitSecretName,
		"tidb.initSql":            tc.InitSql,
		"monitor.create":          strconv.FormatBool(tc.Monitor),
	}

	for k, v := range tc.Resources {
		set[k] = v
	}
	for k, v := range tc.Args {
		set[k] = v
	}
	for k, v := range m {
		set[k] = v
	}

	arr := make([]string, 0, len(set))
	for k, v := range set {
		arr = append(arr, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(arr, ",")
}

func (oi *OperatorConfig) OperatorHelmSetString(m map[string]string) string {
	set := map[string]string{
		"operatorImage":                    oi.Image,
		"controllerManager.autoFailover":   "true",
		"scheduler.kubeSchedulerImageName": oi.SchedulerImage,
		"controllerManager.logLevel":       oi.LogLevel,
		"scheduler.logLevel":               "2",
	}
	if oi.SchedulerTag != "" {
		set["scheduler.kubeSchedulerImageTag"] = oi.SchedulerTag
	}

	arr := make([]string, 0, len(set))
	for k, v := range set {
		arr = append(arr, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(arr, ",")
}

func (oa *operatorActions) DeployOperator(info *OperatorConfig) error {
	if info.Tag != "e2e" {
		if err := oa.cloneOperatorRepo(); err != nil {
			return err
		}
		if err := oa.checkoutTag(info.Tag); err != nil {
			return err
		}
	}

	cmd := fmt.Sprintf(`helm install /charts/%s/tidb-operator \
		--name %s \
		--namespace %s \
		--set-string %s`,
		info.Tag,
		info.ReleaseName,
		info.Namespace,
		info.OperatorHelmSetString(nil))
	glog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to deploy operator: %v, %s", err, string(res))
	}

	return nil
}

func (oa *operatorActions) DeployOperatorOrDie(info *OperatorConfig) {
	if err := oa.DeployOperator(info); err != nil {
		panic(err)
	}
}

func (oa *operatorActions) CleanOperator(info *OperatorConfig) error {
	res, err := exec.Command("helm", "del", "--purge", info.ReleaseName).CombinedOutput()
	if err == nil || !releaseIsNotFound(err) {
		return nil
	}
	return fmt.Errorf("failed to clear operator: %v, %s", err, string(res))
}

func (oa *operatorActions) CleanOperatorOrDie(info *OperatorConfig) {
	if err := oa.CleanOperator(info); err != nil {
		panic(err)
	}
}

func (oa *operatorActions) UpgradeOperator(info *OperatorConfig) error {
	if err := oa.checkoutTag(info.Tag); err != nil {
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

func (oa *operatorActions) DeployTidbCluster(info *TidbClusterConfig) error {
	glog.Infof("begin to deploy tidb cluster cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: info.Namespace,
		},
	}
	_, err := oa.kubeCli.CoreV1().Namespaces().Create(namespace)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace[%s]:%v", info.Namespace, err)
	}

	err = oa.CreateSecret(info)
	if err != nil {
		return fmt.Errorf("failed to create secret of cluster [%s]: %v", info.ClusterName, err)
	}

	cmd := fmt.Sprintf("helm install /charts/%s/tidb-cluster  --name %s --namespace %s --set-string %s",
		info.OperatorTag, info.ClusterName, info.Namespace, info.TidbClusterHelmSetString(nil))
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

func (oa *operatorActions) DeployTidbClusterOrDie(info *TidbClusterConfig) {
	if err := oa.DeployTidbCluster(info); err != nil {
		panic(err)
	}
}

func (oa *operatorActions) CleanTidbCluster(info *TidbClusterConfig) error {
	glog.Infof("cleaning tidbcluster %s/%s", info.Namespace, info.ClusterName)

	charts := []string{
		info.ClusterName,
		fmt.Sprintf("%s-backup", info.ClusterName),
		fmt.Sprintf("%s-restore", info.ClusterName),
		fmt.Sprintf("%s-scheduler-backup", info.ClusterName),
	}
	for _, chartName := range charts {
		res, err := exec.Command("helm", "del", "--purge", chartName).CombinedOutput()
		if err != nil && releaseIsNotFound(err) {
			return fmt.Errorf("failed to delete chart: %s/%s, %v, %s",
				info.Namespace, chartName, err, string(res))
		}
	}

	err := oa.kubeCli.CoreV1().Pods(info.Namespace).Delete(getBackupDirPodName, &metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete dir pod %v", err)
	}

	setStr := label.New().Instance(info.ClusterName).String()

	resources := []string{"pvc"}
	for _, resource := range resources {
		if res, err := exec.Command("kubectl", "delete", resource, "-n", info.Namespace, "-l",
			setStr).CombinedOutput(); err != nil {
			return fmt.Errorf("failed to delete %s: %v, %s", resource, err, string(res))
		}
	}

	// delete all jobs
	allJobsSet := label.Label{}.Instance(info.ClusterName).String()
	if res, err := exec.Command("kubectl", "delete", "jobs", "-n", info.Namespace, "-l", allJobsSet).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete jobs: %v, %s", err, string(res))
	}

	patchPVCmd := fmt.Sprintf(`kubectl get pv -l %s=%s,%s=%s --output=name | xargs -I {} \
		kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'`,
		label.NamespaceLabelKey, info.Namespace, label.InstanceLabelKey, info.ClusterName)
	glog.V(4).Info(patchPVCmd)
	if res, err := exec.Command("/bin/sh", "-c", patchPVCmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to patch pv: %v, %s", err, string(res))
	}

	pollFn := func() (bool, error) {
		if res, err := exec.Command("kubectl", "get", "po", "--output=name", "-n", info.Namespace, "-l", setStr).
			CombinedOutput(); err != nil || len(res) != 0 {
			glog.V(4).Infof("waiting for tidbcluster: %s/%s pods deleting, %v, [%s]",
				info.Namespace, info.ClusterName, err, string(res))
			return false, nil
		}

		pvCmd := fmt.Sprintf("kubectl get pv -l %s=%s,%s=%s 2>/dev/null|grep Released",
			label.NamespaceLabelKey, info.Namespace, label.InstanceLabelKey, info.ClusterName)
		glog.V(4).Info(pvCmd)
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

func (oa *operatorActions) CleanTidbClusterOrDie(info *TidbClusterConfig) {
	if err := oa.CleanTidbCluster(info); err != nil {
		panic(err)
	}
}

func (oa *operatorActions) CheckTidbClusterStatus(info *TidbClusterConfig) error {
	glog.Infof("begin to check tidb cluster cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)

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

func (oa *operatorActions) CheckTidbClusterStatusOrDie(info *TidbClusterConfig) {
	if err := oa.CheckTidbClusterStatus(info); err != nil {
		panic(err)
	}
}

func (oa *operatorActions) BeginInsertDataTo(info *TidbClusterConfig) error {
	dsn := getDSN(info.Namespace, info.ClusterName, "test", info.Password)
	db, err := util.OpenDB(dsn, defaultConcurrency)
	if err != nil {
		return err
	}

	return info.blockWriter.Start(db)
}

func (oa *operatorActions) StopInsertDataTo(info *TidbClusterConfig) error {
	info.blockWriter.Stop()
	return nil
}

func chartPath(name string, tag string) string {
	return "/charts/" + tag + "/" + name
}

func (oa *operatorActions) ScaleTidbCluster(info *TidbClusterConfig) error {
	cmd := fmt.Sprintf("helm upgrade %s %s --set-string %s",
		info.ClusterName, chartPath("tidb-cluster", info.OperatorTag), info.TidbClusterHelmSetString(nil))
	glog.Info("[SCALE] " + cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return pingcapErrors.Wrapf(err, "failed to scale tidb cluster: %s", string(res))
	}
	return nil
}

func (oa *operatorActions) ScaleTidbClusterOrDie(info *TidbClusterConfig) {
	if err := oa.ScaleTidbCluster(info); err != nil {
		panic(err)
	}
}

func (oa *operatorActions) CheckScaleInSafely(info *TidbClusterConfig) error {
	return wait.Poll(DefaultPollInterval, DefaultPollTimeout, func() (done bool, err error) {
		tc, err := oa.cli.PingcapV1alpha1().TidbClusters(info.Namespace).Get(info.ClusterName, metav1.GetOptions{})
		if err != nil {
			glog.Infof("failed to get tidbcluster when scale in tidbcluster, error: %v", err)
			return false, nil
		}

		tikvSetName := controller.TiKVMemberName(info.ClusterName)
		tikvSet, err := oa.kubeCli.AppsV1beta1().StatefulSets(info.Namespace).Get(tikvSetName, metav1.GetOptions{})
		if err != nil {
			glog.Infof("failed to get tikvSet statefulset: [%s], error: %v", tikvSetName, err)
			return false, nil
		}

		pdClient := controller.NewDefaultPDControl().GetPDClient(tc)
		stores, err := pdClient.GetStores()
		if err != nil {
			glog.Infof("pdClient.GetStores failed,error: %v", err)
			return false, nil
		}
		if len(stores.Stores) > int(*tikvSet.Spec.Replicas) {
			glog.Infof("stores.Stores: %v", stores.Stores)
			glog.Infof("tikvSet.Spec.Replicas: %d", *tikvSet.Spec.Replicas)
			return false, fmt.Errorf("the tikvSet.Spec.Replicas may reduce before tikv complete offline")
		}

		if *tikvSet.Spec.Replicas == tc.Spec.TiKV.Replicas {
			return true, nil
		}

		return false, nil
	})
}

func (oa *operatorActions) CheckScaledCorrectly(info *TidbClusterConfig, podUIDsBeforeScale map[string]types.UID) error {
	return wait.Poll(DefaultPollInterval, DefaultPollTimeout, func() (done bool, err error) {
		podUIDs, err := oa.GetPodUIDMap(info)
		if err != nil {
			glog.Infof("failed to get pd pods's uid, error: %v", err)
			return false, nil
		}

		if len(podUIDsBeforeScale) == len(podUIDs) {
			return false, fmt.Errorf("the length of pods before scale equals the length of pods after scale")
		}

		for podName, uidAfter := range podUIDs {
			if uidBefore, ok := podUIDsBeforeScale[podName]; ok && uidBefore != uidAfter {
				return false, fmt.Errorf("pod: [%s] have be recreated", podName)
			}
		}

		return true, nil
	})
}

func (oa *operatorActions) UpgradeTidbCluster(info *TidbClusterConfig) error {
	cmd := fmt.Sprintf("helm upgrade %s %s --set-string %s",
		info.ClusterName, chartPath("tidb-cluster", info.OperatorTag), info.TidbClusterHelmSetString(nil))
	glog.Info("[UPGRADE] " + cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return pingcapErrors.Wrapf(err, "failed to upgrade tidb cluster: %s", string(res))
	}
	return nil
}

func (oa *operatorActions) UpgradeTidbClusterOrDie(info *TidbClusterConfig) {
	if err := oa.UpgradeTidbCluster(info); err != nil {
		panic(err)
	}
}

func (oa *operatorActions) CheckUpgradeProgress(info *TidbClusterConfig) error {
	return wait.Poll(DefaultPollInterval, DefaultPollTimeout, func() (done bool, err error) {
		tc, err := oa.cli.PingcapV1alpha1().TidbClusters(info.Namespace).Get(info.ClusterName, metav1.GetOptions{})
		if err != nil {
			glog.Infof("failed to get tidbcluster: [%s], error: %v", info.ClusterName, err)
			return false, nil
		}

		pdSetName := controller.PDMemberName(info.ClusterName)
		pdSet, err := oa.kubeCli.AppsV1beta1().StatefulSets(info.Namespace).Get(pdSetName, metav1.GetOptions{})
		if err != nil {
			glog.Infof("failed to get pd statefulset: [%s], error: %v", pdSetName, err)
			return false, nil
		}

		tikvSetName := controller.TiKVMemberName(info.ClusterName)
		tikvSet, err := oa.kubeCli.AppsV1beta1().StatefulSets(info.Namespace).Get(tikvSetName, metav1.GetOptions{})
		if err != nil {
			glog.Infof("failed to get tikvSet statefulset: [%s], error: %v", tikvSetName, err)
			return false, nil
		}

		tidbSetName := controller.TiDBMemberName(info.ClusterName)
		tidbSet, err := oa.kubeCli.AppsV1beta1().StatefulSets(info.Namespace).Get(tidbSetName, metav1.GetOptions{})
		if err != nil {
			glog.Infof("failed to get tidbSet statefulset: [%s], error: %v", tidbSetName, err)
			return false, nil
		}

		imageUpgraded := func(memberType v1alpha1.MemberType, set *v1beta1.StatefulSet) bool {
			image := ""
			switch memberType {
			case v1alpha1.PDMemberType:
				image = tc.Spec.PD.Image
			case v1alpha1.TiKVMemberType:
				image = tc.Spec.TiKV.Image
			case v1alpha1.TiDBMemberType:
				image = tc.Spec.TiDB.Image
			}
			memberName := string(memberType)
			c, ok := getComponentContainer(set)
			if !ok || c.Image != image {
				glog.Infof("check %s image: getContainer(set).Image(%s) != tc.Spec.%s.Image(%s)",
					memberName, c.Image, strings.ToUpper(memberName), image)
			}
			return ok && c.Image == image
		}
		setUpgraded := func(set *v1beta1.StatefulSet) bool {
			return set.Generation <= *set.Status.ObservedGeneration && set.Status.CurrentRevision == set.Status.UpdateRevision
		}

		// check upgrade order
		if tc.Status.PD.Phase == v1alpha1.UpgradePhase {
			glog.Infof("pd is upgrading")
			if tc.Status.TiKV.Phase == v1alpha1.UpgradePhase {
				return false, pingcapErrors.New("tikv is upgrading while pd is upgrading")
			}
			if tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
				return false, pingcapErrors.New("tidb is upgrading while pd is upgrading")
			}
			if !imageUpgraded(v1alpha1.PDMemberType, pdSet) {
				return false, pingcapErrors.New("pd image is not updated while pd is upgrading")
			}
			if !setUpgraded(pdSet) {
				if imageUpgraded(v1alpha1.TiKVMemberType, tikvSet) {
					return false, pingcapErrors.New("tikv image is updated while pd is upgrading")
				}
				if imageUpgraded(v1alpha1.TiDBMemberType, tidbSet) {
					return false, pingcapErrors.New("tidb image is updated while pd is upgrading")
				}
			}
			return false, nil
		} else if tc.Status.TiKV.Phase == v1alpha1.UpgradePhase {
			glog.Infof("tikv is upgrading")
			if tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
				return false, pingcapErrors.New("tidb is upgrading while tikv is upgrading")
			}
			if !imageUpgraded(v1alpha1.PDMemberType, pdSet) {
				return false, pingcapErrors.New("pd image is not updated while tikv is upgrading")
			}
			if !setUpgraded(pdSet) {
				return false, pingcapErrors.New("pd stateful set is not upgraded while tikv is upgrading")
			}
			if !imageUpgraded(v1alpha1.TiKVMemberType, tikvSet) {
				return false, pingcapErrors.New("tikv image is not updated while tikv is upgrading")
			}
			if !setUpgraded(tikvSet) {
				if imageUpgraded(v1alpha1.TiDBMemberType, tidbSet) {
					return false, pingcapErrors.New("tidb image is updated while tikv is upgrading")
				}
			}
			return false, nil
		} else if tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
			glog.Infof("tidb is upgrading")
			if !imageUpgraded(v1alpha1.PDMemberType, pdSet) {
				return false, pingcapErrors.New("pd image is not updated while tidb is upgrading")
			}
			if !setUpgraded(pdSet) {
				return false, pingcapErrors.New("pd stateful set is not upgraded while tidb is upgrading")
			}
			if !imageUpgraded(v1alpha1.TiKVMemberType, tikvSet) {
				return false, pingcapErrors.New("tikv image is not updated while tidb is upgrading")
			}
			if !setUpgraded(tikvSet) {
				return false, pingcapErrors.New("tikv stateful set is not upgraded while tidb is upgrading")
			}
			if !imageUpgraded(v1alpha1.TiDBMemberType, tidbSet) {
				return false, pingcapErrors.New("tidb image is not updated while tikv is upgrading")
			}
			return false, nil
		}

		// check pd final state
		if !imageUpgraded(v1alpha1.PDMemberType, pdSet) {
			return false, nil
		}
		if !setUpgraded(pdSet) {
			glog.Infof("check pd stateful set upgraded failed")
			return false, nil
		}
		// check tikv final state
		if !imageUpgraded(v1alpha1.TiKVMemberType, tikvSet) {
			return false, nil
		}
		if !setUpgraded(tikvSet) {
			glog.Infof("check tikv stateful set upgraded failed")
			return false, nil
		}
		// check tidb final state
		if !imageUpgraded(v1alpha1.TiDBMemberType, tidbSet) {
			return false, nil
		}
		if !setUpgraded(tidbSet) {
			glog.Infof("check tidb stateful set upgraded failed")
			return false, nil
		}
		return true, nil
	})
}

func (oa *operatorActions) DeployMonitor(info *TidbClusterConfig) error { return nil }
func (oa *operatorActions) CleanMonitor(info *TidbClusterConfig) error  { return nil }

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
	if len(tc.Status.TiDB.Members) != int(tc.Spec.TiDB.Replicas) {
		glog.Infof("tidbcluster: %s/%s .status.TiDB.Members count(%d) != %d",
			ns, tcName, len(tc.Status.TiDB.Members), tc.Spec.TiDB.Replicas)
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
	_, err = oa.kubeCli.CoreV1().Services(ns).Get(controller.TiDBPeerMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		glog.Errorf("failed to get peer service: %s/%s", ns, controller.TiDBPeerMemberName(tcName))
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

func (oa *operatorActions) passwordIsSet(clusterInfo *TidbClusterConfig) (bool, error) {
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

func (oa *operatorActions) monitorNormal(clusterInfo *TidbClusterConfig) (bool, error) {
	ns := clusterInfo.Namespace
	tcName := clusterInfo.ClusterName
	monitorDeploymentName := fmt.Sprintf("%s-monitor", tcName)
	monitorDeployment, err := oa.kubeCli.AppsV1().Deployments(ns).Get(monitorDeploymentName, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("get monitor deployment: [%s/%s] failed", ns, monitorDeploymentName)
		return false, nil
	}
	if monitorDeployment.Status.ReadyReplicas < 1 {
		glog.Infof("monitor ready replicas %d < 1", monitorDeployment.Status.ReadyReplicas)
		return false, nil
	}
	if err := oa.checkPrometheus(clusterInfo); err != nil {
		glog.Infof("check [%s/%s]'s prometheus data failed: %v", ns, monitorDeploymentName, err)
		return false, nil
	}

	if err := oa.checkGrafanaData(clusterInfo); err != nil {
		glog.Infof("check [%s/%s]'s grafana data failed: %v", ns, monitorDeploymentName, err)
		return false, nil
	}
	return true, nil
}

func (oa *operatorActions) checkPrometheus(clusterInfo *TidbClusterConfig) error {
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

func (oa *operatorActions) checkGrafanaData(clusterInfo *TidbClusterConfig) error {
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

func (oa *operatorActions) cloneOperatorRepo() error {
	cmd := fmt.Sprintf("git clone https://github.com/pingcap/tidb-operator.git %s", oa.cfg.OperatorRepoDir)
	glog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil && !strings.Contains(string(res), "already exists") {
		return fmt.Errorf("failed to clone tidb-operator repository: %v, %s", err, string(res))
	}

	return nil
}

func (oa *operatorActions) checkoutTag(tagName string) error {
	cmd := fmt.Sprintf(`cd %s &&
		git stash -u &&
		git checkout %s &&
		mkdir -p /charts/%s &&
		cp -rf charts/tidb-operator /charts/%s/tidb-operator &&
		cp -rf charts/tidb-cluster /charts/%s/tidb-cluster &&
		cp -rf charts/tidb-backup /charts/%s/tidb-backup`,
		oa.cfg.OperatorRepoDir, tagName, tagName, tagName, tagName, tagName)
	glog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check tag: %s, %v, %s", tagName, err, string(res))
	}

	return nil
}

func (oa *operatorActions) DeployAdHocBackup(info *TidbClusterConfig) error {
	glog.Infof("begin to deploy adhoc backup cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)

	sets := map[string]string{
		"name":         info.BackupPVC,
		"mode":         "backup",
		"user":         "root",
		"password":     info.Password,
		"storage.size": "10Gi",
	}

	setString := info.BackupHelmSetString(sets)

	fullbackupName := fmt.Sprintf("%s-backup", info.ClusterName)
	cmd := fmt.Sprintf("helm install -n %s --namespace %s /charts/%s/tidb-backup --set-string %s",
		fullbackupName, info.Namespace, info.OperatorTag, setString)
	glog.Infof("install adhoc deployment [%s]", cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to launch adhoc backup job: %v, %s", err, string(res))
	}

	return nil
}

func (oa *operatorActions) CheckAdHocBackup(info *TidbClusterConfig) error {
	glog.Infof("begin to clean adhoc backup cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)

	jobName := fmt.Sprintf("%s-%s", info.ClusterName, info.BackupPVC)
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

func (oa *operatorActions) Restore(from *TidbClusterConfig, to *TidbClusterConfig) error {
	glog.Infof("begin to deploy restore cluster[%s] namespace[%s]", from.ClusterName, from.Namespace)

	sets := map[string]string{
		"name":         to.BackupPVC,
		"mode":         "restore",
		"user":         "root",
		"password":     to.Password,
		"storage.size": "10Gi",
	}

	setString := to.BackupHelmSetString(sets)

	restoreName := fmt.Sprintf("%s-restore", from.ClusterName)
	cmd := fmt.Sprintf("helm install -n %s --namespace %s /charts/%s/tidb-backup --set-string %s",
		restoreName, to.Namespace, to.OperatorTag, setString)
	glog.Infof("install restore [%s]", cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to launch restore job: %v, %s", err, string(res))
	}

	return nil
}

func (oa *operatorActions) CheckRestore(from *TidbClusterConfig, to *TidbClusterConfig) error {
	glog.Infof("begin to check restore backup cluster[%s] namespace[%s]", from.ClusterName, from.Namespace)
	jobName := fmt.Sprintf("%s-restore-%s", to.ClusterName, from.BackupPVC)
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

func (oa *operatorActions) ForceDeploy(info *TidbClusterConfig) error {
	if err := oa.CleanTidbCluster(info); err != nil {
		return err
	}

	if err := oa.DeployTidbCluster(info); err != nil {
		return err
	}

	return nil
}

func (info *TidbClusterConfig) QueryCount() (int, error) {
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

func (oa *operatorActions) CreateSecret(info *TidbClusterConfig) error {
	initSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      info.InitSecretName,
			Namespace: info.Namespace,
		},
		Data: map[string][]byte{
			info.UserName: []byte(info.Password),
		},
		Type: corev1.SecretTypeOpaque,
	}

	_, err := oa.kubeCli.CoreV1().Secrets(info.Namespace).Create(&initSecret)
	if err != nil && !releaseIsExist(err) {
		return err
	}

	backupSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      info.BackupSecretName,
			Namespace: info.Namespace,
		},
		Data: map[string][]byte{
			"user":     []byte(info.UserName),
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

func (oa *operatorActions) DeployScheduledBackup(info *TidbClusterConfig) error {
	glog.Infof("begin to deploy scheduled backup")

	cron := fmt.Sprintf("'*/1 * * * *'")
	sets := map[string]string{
		"clusterName":                info.ClusterName,
		"scheduledBackup.create":     "true",
		"scheduledBackup.user":       "root",
		"scheduledBackup.password":   info.Password,
		"scheduledBackup.schedule":   cron,
		"scheduledBackup.storage":    "10Gi",
		"scheduledBackup.secretName": info.BackupSecretName,
	}

	setString := info.TidbClusterHelmSetString(sets)

	cmd := fmt.Sprintf("helm upgrade %s /charts/%s/tidb-cluster --set-string %s",
		info.ClusterName, info.OperatorTag, setString)

	glog.Infof("scheduled-backup delploy [%s]", cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to launch scheduler backup job: %v, %s", err, string(res))
	}
	return nil
}

func (oa *operatorActions) CheckScheduledBackup(info *TidbClusterConfig) error {
	glog.Infof("begin to check scheduler backup cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)

	jobName := fmt.Sprintf("%s-scheduled-backup", info.ClusterName)
	fn := func() (bool, error) {
		job, err := oa.kubeCli.BatchV1beta1().CronJobs(info.Namespace).Get(jobName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("failed to get cronjobs %s ,%v", jobName, err)
			return false, nil
		}

		jobs, err := oa.kubeCli.BatchV1().Jobs(info.Namespace).List(metav1.ListOptions{})
		if err != nil {
			glog.Errorf("failed to list jobs %s ,%v", info.Namespace, err)
			return false, nil
		}

		backupJobs := []batchv1.Job{}
		for _, j := range jobs.Items {
			if pid, found := getParentUIDFromJob(j); found && pid == job.UID {
				backupJobs = append(backupJobs, j)
			}
		}

		if len(backupJobs) == 0 {
			glog.Errorf("cluster [%s] scheduler jobs is creating, please wait!", info.ClusterName)
			return false, nil
		}

		for _, j := range backupJobs {
			if j.Status.Succeeded == 0 {
				glog.Errorf("cluster [%s] back up job is not completed, please wait! ", info.ClusterName)
				return false, nil
			}
		}

		return true, nil
	}

	err := wait.Poll(DefaultPollInterval, DefaultPollTimeout, fn)
	if err != nil {
		return fmt.Errorf("failed to launch scheduler backup job: %v", err)
	}

	// sleep 1 minute for cronjob
	time.Sleep(60 * time.Second)

	dirs, err := oa.getBackupDir(info)
	if err != nil {
		return fmt.Errorf("failed to get backup dir: %v", err)
	}

	if len(dirs) <= 2 {
		return fmt.Errorf("scheduler job failed!")
	}

	return nil
}

func getParentUIDFromJob(j batchv1.Job) (types.UID, bool) {
	controllerRef := metav1.GetControllerOf(&j)

	if controllerRef == nil {
		return types.UID(""), false
	}

	if controllerRef.Kind != "CronJob" {
		glog.Infof("Job with non-CronJob parent, name %s namespace %s", j.Name, j.Namespace)
		return types.UID(""), false
	}

	return controllerRef.UID, true
}

func (oa *operatorActions) getBackupDir(info *TidbClusterConfig) ([]string, error) {
	scheduledPvcName := fmt.Sprintf("%s-scheduled-backup", info.ClusterName)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getBackupDirPodName,
			Namespace: info.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    getBackupDirPodName,
					Image:   "pingcap/tidb-cloud-backup:latest",
					Command: []string{"sleep", "3000"},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "data",
							MountPath: "/data",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: scheduledPvcName,
						},
					},
				},
			},
		},
	}

	fn := func() (bool, error) {
		_, err := oa.kubeCli.CoreV1().Pods(info.Namespace).Get(getBackupDirPodName, metav1.GetOptions{})
		if !errors.IsNotFound(err) {
			return false, nil
		}
		return true, nil
	}

	err := wait.Poll(DefaultPollInterval, DefaultPollTimeout, fn)

	if err != nil {
		return nil, fmt.Errorf("failed to delete pod %s", getBackupDirPodName)
	}

	_, err = oa.kubeCli.CoreV1().Pods(info.Namespace).Create(pod)
	if err != nil && !errors.IsAlreadyExists(err) {
		glog.Errorf("cluster: [%s/%s] create get backup dir pod failed, error :%v", info.Namespace, info.ClusterName, err)
		return nil, err
	}

	fn = func() (bool, error) {
		_, err := oa.kubeCli.CoreV1().Pods(info.Namespace).Get(getBackupDirPodName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return false, nil
		}
		return true, nil
	}

	err = wait.Poll(DefaultPollInterval, DefaultPollTimeout, fn)

	if err != nil {
		return nil, fmt.Errorf("failed to create pod %s", getBackupDirPodName)
	}

	cmd := fmt.Sprintf("kubectl exec %s -n %s ls /data", getBackupDirPodName, info.Namespace)

	time.Sleep(20 * time.Second)

	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		glog.Errorf("cluster:[%s/%s] exec :%s failed,error:%v,result:%s", info.Namespace, info.ClusterName, cmd, err, string(res))
		return nil, err
	}

	dirs := strings.Split(string(res), "\n")
	glog.Infof("dirs in pod info name [%s] dir name [%s]", scheduledPvcName, strings.Join(dirs, ","))
	return dirs, nil
}

func (info *TidbClusterConfig) FullName() string {
	return fmt.Sprintf("%s/%s", info.Namespace, info.ClusterName)
}

func (oa *operatorActions) DeployIncrementalBackup(from *TidbClusterConfig, to *TidbClusterConfig) error {
	glog.Infof("begin to deploy incremental backup cluster[%s] namespace[%s]", from.ClusterName, from.Namespace)

	sets := map[string]string{
		"binlog.pump.create":            "true",
		"binlog.drainer.destDBType":     "mysql",
		"binlog.drainer.create":         "true",
		"binlog.drainer.mysql.host":     fmt.Sprintf("%s-tidb.%s", to.ClusterName, to.Namespace),
		"binlog.drainer.mysql.user":     "root",
		"binlog.drainer.mysql.password": to.Password,
		"binlog.drainer.mysql.port":     "4000",
	}

	setString := from.TidbClusterHelmSetString(sets)

	cmd := fmt.Sprintf("helm upgrade %s /charts/%s/tidb-cluster --set-string %s",
		from.ClusterName, from.OperatorTag, setString)
	glog.Infof(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to launch scheduler backup job: %v, %s", err, string(res))
	}
	return nil
}

func (oa *operatorActions) CheckIncrementalBackup(info *TidbClusterConfig) error {
	glog.Infof("begin to check incremental backup cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)

	pumpStatefulSetName := fmt.Sprintf("%s-pump", info.ClusterName)
	fn := func() (bool, error) {
		pumpStatefulSet, err := oa.kubeCli.AppsV1().StatefulSets(info.Namespace).Get(pumpStatefulSetName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("failed to get jobs %s ,%v", pumpStatefulSetName, err)
			return false, nil
		}
		if pumpStatefulSet.Status.Replicas != pumpStatefulSet.Status.ReadyReplicas {
			glog.Errorf("pump replicas is not ready, please wait ! %s ", pumpStatefulSetName)
			return false, nil
		}

		listOps := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(
				pumpStatefulSet.Labels,
			).String(),
		}

		pods, err := oa.kubeCli.CoreV1().Pods(info.Namespace).List(listOps)
		if err != nil {
			glog.Errorf("failed to get pods via pump labels %s ,%v", pumpStatefulSetName, err)
			return false, nil
		}

		for _, pod := range pods.Items {
			if !oa.pumpHealth(info, pod.Spec.Hostname) {
				glog.Errorf("some pods is not health %s ,%v", pumpStatefulSetName, err)
				return false, nil
			}
		}

		drainerStatefulSetName := fmt.Sprintf("%s-drainer", info.ClusterName)
		drainerStatefulSet, err := oa.kubeCli.AppsV1().StatefulSets(info.Namespace).Get(drainerStatefulSetName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("failed to get jobs %s ,%v", pumpStatefulSetName, err)
			return false, nil
		}
		if drainerStatefulSet.Status.Replicas != drainerStatefulSet.Status.ReadyReplicas {
			glog.Errorf("drainer replicas is not ready, please wait ! %s ", pumpStatefulSetName)
			return false, nil
		}

		listOps = metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(
				drainerStatefulSet.Labels,
			).String(),
		}

		pods, err = oa.kubeCli.CoreV1().Pods(info.Namespace).List(listOps)
		if err != nil {
			return false, nil
		}
		for _, pod := range pods.Items {
			if !oa.drainerHealth(info, pod.Spec.Hostname) {
				return false, nil
			}
		}

		return true, nil
	}

	err := wait.Poll(DefaultPollInterval, DefaultPollTimeout, fn)
	if err != nil {
		return fmt.Errorf("failed to launch scheduler backup job: %v", err)
	}
	return nil

}

type pumpStatus struct {
	StatusMap map[string]*nodeStatus
}

type nodeStatus struct {
	State string `json:"state"`
}

func (oa *operatorActions) pumpHealth(info *TidbClusterConfig, hostName string) bool {
	pumpHealthUrl := fmt.Sprintf("%s.%s-pump.%s:8250/status", hostName, info.ClusterName, info.Namespace)
	res, err := http.Get(pumpHealthUrl)
	if err != nil {
		glog.Errorf("cluster:[%s] call %s failed,error:%v", info.ClusterName, pumpHealthUrl, err)
		return false
	}
	if res.StatusCode >= 400 {
		glog.Errorf("Error response %v", res.StatusCode)
		return false
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		glog.Errorf("cluster:[%s] read response body failed,error:%v", info.ClusterName, err)
		return false
	}
	healths := pumpStatus{}
	err = json.Unmarshal(body, &healths)
	if err != nil {
		glog.Errorf("cluster:[%s] unmarshal failed,error:%v", info.ClusterName, err)
		return false
	}
	for _, status := range healths.StatusMap {
		if status.State != "online" {
			glog.Errorf("cluster:[%s] pump's state is not online", info.ClusterName)
			return false
		}
	}
	return true
}

type drainerStatus struct {
	PumpPos map[string]int64 `json:"PumpPos"`
	Synced  bool             `json:"Synced"`
	LastTS  int64            `json:"LastTS"`
	TsMap   string           `json:"TsMap"`
}

func (oa *operatorActions) drainerHealth(info *TidbClusterConfig, hostName string) bool {
	drainerHealthUrl := fmt.Sprintf("%s.%s-drainer.%s:8249/status", hostName, info.ClusterName, info.Namespace)
	res, err := http.Get(drainerHealthUrl)
	if err != nil {
		glog.Errorf("cluster:[%s] call %s failed,error:%v", info.ClusterName, drainerHealthUrl, err)
		return false
	}
	if res.StatusCode >= 400 {
		glog.Errorf("Error response %v", res.StatusCode)
		return false
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		glog.Errorf("cluster:[%s] read response body failed,error:%v", info.ClusterName, err)
		return false
	}
	healths := drainerStatus{}
	err = json.Unmarshal(body, &healths)
	if err != nil {
		glog.Errorf("cluster:[%s] unmarshal failed,error:%v", info.ClusterName, err)
		return false
	}
	return len(healths.PumpPos) > 0 && healths.Synced
}
