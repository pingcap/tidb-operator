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
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	// To register MySQL driver
	_ "github.com/go-sql-driver/mysql"

	"github.com/ghodss/yaml"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	pingcapErrors "github.com/pingcap/errors"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	utildiscovery "github.com/pingcap/tidb-operator/pkg/util/discovery"
	e2eutil "github.com/pingcap/tidb-operator/tests/e2e/util"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/e2e/util/proxiedpdclient"
	"github.com/pingcap/tidb-operator/tests/e2e/util/proxiedtidbclient"
	utilstatefulset "github.com/pingcap/tidb-operator/tests/e2e/util/statefulset"
	"github.com/pingcap/tidb-operator/tests/pkg/apimachinery"
	"github.com/pingcap/tidb-operator/tests/pkg/blockwriter"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	"github.com/pingcap/tidb-operator/tests/pkg/metrics"
	"github.com/pingcap/tidb-operator/tests/pkg/webhook"
	"github.com/pingcap/tidb-operator/tests/slack"
	admissionV1beta1 "k8s.io/api/admissionregistration/v1beta1"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/klog"
	aggregatorclientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

const (
	period = 5 * time.Minute

	tidbControllerName string = "tidb-controller-manager"
	tidbSchedulerName  string = "tidb-scheduler"

	// NodeUnreachablePodReason is defined in k8s.io/kubernetes/pkg/util/node
	// but not in client-go and apimachinery, so we define it here
	NodeUnreachablePodReason = "NodeLost"

	WebhookServiceName = "webhook-service"
)

func NewOperatorActions(cli versioned.Interface,
	kubeCli kubernetes.Interface,
	asCli asclientset.Interface,
	aggrCli aggregatorclientset.Interface,
	apiExtCli apiextensionsclientset.Interface,
	pollInterval time.Duration,
	operatorConfig *OperatorConfig,
	cfg *Config,
	clusters []*TidbClusterConfig,
	fw portforward.PortForward, f *framework.Framework) OperatorActions {

	var tcStsGetter typedappsv1.StatefulSetsGetter
	if operatorConfig != nil && operatorConfig.Enabled(features.AdvancedStatefulSet) {
		tcStsGetter = helper.NewHijackClient(kubeCli, asCli).AppsV1()
	} else {
		tcStsGetter = kubeCli.AppsV1()
	}

	oa := &operatorActions{
		framework:    f,
		cli:          cli,
		kubeCli:      kubeCli,
		pdControl:    pdapi.NewDefaultPDControl(kubeCli),
		asCli:        asCli,
		aggrCli:      aggrCli,
		apiExtCli:    apiExtCli,
		tcStsGetter:  tcStsGetter,
		pollInterval: pollInterval,
		cfg:          cfg,
		fw:           fw,
		crdUtil:      NewCrdTestUtil(cli, kubeCli, asCli, tcStsGetter),
	}
	if fw != nil {
		kubeCfg, err := framework.LoadConfig()
		framework.ExpectNoError(err)
		oa.tidbControl = proxiedtidbclient.NewProxiedTiDBClient(fw, kubeCfg.TLSClientConfig.CAData)
	} else {
		oa.tidbControl = controller.NewDefaultTiDBControl(kubeCli)
	}
	oa.clusterEvents = make(map[string]*clusterEvent)
	for _, c := range clusters {
		oa.clusterEvents[c.String()] = &clusterEvent{
			ns:          c.Namespace,
			clusterName: c.ClusterName,
			events:      make([]event, 0),
		}
	}
	return oa
}

const (
	DefaultPollTimeout          time.Duration = 20 * time.Minute
	DefaultPollInterval         time.Duration = 5 * time.Second
	BackupAndRestorePollTimeOut time.Duration = 60 * time.Minute
	grafanaUsername                           = "admin"
	grafanaPassword                           = "admin"
	operartorChartName                        = "tidb-operator"
	tidbClusterChartName                      = "tidb-cluster"
	backupChartName                           = "tidb-backup"
	drainerChartName                          = "tidb-drainer"
	statbilityTestTag                         = "stability"
)

type OperatorActions interface {
	CleanCRDOrDie()
	InstallCRDOrDie(info *OperatorConfig)
	DeployOperator(info *OperatorConfig) error
	DeployOperatorOrDie(info *OperatorConfig)
	CleanOperator(info *OperatorConfig) error
	CleanOperatorOrDie(info *OperatorConfig)
	UpgradeOperator(info *OperatorConfig) error
	UpgradeOperatorOrDie(info *OperatorConfig)
	DumpAllLogs(info *OperatorConfig, clusterInfos []*TidbClusterConfig) error
	DeployTidbCluster(info *TidbClusterConfig) error
	DeployTidbClusterOrDie(info *TidbClusterConfig)
	CleanTidbCluster(info *TidbClusterConfig) error
	CleanTidbClusterOrDie(info *TidbClusterConfig)
	CheckTidbClusterStatus(info *TidbClusterConfig) error
	CheckTidbClusterStatusOrDie(info *TidbClusterConfig)
	BeginInsertDataTo(info *TidbClusterConfig) error
	BeginInsertDataToOrDie(info *TidbClusterConfig)
	StopInsertDataTo(info *TidbClusterConfig)
	ScaleTidbCluster(info *TidbClusterConfig) error
	ScaleTidbClusterOrDie(info *TidbClusterConfig)
	CheckScaleInSafely(info *TidbClusterConfig) error
	CheckScaledCorrectly(info *TidbClusterConfig, podUIDsBeforeScale map[string]types.UID) error
	CheckUpgradeOrDie(ctx context.Context, info *TidbClusterConfig)
	CheckUpgrade(ctx context.Context, info *TidbClusterConfig) error
	UpgradeTidbCluster(info *TidbClusterConfig) error
	UpgradeTidbClusterOrDie(info *TidbClusterConfig)
	DeployAdHocBackup(info *TidbClusterConfig) error
	CheckAdHocBackup(info *TidbClusterConfig) (string, error)
	DeployScheduledBackup(info *TidbClusterConfig) error
	CheckScheduledBackup(info *TidbClusterConfig) error
	DeployIncrementalBackup(from *TidbClusterConfig, to *TidbClusterConfig, withDrainer bool, ts string) error
	CheckIncrementalBackup(info *TidbClusterConfig, withDrainer bool) error
	DeployDrainer(info *DrainerConfig, from *TidbClusterConfig) error
	DeployDrainerOrDie(info *DrainerConfig, from *TidbClusterConfig)
	CheckDrainer(info *DrainerConfig, source *TidbClusterConfig) error
	Restore(from *TidbClusterConfig, to *TidbClusterConfig) error
	CheckRestore(from *TidbClusterConfig, to *TidbClusterConfig) error
	RestoreIncrementalFiles(from *DrainerConfig, to *TidbClusterConfig, stopTSO int64) error
	ForceDeploy(info *TidbClusterConfig) error
	CreateSecret(info *TidbClusterConfig) error
	GetPodUIDMap(info *TidbClusterConfig) (map[string]types.UID, error)
	GetNodeMap(info *TidbClusterConfig, component string) (map[string][]string, error)
	TruncateSSTFileThenCheckFailover(info *TidbClusterConfig, tikvFailoverPeriod time.Duration) error
	TruncateSSTFileThenCheckFailoverOrDie(info *TidbClusterConfig, tikvFailoverPeriod time.Duration)
	DeletePDDataThenCheckFailover(info *TidbClusterConfig, tikvFailoverPeriod time.Duration) error
	DeletePDDataThenCheckFailoverOrDie(info *TidbClusterConfig, tikvFailoverPeriod time.Duration)
	CheckFailoverPending(info *TidbClusterConfig, node string, faultPoint *time.Time) (bool, error)
	CheckFailoverPendingOrDie(clusters []*TidbClusterConfig, node string, faultPoint *time.Time)
	CheckFailover(info *TidbClusterConfig, faultNode string) (bool, error)
	CheckFailoverOrDie(clusters []*TidbClusterConfig, faultNode string)
	CheckRecover(cluster *TidbClusterConfig) (bool, error)
	CheckRecoverOrDie(clusters []*TidbClusterConfig)
	CheckK8sAvailable(excludeNodes map[string]string, excludePods map[string]*corev1.Pod) error
	CheckK8sAvailableOrDie(excludeNodes map[string]string, excludePods map[string]*corev1.Pod)
	CheckOperatorAvailable(operatorConfig *OperatorConfig) error
	CheckTidbClustersAvailable(infos []*TidbClusterConfig) error
	CheckOperatorDownOrDie(infos []*TidbClusterConfig)
	CheckTidbClustersAvailableOrDie(infos []*TidbClusterConfig)
	CheckEtcdDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig, faultNode string)
	CheckKubeletDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig, faultNode string)
	CheckOneApiserverDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig, faultNode string)
	CheckAllApiserverDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig)
	CheckKubeProxyDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig)
	CheckKubeSchedulerDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig)
	CheckKubeControllerManagerDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig)
	RegisterWebHookAndService(configName, namespace, service string, context *apimachinery.CertContext) error
	RegisterWebHookAndServiceOrDie(configName, namespace, service string, context *apimachinery.CertContext)
	CleanWebHookAndService(name string) error
	CleanWebHookAndServiceOrDie(name string)
	RunEventWorker()
	EmitEvent(info *TidbClusterConfig, msg string)
	BackupRestore(from, to *TidbClusterConfig) error
	BackupRestoreOrDie(from, to *TidbClusterConfig)
	BackupAndRestoreToMultipleClusters(source *TidbClusterConfig, targets []BackupTarget) error
	BackupAndRestoreToMultipleClustersOrDie(source *TidbClusterConfig, targets []BackupTarget)
	LabelNodes() error
	LabelNodesOrDie()
	CheckDisasterTolerance(info *TidbClusterConfig) error
	CheckDisasterToleranceOrDie(info *TidbClusterConfig)
	CheckUpgradeComplete(info *TidbClusterConfig) error
	CheckUpgradeCompleteOrDie(info *TidbClusterConfig)
	CheckInitSQL(info *TidbClusterConfig) error
	CheckInitSQLOrDie(info *TidbClusterConfig)
	DeployAndCheckPump(tc *TidbClusterConfig) error
	WaitForTidbClusterReady(tc *v1alpha1.TidbCluster, timeout, pollInterval time.Duration) error
	WaitPodOnNodeReadyOrDie(clusters []*TidbClusterConfig, faultNode string)
	DataIsTheSameAs(from, to *TidbClusterConfig) (bool, error)
}

type operatorActions struct {
	framework          *framework.Framework
	cli                versioned.Interface
	kubeCli            kubernetes.Interface
	asCli              asclientset.Interface
	aggrCli            aggregatorclientset.Interface
	apiExtCli          apiextensionsclientset.Interface
	tcStsGetter        typedappsv1.StatefulSetsGetter
	pdControl          pdapi.PDControlInterface
	tidbControl        controller.TiDBControlInterface
	pollInterval       time.Duration
	cfg                *Config
	clusterEvents      map[string]*clusterEvent
	lock               sync.Mutex
	eventWorkerRunning bool
	fw                 portforward.PortForward
	crdUtil            *CrdTestUtil
}

type clusterEvent struct {
	ns          string
	clusterName string
	events      []event
}

type event struct {
	message string
	ts      int64
}

var _ = OperatorActions(&operatorActions{})

type OperatorConfig struct {
	Namespace                 string
	ReleaseName               string
	Image                     string
	Tag                       string
	ControllerManagerReplicas *int
	SchedulerImage            string
	SchedulerTag              string
	SchedulerReplicas         *int
	Features                  []string
	LogLevel                  string
	WebhookServiceName        string
	WebhookSecretName         string
	WebhookConfigName         string
	Context                   *apimachinery.CertContext
	ImagePullPolicy           corev1.PullPolicy
	TestMode                  bool
	WebhookEnabled            bool
	PodWebhookEnabled         bool
	StsWebhookEnabled         bool
	DefaultingEnabled         bool
	ValidatingEnabled         bool
	Cabundle                  string
	BackupImage               string
	AutoFailover              *bool
	// Additional STRING values, set via --set-string flag.
	StringValues map[string]string
}

type TidbClusterConfig struct {
	BackupName             string
	Namespace              string
	ClusterName            string
	EnablePVReclaim        bool
	OperatorTag            string
	PDImage                string
	TiKVImage              string
	TiDBImage              string
	PumpImage              string
	StorageClassName       string
	Password               string
	RecordCount            string
	InsertBatchSize        string
	Resources              map[string]string
	Args                   map[string]string
	blockWriterPod         *corev1.Pod
	Monitor                bool
	UserName               string
	InitSecretName         string
	BackupSecretName       string
	EnableConfigMapRollout bool
	ClusterVersion         string

	PDPreStartScript   string
	TiDBPreStartScript string
	TiKVPreStartScript string

	PDMaxReplicas       int
	TiKVGrpcConcurrency int
	TiDBTokenLimit      int
	PDLogLevel          string

	BlockWriteConfig blockwriter.Config
	GrafanaClient    *metrics.Client
	TopologyKey      string

	pumpConfig    []string
	drainerConfig []string

	Clustrer *v1alpha1.TidbCluster
}

func (tc *TidbClusterConfig) String() string {
	return fmt.Sprintf("%s/%s", tc.Namespace, tc.ClusterName)
}

func (tc *TidbClusterConfig) GenerateBackupDirPodName() string {
	return fmt.Sprintf("%s-get-backup-dir", tc.ClusterName)
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
		"enablePVReclaim":         strconv.FormatBool(tc.EnablePVReclaim),
		"pd.storageClassName":     tc.StorageClassName,
		"tikv.storageClassName":   tc.StorageClassName,
		"tidb.storageClassName":   tc.StorageClassName,
		"tidb.password":           tc.Password,
		"pd.image":                tc.PDImage,
		"tikv.image":              tc.TiKVImage,
		"tidb.image":              tc.TiDBImage,
		"binlog.pump.image":       tc.PumpImage,
		"tidb.passwordSecretName": tc.InitSecretName,
		"monitor.create":          strconv.FormatBool(tc.Monitor),
		"enableConfigMapRollout":  strconv.FormatBool(tc.EnableConfigMapRollout),
		"pd.preStartScript":       tc.PDPreStartScript,
		"tikv.preStartScript":     tc.TiKVPreStartScript,
		"tidb.preStartScript":     tc.TiDBPreStartScript,
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

func (oi *OperatorConfig) OperatorHelmSetBoolean() string {
	set := map[string]bool{
		"admissionWebhook.create":                      oi.WebhookEnabled,
		"admissionWebhook.validation.pods":             oi.PodWebhookEnabled,
		"admissionWebhook.mutation.pods":               oi.PodWebhookEnabled,
		"admissionWebhook.validation.statefulSets":     oi.StsWebhookEnabled,
		"admissionWebhook.mutation.pingcapResources":   oi.DefaultingEnabled,
		"admissionWebhook.validation.pingcapResources": oi.ValidatingEnabled,
	}
	arr := make([]string, 0, len(set))
	for k, v := range set {
		arr = append(arr, fmt.Sprintf("--set %s=%v", k, v))
	}
	return strings.Join(arr, " ")
}

func (oi *OperatorConfig) OperatorHelmSetString(m map[string]string) string {
	set := map[string]string{
		"operatorImage":             oi.Image,
		"tidbBackupManagerImage":    oi.BackupImage,
		"scheduler.logLevel":        "4",
		"testMode":                  strconv.FormatBool(oi.TestMode),
		"admissionWebhook.cabundle": oi.Cabundle,
	}
	if oi.LogLevel != "" {
		set["controllerManager.logLevel"] = oi.LogLevel
	}
	if oi.SchedulerImage != "" {
		set["scheduler.kubeSchedulerImageName"] = oi.SchedulerImage
	}
	if string(oi.ImagePullPolicy) != "" {
		set["imagePullPolicy"] = string(oi.ImagePullPolicy)
	}
	if oi.ControllerManagerReplicas != nil {
		set["controllerManager.replicas"] = strconv.Itoa(*oi.ControllerManagerReplicas)
	}
	if oi.SchedulerReplicas != nil {
		set["scheduler.replicas"] = strconv.Itoa(*oi.SchedulerReplicas)
	}
	if oi.SchedulerTag != "" {
		set["scheduler.kubeSchedulerImageTag"] = oi.SchedulerTag
	}
	if len(oi.Features) > 0 {
		set["features"] = fmt.Sprintf("{%s}", strings.Join(oi.Features, ","))
	}
	if oi.Enabled(features.AdvancedStatefulSet) {
		set["advancedStatefulset.create"] = "true"
	}
	if oi.AutoFailover != nil {
		set["controllerManager.autoFailover"] = strconv.FormatBool(*oi.AutoFailover)
	}

	// merge with additional STRING values
	for k, v := range oi.StringValues {
		set[k] = v
	}

	arr := make([]string, 0, len(set))
	for k, v := range set {
		arr = append(arr, fmt.Sprintf("%s=%s", k, v))
	}
	return fmt.Sprintf("\"%s\"", strings.Join(arr, ","))
}

func (oi *OperatorConfig) Enabled(feature string) bool {
	k := fmt.Sprintf("%s=true", feature)
	for _, v := range oi.Features {
		if v == k {
			return true
		}
	}
	return false
}

func (oa *operatorActions) runKubectlOrDie(args ...string) string {
	cmd := "kubectl"
	klog.Infof("Running '%s %s'", cmd, strings.Join(args, " "))
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		klog.Fatalf("Failed to run '%s %s'\nCombined output: %q\nError: %v", cmd, strings.Join(args, " "), string(out), err)
	}
	klog.Infof("Combined output: %q", string(out))
	return string(out)
}

func (oa *operatorActions) CleanCRDOrDie() {
	crdList, err := oa.apiExtCli.ApiextensionsV1beta1().CustomResourceDefinitions().List(metav1.ListOptions{})
	framework.ExpectNoError(err)
	for _, crd := range crdList.Items {
		if !strings.HasSuffix(crd.Name, ".pingcap.com") {
			framework.Logf("CRD %q ignored", crd.Name)
			continue
		}
		framework.Logf("Deleting CRD %q", crd.Name)
		err = oa.apiExtCli.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.Name, &metav1.DeleteOptions{})
		framework.ExpectNoError(err)
		// Even if DELETE API request succeeds, the CRD object may still exists
		// in ap server. We should wait for it to be gone.
		e2eutil.WaitForCRDNotFound(oa.apiExtCli, crd.Name)
		framework.ExpectNoError(err)
	}
}

// InstallCRDOrDie install CRDs and wait for them to be established in Kubernetes.
func (oa *operatorActions) InstallCRDOrDie(info *OperatorConfig) {
	if info.Enabled(features.AdvancedStatefulSet) {
		if isSupported, err := utildiscovery.IsAPIGroupVersionSupported(oa.kubeCli.Discovery(), "apiextensions.k8s.io/v1"); err != nil {
			klog.Fatal(err)
		} else if isSupported {
			oa.runKubectlOrDie("apply", "-f", oa.manifestPath("e2e/advanced-statefulset-crd.v1.yaml"))
		} else {
			oa.runKubectlOrDie("apply", "-f", oa.manifestPath("e2e/advanced-statefulset-crd.v1beta1.yaml"))
		}
	}
	oa.runKubectlOrDie("apply", "-f", oa.manifestPath("e2e/crd.yaml"))
	oa.runKubectlOrDie("apply", "-f", oa.manifestPath("e2e/data-resource-crd.yaml"))
	klog.Infof("Wait for all CRDs are established")
	e2eutil.WaitForCRDsEstablished(oa.apiExtCli, labels.Everything())
	// workaround for https://github.com/kubernetes/kubernetes/issues/65517
	klog.Infof("force sync kubectl cache")
	cmdArgs := []string{"sh", "-c", "rm -rf ~/.kube/cache ~/.kube/http-cache"}
	_, err := exec.Command(cmdArgs[0], cmdArgs[1:]...).CombinedOutput()
	if err != nil {
		klog.Fatalf("Failed to run '%s': %v", strings.Join(cmdArgs, " "), err)
	}
}

func (oa *operatorActions) DeployOperator(info *OperatorConfig) error {
	klog.Infof("deploying tidb-operator %s", info.ReleaseName)

	if info.Tag != "e2e" {
		if err := oa.cloneOperatorRepo(); err != nil {
			return err
		}
		if err := oa.checkoutTag(info.Tag); err != nil {
			return err
		}
	}

	if info.WebhookEnabled {
		err := oa.setCabundleFromApiServer(info)
		if err != nil {
			return err
		}
	}

	cmd := fmt.Sprintf(`helm install %s --name %s --namespace %s %s --set-string %s`,
		oa.operatorChartPath(info.Tag),
		info.ReleaseName,
		info.Namespace,
		info.OperatorHelmSetBoolean(),
		info.OperatorHelmSetString(nil))
	klog.Info(cmd)

	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to deploy operator: %v, %s", err, string(res))
	}
	klog.Infof("deploy operator response: %v\n", string(res))

	klog.Infof("Wait for all apiesrvices are available")
	return e2eutil.WaitForAPIServicesAvaiable(oa.aggrCli, labels.Everything())
}

func (oa *operatorActions) DeployOperatorOrDie(info *OperatorConfig) {
	if err := oa.DeployOperator(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CleanOperator(info *OperatorConfig) error {
	klog.Infof("cleaning tidb-operator %s", info.ReleaseName)

	res, err := exec.Command("helm", "del", "--purge", info.ReleaseName).CombinedOutput()

	if err == nil || !releaseIsNotFound(err) {
		return nil
	}

	return fmt.Errorf("failed to clear operator: %v, %s", err, string(res))
}

func (oa *operatorActions) CleanOperatorOrDie(info *OperatorConfig) {
	if err := oa.CleanOperator(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) UpgradeOperator(info *OperatorConfig) error {
	klog.Infof("upgrading tidb-operator %s", info.ReleaseName)

	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Labels()).String(),
	}
	pods1, err := oa.kubeCli.CoreV1().Pods(metav1.NamespaceAll).List(listOptions)
	if err != nil {
		return err
	}

	if info.Tag != "e2e" {
		if err := oa.checkoutTag(info.Tag); err != nil {
			return err
		}
	}

	cmd := fmt.Sprintf("helm upgrade %s %s %s --set-string %s",
		info.ReleaseName, oa.operatorChartPath(info.Tag),
		info.OperatorHelmSetBoolean(),
		info.OperatorHelmSetString(nil))

	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to upgrade operator to: %s, %v, %s", info.Image, err, string(res))
	}

	klog.Infof("Wait for all apiesrvices are available")
	err = e2eutil.WaitForAPIServicesAvaiable(oa.aggrCli, labels.Everything())
	if err != nil {
		return err
	}

	if info.Tag == "e2e" {
		return nil
	}

	// ensure pods unchanged when upgrading operator
	waitFn := func() (done bool, err error) {
		pods2, err := oa.kubeCli.CoreV1().Pods(metav1.NamespaceAll).List(listOptions)
		if err != nil {
			klog.Error(err)
			return false, nil
		}

		err = ensurePodsUnchanged(pods1, pods2)
		if err != nil {
			return true, err
		}

		return false, nil
	}

	err = wait.Poll(oa.pollInterval, 5*time.Minute, waitFn)
	if err == wait.ErrWaitTimeout {
		return nil
	}
	return err
}

func ensurePodsUnchanged(pods1, pods2 *corev1.PodList) error {
	pods1UIDs := getUIDs(pods1)
	pods2UIDs := getUIDs(pods2)
	pods1Yaml, err := yaml.Marshal(pods1)
	if err != nil {
		return err
	}
	pods2Yaml, err := yaml.Marshal(pods2)
	if err != nil {
		return err
	}
	if reflect.DeepEqual(pods1UIDs, pods2UIDs) {
		klog.V(4).Infof("%s", string(pods1Yaml))
		klog.V(4).Infof("%s", string(pods2Yaml))
		klog.V(4).Infof("%v", pods1UIDs)
		klog.V(4).Infof("%v", pods2UIDs)
		klog.V(4).Infof("pods unchanged after operator upgraded")
		return nil
	}

	klog.Infof("%s", string(pods1Yaml))
	klog.Infof("%s", string(pods2Yaml))
	klog.Infof("%v", pods1UIDs)
	klog.Infof("%v", pods2UIDs)
	return fmt.Errorf("some pods changed after operator upgraded")
}

func getUIDs(pods *corev1.PodList) []string {
	arr := make([]string, 0, len(pods.Items))

	for _, pod := range pods.Items {
		arr = append(arr, string(pod.UID))
	}

	sort.Strings(arr)
	return arr
}

func (oa *operatorActions) UpgradeOperatorOrDie(info *OperatorConfig) {
	if err := oa.UpgradeOperator(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) DeployTidbCluster(info *TidbClusterConfig) error {
	ns := info.Namespace
	tcName := info.ClusterName
	if _, err := oa.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{}); err == nil {
		// already deployed
		return nil
	}

	klog.Infof("deploying tidb cluster [%s/%s]", info.Namespace, info.ClusterName)
	oa.EmitEvent(info, "DeployTidbCluster")

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

	cmd := fmt.Sprintf("helm install %s  --name %s --namespace %s --set-string %s",
		oa.tidbClusterChartPath(info.OperatorTag), info.ClusterName, info.Namespace, info.TidbClusterHelmSetString(nil))

	svFilePath, err := info.BuildSubValues(oa.tidbClusterChartPath(info.OperatorTag))
	if err != nil {
		return err
	}
	cmd = fmt.Sprintf(" %s --values %s", cmd, svFilePath)
	klog.Info(cmd)

	if res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to deploy tidbcluster: %s/%s, %v, %s",
			info.Namespace, info.ClusterName, err, string(res))
	}

	return nil
}

func (oa *operatorActions) DeployTidbClusterOrDie(info *TidbClusterConfig) {
	if err := oa.DeployTidbCluster(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CleanTidbCluster(info *TidbClusterConfig) error {
	klog.Infof("cleaning tidbcluster %s/%s", info.Namespace, info.ClusterName)
	oa.EmitEvent(info, "CleanTidbCluster")
	ns := info.Namespace
	tcName := info.ClusterName

	oa.StopInsertDataTo(info)

	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			label.InstanceLabelKey: tcName,
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      label.ComponentLabelKey,
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{label.PDLabelVal, label.TiKVLabelVal},
			},
		},
	})
	if err != nil {
		return err
	}
	pvcList, err := oa.kubeCli.CoreV1().PersistentVolumeClaims(ns).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}
	var beforePVCNames []string
	for _, pvc := range pvcList.Items {
		beforePVCNames = append(beforePVCNames, pvc.GetName())
	}
	klog.V(4).Info(beforePVCNames)

	pvList, err := oa.kubeCli.CoreV1().PersistentVolumes().List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}
	var beforePVNames []string
	for _, pv := range pvList.Items {
		beforePVNames = append(beforePVNames, pv.GetName())
		klog.V(4).Infof("%s, %s, %v", pv.Name, pv.Spec.PersistentVolumeReclaimPolicy, pv.Labels)
		klog.V(4).Info(pv.Spec.ClaimRef)
	}
	klog.V(4).Info(beforePVNames)

	charts := []string{
		info.ClusterName,
		fmt.Sprintf("%s-backup", info.ClusterName),
		fmt.Sprintf("%s-restore", info.ClusterName),
		fmt.Sprintf("%s-scheduler-backup", info.ClusterName),
		fmt.Sprintf("%s-%s-drainer", info.ClusterName, DbTypeFile),
		fmt.Sprintf("%s-%s-drainer", info.ClusterName, DbTypeTiDB),
		fmt.Sprintf("%s-%s-drainer", info.ClusterName, DbTypeMySQL),
	}
	for _, chartName := range charts {
		res, err := exec.Command("helm", "del", "--purge", chartName).CombinedOutput()
		if err != nil && !notFound(string(res)) {
			return fmt.Errorf("failed to delete chart: %s/%s, %v, %s",
				info.Namespace, chartName, err, string(res))
		}
	}

	time.Sleep(time.Minute)

	pvcList, err = oa.kubeCli.CoreV1().PersistentVolumeClaims(ns).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}
	var afterPVCNames []string
	for _, pvc := range pvcList.Items {
		afterPVCNames = append(afterPVCNames, pvc.GetName())
	}
	klog.V(4).Info(afterPVCNames)
	if !reflect.DeepEqual(beforePVCNames, afterPVCNames) {
		return fmt.Errorf("pvc changed when we delete cluster: %s/%s, before: %v, after: %v",
			ns, tcName, beforePVCNames, afterPVCNames)
	}

	waitPVFn := func() (done bool, err error) {
		pvList, err = oa.kubeCli.CoreV1().PersistentVolumes().List(metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return false, nil
		}
		var afterPVNames []string
		for _, pv := range pvList.Items {
			afterPVNames = append(afterPVNames, pv.GetName())
		}
		klog.V(4).Info(afterPVNames)

		if !reflect.DeepEqual(beforePVNames, afterPVNames) {
			klog.Errorf("pv changed when we delete cluster: %s/%s, before: %v, after: %v",
				ns, tcName, beforePVNames, afterPVNames)
			return false, nil
		}

		return true, nil
	}

	err = wait.Poll(oa.pollInterval, DefaultPollTimeout, waitPVFn)
	if err != nil {
		return err
	}

	err = oa.kubeCli.CoreV1().Pods(info.Namespace).Delete(info.GenerateBackupDirPodName(), &metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete dir pod %v", err)
	}

	err = oa.kubeCli.CoreV1().Pods(info.Namespace).Delete(blockWriterPodName(info), nil)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete blockwriter pod %v", err)
	}

	err = oa.kubeCli.CoreV1().Secrets(info.Namespace).Delete(info.InitSecretName, &metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete secret: %s, %v", info.InitSecretName, err)
	}

	setStr := label.New().Instance(info.ClusterName).String()

	// delete all jobs
	allJobsSet := label.Label{}.Instance(info.ClusterName).String()
	if res, err := exec.Command("kubectl", "delete", "jobs", "-n", info.Namespace, "-l", allJobsSet).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete jobs: %v, %s", err, string(res))
	}

	resources := []string{"pvc"}
	for _, resource := range resources {
		if res, err := exec.Command("kubectl", "delete", resource, "-n", info.Namespace, "-l",
			setStr).CombinedOutput(); err != nil {
			return fmt.Errorf("failed to delete %s: %v, %s", resource, err, string(res))
		}
	}

	// delete pvc of drainer
	drainerPvcSet := label.Label{}.Instance(info.ClusterName).Component("drainer").String()
	if res, err := exec.Command("kubectl", "delete", "pvc", "-n", info.Namespace, "-l",
		drainerPvcSet).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete drainer pvc: %v, %s", err, string(res))
	}

	// delete all configmaps
	allConfigMaps := label.New().Instance(info.ClusterName).String()
	if res, err := exec.Command("kubectl", "delete", "configmaps", "-n", info.Namespace, "-l", allConfigMaps).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete configmaps: %v, %s", err, string(res))
	}

	err = wait.Poll(10*time.Second, 5*time.Minute, func() (done bool, err error) {
		patchPVCmd := fmt.Sprintf("kubectl get pv --no-headers -l %s=%s,%s=%s,%s=%s | awk '{print $1}' | "+
			"xargs -I {} kubectl patch pv {} -p '{\"spec\":{\"persistentVolumeReclaimPolicy\":\"Delete\"}}'",
			label.ManagedByLabelKey, "tidb-operator",
			label.NamespaceLabelKey, info.Namespace,
			label.InstanceLabelKey, info.ClusterName)
		klog.V(4).Info(patchPVCmd)
		if res, err := exec.Command("/bin/sh", "-c", patchPVCmd).CombinedOutput(); err != nil {
			klog.Errorf(fmt.Errorf("failed to patch pv: %v, %s", err, string(res)).Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	pollFn := func() (bool, error) {
		if res, err := exec.Command("kubectl", "get", "po", "--output=name", "-n", info.Namespace, "-l", setStr).
			CombinedOutput(); err != nil || len(res) != 0 {
			klog.V(4).Infof("waiting for tidbcluster: %s/%s pods deleting, %v, [%s]",
				info.Namespace, info.ClusterName, err, string(res))
			return false, nil
		}

		pvCmd := fmt.Sprintf("kubectl get pv | grep %s | grep %s 2>/dev/null|grep Released",
			info.Namespace, info.ClusterName)
		klog.V(4).Info(pvCmd)
		if res, err := exec.Command("/bin/sh", "-c", pvCmd).CombinedOutput(); len(res) == 0 {
			return true, nil
		} else if err != nil {
			klog.V(4).Infof("waiting for tidbcluster: %s/%s pv deleting, %v, %s",
				info.Namespace, info.ClusterName, err, string(res))
			return false, nil
		}
		return true, nil
	}
	return wait.PollImmediate(oa.pollInterval, DefaultPollTimeout, pollFn)
}

func (oa *operatorActions) CleanTidbClusterOrDie(info *TidbClusterConfig) {
	if err := oa.CleanTidbCluster(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CheckTidbClusterStatus(info *TidbClusterConfig) error {
	klog.Infof("checking tidb cluster [%s/%s] status", info.Namespace, info.ClusterName)
	if info.Clustrer != nil {
		return oa.crdUtil.WaitForTidbClusterReady(info.Clustrer, 5*time.Minute, 1*time.Minute)
	}

	ns := info.Namespace
	tcName := info.ClusterName
	if err := wait.Poll(oa.pollInterval, 10*time.Minute, func() (bool, error) {
		var tc *v1alpha1.TidbCluster
		var err error
		if tc, err = oa.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{}); err != nil {
			klog.Errorf("failed to get tidbcluster: %s/%s, %v", ns, tcName, err)
			return false, nil
		}

		if b, err := oa.pdMembersReadyFn(tc); !b && err == nil {
			return false, nil
		}
		if b, err := oa.tikvMembersReadyFn(tc); !b && err == nil {
			return false, nil
		}

		klog.V(4).Infof("check tidb cluster begin tidbMembersReadyFn")
		if b, err := oa.tidbMembersReadyFn(tc); !b && err == nil {
			return false, nil
		}

		klog.V(4).Infof("check tidb cluster begin reclaimPolicySyncFn")
		if b, err := oa.reclaimPolicySyncFn(tc); !b && err == nil {
			return false, nil
		}

		klog.V(4).Infof("check tidb cluster begin metaSyncFn")
		if b, err := oa.metaSyncFn(tc); !b && err == nil {
			return false, nil
		} else if err != nil {
			klog.Error(err)
			return false, nil
		}

		klog.V(4).Infof("check tidb cluster begin schedulerHAFn")
		if b, err := oa.schedulerHAFn(tc); !b && err == nil {
			return false, nil
		}

		klog.V(4).Infof("check all pd and tikv instances have not pod scheduling annotation")
		if info.OperatorTag != "v1.0.0" {
			if b, err := oa.podsScheduleAnnHaveDeleted(tc); !b && err == nil {
				return false, nil
			}
		}

		klog.V(4).Infof("check store labels")
		if b, err := oa.storeLabelsIsSet(tc, info.TopologyKey); !b && err == nil {
			return false, nil
		} else if err != nil {
			return false, err
		}

		klog.V(4).Infof("check tidb cluster begin passwordIsSet")
		if b, err := oa.passwordIsSet(info); !b && err == nil {
			return false, nil
		}

		if info.Monitor {
			klog.V(4).Infof("check tidb monitor normal")
			if b, err := oa.monitorNormal(info); !b && err == nil {
				return false, nil
			}
		}
		if info.EnableConfigMapRollout {
			klog.V(4).Info("check tidb cluster configuration synced")
			if b, err := oa.checkTidbClusterConfigUpdated(tc, info); !b && err == nil {
				return false, nil
			}
		}
		if info.EnablePVReclaim {
			klog.V(4).Infof("check reclaim pvs success when scale in pd or tikv")
			if b, err := oa.checkReclaimPVSuccess(tc); !b && err == nil {
				return false, nil
			}
		}
		return true, nil
	}); err != nil {
		klog.Errorf("check tidb cluster status failed: %s", err.Error())
		return fmt.Errorf("failed to waiting for tidbcluster %s/%s ready in 10 minutes", ns, tcName)
	}

	return nil
}

func (oa *operatorActions) CheckTidbClusterStatusOrDie(info *TidbClusterConfig) {
	if err := oa.CheckTidbClusterStatus(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) getBlockWriterPod(info *TidbClusterConfig, database string) *corev1.Pod {

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: info.Namespace,
			Name:      blockWriterPodName(info),
			Labels: map[string]string{
				"app": "blockwriter",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "blockwriter",
					Image:           oa.cfg.E2EImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/usr/local/bin/blockwriter"},
					Args: []string{
						fmt.Sprintf("--namespace=%s", info.Namespace),
						fmt.Sprintf("--cluster-name=%s", info.ClusterName),
						fmt.Sprintf("--database=%s", database),
						fmt.Sprintf("--password=%s", info.Password),
						fmt.Sprintf("--table-num=%d", info.BlockWriteConfig.TableNum),
						fmt.Sprintf("--concurrency=%d", info.BlockWriteConfig.Concurrency),
						fmt.Sprintf("--batch-size=%d", info.BlockWriteConfig.BatchSize),
						fmt.Sprintf("--raw-size=%d", info.BlockWriteConfig.RawSize),
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}
	if info.OperatorTag != "e2e" {
		pod.Spec.Containers[0].ImagePullPolicy = corev1.PullAlways
	}
	return pod
}

func (oa *operatorActions) BeginInsertDataTo(info *TidbClusterConfig) error {
	oa.EmitEvent(info, fmt.Sprintf("BeginInsertData: concurrency: %d", info.BlockWriteConfig.Concurrency))

	pod := oa.getBlockWriterPod(info, "sbtest")
	pod, err := oa.kubeCli.CoreV1().Pods(info.Namespace).Create(pod)
	if err != nil {
		klog.Error(err)
		return err
	}
	info.blockWriterPod = pod
	err = e2epod.WaitForPodRunningInNamespace(oa.kubeCli, pod)
	if err != nil {
		return err
	}
	klog.Infof("begin insert Data in pod[%s/%s]", pod.Namespace, pod.Name)
	return nil
}

func (oa *operatorActions) BeginInsertDataToOrDie(info *TidbClusterConfig) {
	err := oa.BeginInsertDataTo(info)
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) StopInsertDataTo(info *TidbClusterConfig) {
	if info.blockWriterPod == nil {
		return
	}
	oa.EmitEvent(info, "StopInsertData")

	err := wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		pod := info.blockWriterPod
		err = oa.kubeCli.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		slack.NotifyAndPanic(err)
	}
	info.blockWriterPod = nil
}

func (oa *operatorActions) manifestPath(tag string) string {
	return filepath.Join(oa.cfg.ManifestDir, tag)
}

func (oa *operatorActions) chartPath(name string, tag string) string {
	return filepath.Join(oa.cfg.ChartDir, tag, name)
}

func (oa *operatorActions) operatorChartPath(tag string) string {
	return oa.chartPath(operartorChartName, tag)
}

func (oa *operatorActions) tidbClusterChartPath(tag string) string {
	return oa.chartPath(tidbClusterChartName, tag)
}

func (oa *operatorActions) backupChartPath(tag string) string {
	return oa.chartPath(backupChartName, tag)
}

func (oa *operatorActions) drainerChartPath(tag string) string {
	return oa.chartPath(drainerChartName, tag)
}

func (oa *operatorActions) ScaleTidbCluster(info *TidbClusterConfig) error {
	oa.EmitEvent(info, fmt.Sprintf("ScaleTidbCluster to pd: %s, tikv: %s, tidb: %s",
		info.Args["pd.replicas"], info.Args["tikv.replicas"], info.Args["tidb.replicas"]))

	cmd, err := oa.getHelmUpgradeClusterCmd(info, nil)
	if err != nil {
		return err
	}
	klog.Info("[SCALE] " + cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return pingcapErrors.Wrapf(err, "failed to scale tidb cluster: %s", string(res))
	}
	return nil
}

func (oa *operatorActions) ScaleTidbClusterOrDie(info *TidbClusterConfig) {
	if err := oa.ScaleTidbCluster(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CheckScaleInSafely(info *TidbClusterConfig) error {
	return wait.Poll(oa.pollInterval, DefaultPollTimeout, func() (done bool, err error) {
		tc, err := oa.cli.PingcapV1alpha1().TidbClusters(info.Namespace).Get(info.ClusterName, metav1.GetOptions{})
		if err != nil {
			klog.Infof("failed to get tidbcluster when scale in tidbcluster, error: %v", err)
			return false, nil
		}

		tikvSetName := controller.TiKVMemberName(info.ClusterName)
		tikvSet, err := oa.tcStsGetter.StatefulSets(info.Namespace).Get(tikvSetName, metav1.GetOptions{})
		if err != nil {
			klog.Infof("failed to get tikvSet statefulset: [%s], error: %v", tikvSetName, err)
			return false, nil
		}

		pdClient, cancel, err := oa.getPDClient(tc)
		if err != nil {
			klog.Errorf("Failed to create external PD client for tidb cluster %q: %v", tc.GetName(), err)
			return false, nil
		}
		defer cancel()

		stores, err := pdClient.GetStores()
		if err != nil {
			klog.Infof("pdClient.GetStores failed,error: %v", err)
			return false, nil
		}
		if len(stores.Stores) > int(*tikvSet.Spec.Replicas) {
			klog.Infof("stores.Stores: %v", stores.Stores)
			klog.Infof("tikvSet.Spec.Replicas: %d", *tikvSet.Spec.Replicas)
			return false, fmt.Errorf("the tikvSet.Spec.Replicas may reduce before tikv complete offline")
		}

		if *tikvSet.Spec.Replicas == tc.Spec.TiKV.Replicas {
			return true, nil
		}

		return false, nil
	})
}

func (oa *operatorActions) CheckScaledCorrectly(info *TidbClusterConfig, podUIDsBeforeScale map[string]types.UID) error {
	return wait.Poll(oa.pollInterval, DefaultPollTimeout, func() (done bool, err error) {
		podUIDs, err := oa.GetPodUIDMap(info)
		if err != nil {
			klog.Infof("failed to get pd pods's uid, error: %v", err)
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

func (oa *operatorActions) setPartitionAnnotation(namespace, tcName, component string, ordinal int) error {
	// add annotation to pause statefulset upgrade process
	cmd := fmt.Sprintf("kubectl annotate tc %s -n %s tidb.pingcap.com/%s-partition=%d --overwrite",
		tcName, namespace, component, ordinal)
	klog.Infof("%s", cmd)
	output, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("fail to set annotation for [%s/%s], component: %s, partition: %d, err: %v, output: %s", namespace, tcName, component, ordinal, err, string(output))
	}
	return nil
}

func (oa *operatorActions) UpgradeTidbCluster(info *TidbClusterConfig) error {
	oa.EmitEvent(info, "UpgradeTidbCluster")

	cmd, err := oa.getHelmUpgradeClusterCmd(info, nil)
	if err != nil {
		return err
	}
	klog.Info("[UPGRADE] " + cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return pingcapErrors.Wrapf(err, "failed to upgrade tidb cluster: %s", string(res))
	}
	return nil
}

func (oa *operatorActions) UpgradeTidbClusterOrDie(info *TidbClusterConfig) {
	if err := oa.UpgradeTidbCluster(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CheckUpgrade(ctx context.Context, info *TidbClusterConfig) error {
	ns := info.Namespace
	tcName := info.ClusterName

	findStoreFn := func(tc *v1alpha1.TidbCluster, podName string) string {
		for storeID, store := range tc.Status.TiKV.Stores {
			if store.PodName == podName {
				return storeID
			}
		}

		return ""
	}

	// set partition annotation to protect tikv pod
	if err := oa.setPartitionAnnotation(ns, info.ClusterName, label.TiKVLabelVal, 1); err != nil {
		return err
	}

	// set partition annotation to protect tidb pod
	if err := oa.setPartitionAnnotation(ns, info.ClusterName, label.TiDBLabelVal, 1); err != nil {
		return err
	}

	tc, err := oa.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get tidbcluster: %s/%s, %v", ns, tcName, err)

	}

	replicas := tc.TiKVStsDesiredReplicas()
	for i := replicas - 1; i >= 0; i-- {
		err := wait.PollImmediate(1*time.Second, 10*time.Minute, func() (done bool, err error) {
			podName := fmt.Sprintf("%s-tikv-%d", tcName, i)
			scheduler := fmt.Sprintf("evict-leader-scheduler-%s", findStoreFn(tc, podName))
			pdClient, cancel, err := oa.getPDClient(tc)
			if err != nil {
				klog.Errorf("Failed to create external PD client for tidb cluster %q: %v", tc.GetName(), err)
				return false, nil
			}
			defer cancel()
			schedulers, err := pdClient.GetEvictLeaderSchedulers()
			if err != nil {
				klog.Errorf("failed to get evict leader schedulers, %v", err)
				return false, nil
			}
			klog.V(4).Infof("index:%d,schedulers:%v,error:%v", i, schedulers, err)
			if len(schedulers) > 1 {
				klog.Errorf("there are too many evict leader schedulers: %v", schedulers)
				for _, s := range schedulers {
					if s == scheduler {
						klog.Infof("found scheudler: %s", scheduler)
						return true, nil
					}
				}
				return false, nil
			}
			if len(schedulers) == 0 {
				return false, nil
			}
			if schedulers[0] == scheduler {
				klog.Infof("index: %d,the schedulers: %s = %s", i, schedulers[0], scheduler)
				return true, nil
			}
			klog.Errorf("index: %d,the scheduler: %s != %s", i, schedulers[0], scheduler)
			return false, nil
		})
		if err != nil {
			klog.Errorf("failed to check upgrade %s/%s, %v", ns, tcName, err)
			return err
		}
	}

	if err := oa.checkManualPauseComponent(info, label.TiKVLabelVal); err != nil {
		return err
	}
	// remove the protect partition annotation for tikv
	if err := oa.setPartitionAnnotation(ns, info.ClusterName, label.TiKVLabelVal, 0); err != nil {
		return err
	}

	if err := oa.checkManualPauseComponent(info, label.TiDBLabelVal); err != nil {
		return err
	}
	// remove the protect partition annotation for tidb
	if err := oa.setPartitionAnnotation(ns, info.ClusterName, label.TiDBLabelVal, 0); err != nil {
		return err
	}

	return wait.PollImmediate(1*time.Second, 6*time.Minute, func() (done bool, err error) {
		pdClient, cancel, err := oa.getPDClient(tc)
		if err != nil {
			klog.Errorf("Failed to create external PD client for tidb cluster %q: %v", tc.GetName(), err)
			return false, nil
		}
		defer cancel()
		schedulers, err := pdClient.GetEvictLeaderSchedulers()
		if err != nil {
			klog.Errorf("failed to get evict leader schedulers, %v", err)
			return false, nil
		}
		if len(schedulers) == 0 {
			return true, nil
		}
		klog.Errorf("schedulers: %v is not empty", schedulers)
		return false, nil
	})
}

func (oa *operatorActions) CheckUpgradeOrDie(ctx context.Context, info *TidbClusterConfig) {
	if err := oa.CheckUpgrade(ctx, info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) DeployMonitor(info *TidbClusterConfig) error { return nil }
func (oa *operatorActions) CleanMonitor(info *TidbClusterConfig) error  { return nil }

// getMemberContainer gets member container
func getMemberContainer(kubeCli kubernetes.Interface, stsGetter typedappsv1.StatefulSetsGetter, namespace, tcName, component string) (*corev1.Container, bool) {
	sts, err := stsGetter.StatefulSets(namespace).Get(fmt.Sprintf("%s-%s", tcName, component), metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get sts for component %s of cluster %s/%s", component, namespace, tcName)
		return nil, false
	}
	return getStsContainer(kubeCli, sts, component)
}

func getStsContainer(kubeCli kubernetes.Interface, sts *apps.StatefulSet, containerName string) (*corev1.Container, bool) {
	listOption := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(sts.Spec.Selector.MatchLabels).String(),
	}
	podList, err := kubeCli.CoreV1().Pods(sts.Namespace).List(listOption)
	if err != nil {
		klog.Errorf("fail to get pods for container %s of sts %s/%s", containerName, sts.Namespace, sts.Name)
		return nil, false
	}
	if len(podList.Items) == 0 {
		klog.Errorf("no pods found for component %s of cluster %s/%s", containerName, sts.Namespace, sts.Name)
		return nil, false
	}
	pod := podList.Items[0]
	if len(pod.Spec.Containers) == 0 {
		klog.Errorf("no containers found for component %s of cluster %s/%s", containerName, sts.Namespace, sts.Name)
		return nil, false
	}
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			return &container, true
		}
	}
	return nil, false
}

func (oa *operatorActions) pdMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	if tc.Spec.PD == nil {
		return true, nil
	}
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	pdSetName := controller.PDMemberName(tcName)

	pdSet, err := oa.tcStsGetter.StatefulSets(ns).Get(pdSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get statefulset: %s/%s, %v", ns, pdSetName, err)
		return false, nil
	}

	if pdSet.Status.CurrentRevision != pdSet.Status.UpdateRevision {
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), pdSet) {
		return false, nil
	}

	if tc.Status.PD.StatefulSet == nil {
		klog.Infof("tidbcluster: %s/%s .status.PD.StatefulSet is nil", ns, tcName)
		return false, nil
	}
	failureCount := len(tc.Status.PD.FailureMembers)
	replicas := tc.Spec.PD.Replicas + int32(failureCount)
	if *pdSet.Spec.Replicas != replicas {
		klog.Infof("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, pdSetName, *pdSet.Spec.Replicas, replicas)
		return false, nil
	}
	if pdSet.Status.ReadyReplicas != tc.Spec.PD.Replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, pdSetName, pdSet.Status.ReadyReplicas, tc.Spec.PD.Replicas)
		return false, nil
	}
	if len(tc.Status.PD.Members) != int(tc.Spec.PD.Replicas) {
		klog.Infof("tidbcluster: %s/%s .status.PD.Members count(%d) != %d",
			ns, tcName, len(tc.Status.PD.Members), tc.Spec.PD.Replicas)
		return false, nil
	}
	if pdSet.Status.ReadyReplicas != pdSet.Status.Replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, pdSetName, pdSet.Status.ReadyReplicas, pdSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(oa.kubeCli, oa.tcStsGetter, ns, tc.Name, label.PDLabelVal)
	if !found {
		klog.Infof("statefulset: %s/%s not found containers[name=pd] or pod %s-0",
			ns, pdSetName, pdSetName)
		return false, nil
	}

	if tc.PDImage() != c.Image {
		klog.Infof("statefulset: %s/%s .spec.template.spec.containers[name=pd].image(%s) != %s",
			ns, pdSetName, c.Image, tc.PDImage())
		return false, nil
	}

	for _, member := range tc.Status.PD.Members {
		if !member.Health {
			klog.Infof("tidbcluster: %s/%s pd member(%s/%s) is not health",
				ns, tcName, member.ID, member.Name)
			return false, nil
		}
	}

	pdServiceName := controller.PDMemberName(tcName)
	pdPeerServiceName := controller.PDPeerMemberName(tcName)
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(pdServiceName, metav1.GetOptions{}); err != nil {
		klog.Errorf("failed to get service: %s/%s", ns, pdServiceName)
		return false, nil
	}
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(pdPeerServiceName, metav1.GetOptions{}); err != nil {
		klog.Errorf("failed to get peer service: %s/%s", ns, pdPeerServiceName)
		return false, nil
	}

	return true, nil
}

func (oa *operatorActions) tikvMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	if tc.Spec.TiKV == nil {
		return true, nil
	}
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	tikvSetName := controller.TiKVMemberName(tcName)

	tikvSet, err := oa.tcStsGetter.StatefulSets(ns).Get(tikvSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get statefulset: %s/%s, %v", ns, tikvSetName, err)
		return false, nil
	}

	if tikvSet.Status.CurrentRevision != tikvSet.Status.UpdateRevision {
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), tikvSet) {
		return false, nil
	}

	if tc.Status.TiKV.StatefulSet == nil {
		klog.Infof("tidbcluster: %s/%s .status.TiKV.StatefulSet is nil", ns, tcName)
		return false, nil
	}
	failureCount := len(tc.Status.TiKV.FailureStores)
	replicas := tc.Spec.TiKV.Replicas + int32(failureCount)
	if *tikvSet.Spec.Replicas != replicas {
		klog.Infof("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, tikvSetName, *tikvSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tikvSet.Status.ReadyReplicas != replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, tikvSetName, tikvSet.Status.ReadyReplicas, replicas)
		return false, nil
	}
	if len(tc.Status.TiKV.Stores) != int(replicas) {
		klog.Infof("tidbcluster: %s/%s .status.TiKV.Stores.count(%d) != %d",
			ns, tcName, len(tc.Status.TiKV.Stores), replicas)
		return false, nil
	}
	if tikvSet.Status.ReadyReplicas != tikvSet.Status.Replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, tikvSetName, tikvSet.Status.ReadyReplicas, tikvSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(oa.kubeCli, oa.tcStsGetter, ns, tc.Name, label.TiKVLabelVal)
	if !found {
		klog.Infof("statefulset: %s/%s not found containers[name=tikv] or pod %s-0",
			ns, tikvSetName, tikvSetName)
		return false, nil
	}

	if tc.TiKVImage() != c.Image {
		klog.Infof("statefulset: %s/%s .spec.template.spec.containers[name=tikv].image(%s) != %s",
			ns, tikvSetName, c.Image, tc.TiKVImage())
		return false, nil
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.State != v1alpha1.TiKVStateUp {
			klog.Infof("tidbcluster: %s/%s's store(%s) state != %s", ns, tcName, store.ID, v1alpha1.TiKVStateUp)
			return false, nil
		}
	}

	tikvPeerServiceName := controller.TiKVPeerMemberName(tcName)
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(tikvPeerServiceName, metav1.GetOptions{}); err != nil {
		klog.Errorf("failed to get peer service: %s/%s", ns, tikvPeerServiceName)
		return false, nil
	}

	return true, nil
}

func (oa *operatorActions) tiflashMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	tiflashSetName := controller.TiFlashMemberName(tcName)

	tiflashSet, err := oa.tcStsGetter.StatefulSets(ns).Get(tiflashSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("TiFlash failed to get statefulset: %s/%s, %v", ns, tiflashSetName, err)
		return false, nil
	}

	if tiflashSet.Status.CurrentRevision != tiflashSet.Status.UpdateRevision {
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), tiflashSet) {
		return false, nil
	}

	if tc.Status.TiFlash.StatefulSet == nil {
		klog.Infof("tidbcluster: %s/%s .status.TiFlash.StatefulSet is nil", ns, tcName)
		return false, nil
	}
	failureCount := len(tc.Status.TiFlash.FailureStores)
	replicas := tc.Spec.TiFlash.Replicas + int32(failureCount)
	if *tiflashSet.Spec.Replicas != replicas {
		klog.Infof("TiFlash statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, tiflashSetName, *tiflashSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tiflashSet.Status.ReadyReplicas != replicas {
		klog.Infof("TiFlash statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, tiflashSetName, tiflashSet.Status.ReadyReplicas, replicas)
		return false, nil
	}
	if len(tc.Status.TiFlash.Stores) != int(replicas) {
		klog.Infof("tidbcluster: %s/%s .status.TiFlash.Stores.count(%d) != %d",
			ns, tcName, len(tc.Status.TiFlash.Stores), replicas)
		return false, nil
	}
	if tiflashSet.Status.ReadyReplicas != tiflashSet.Status.Replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, tiflashSetName, tiflashSet.Status.ReadyReplicas, tiflashSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(oa.kubeCli, oa.tcStsGetter, ns, tc.Name, label.TiFlashLabelVal)
	if !found {
		klog.Infof("statefulset: %s/%s not found containers[name=tiflash] or pod %s-0",
			ns, tiflashSetName, tiflashSetName)
		return false, nil
	}

	if tc.TiFlashImage() != c.Image {
		klog.Infof("statefulset: %s/%s .spec.template.spec.containers[name=tiflash].image(%s) != %s",
			ns, tiflashSetName, c.Image, tc.TiFlashImage())
		return false, nil
	}

	for _, store := range tc.Status.TiFlash.Stores {
		if store.State != v1alpha1.TiKVStateUp {
			klog.Infof("TiFlash tidbcluster: %s/%s's store(%s) state != %s", ns, tcName, store.ID, v1alpha1.TiKVStateUp)
			return false, nil
		}
	}

	tiflashPeerServiceName := controller.TiFlashPeerMemberName(tcName)
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(tiflashPeerServiceName, metav1.GetOptions{}); err != nil {
		klog.Errorf("failed to get peer service: %s/%s", ns, tiflashPeerServiceName)
		return false, nil
	}
	klog.Infof("TiFlash ready: %s/%s", ns, tcName)
	return true, nil
}

func (oa *operatorActions) tidbMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	if tc.Spec.TiDB == nil {
		return true, nil
	}
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	tidbSetName := controller.TiDBMemberName(tcName)

	tidbSet, err := oa.tcStsGetter.StatefulSets(ns).Get(tidbSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get statefulset: %s/%s, %v", ns, tidbSetName, err)
		return false, nil
	}

	if tidbSet.Status.CurrentRevision != tidbSet.Status.UpdateRevision {
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), tidbSet) {
		return false, nil
	}

	if tc.Status.TiDB.StatefulSet == nil {
		klog.Infof("tidbcluster: %s/%s .status.TiDB.StatefulSet is nil", ns, tcName)
		return false, nil
	}
	failureCount := len(tc.Status.TiDB.FailureMembers)
	replicas := tc.Spec.TiDB.Replicas + int32(failureCount)
	if *tidbSet.Spec.Replicas != replicas {
		klog.Infof("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, tidbSetName, *tidbSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tidbSet.Status.ReadyReplicas != tc.Spec.TiDB.Replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, tidbSetName, tidbSet.Status.ReadyReplicas, tc.Spec.TiDB.Replicas)
		return false, nil
	}
	if len(tc.Status.TiDB.Members) != int(tc.Spec.TiDB.Replicas) {
		klog.Infof("tidbcluster: %s/%s .status.TiDB.Members count(%d) != %d",
			ns, tcName, len(tc.Status.TiDB.Members), tc.Spec.TiDB.Replicas)
		return false, nil
	}
	if tidbSet.Status.ReadyReplicas != tidbSet.Status.Replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, tidbSetName, tidbSet.Status.ReadyReplicas, tidbSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(oa.kubeCli, oa.tcStsGetter, ns, tc.Name, label.TiDBLabelVal)
	if !found {
		klog.Infof("statefulset: %s/%s not found containers[name=tidb] or pod %s-0",
			ns, tidbSetName, tidbSetName)
		return false, nil
	}

	if tc.TiDBImage() != c.Image {
		klog.Infof("statefulset: %s/%s .spec.template.spec.containers[name=tidb].image(%s) != %s",
			ns, tidbSetName, c.Image, tc.TiDBImage())
		return false, nil
	}

	_, err = oa.kubeCli.CoreV1().Services(ns).Get(tidbSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get service: %s/%s", ns, tidbSetName)
		return false, nil
	}
	_, err = oa.kubeCli.CoreV1().Services(ns).Get(controller.TiDBPeerMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get peer service: %s/%s", ns, controller.TiDBPeerMemberName(tcName))
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
		klog.Errorf("failed to list pvs for tidbcluster %s/%s, %v", ns, tcName, err)
		return false, nil
	}

	for _, pvc := range pvcList.Items {
		pvName := pvc.Spec.VolumeName
		if pv, err := oa.kubeCli.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{}); err != nil {
			klog.Errorf("failed to get pv: %s, error: %v", pvName, err)
			return false, nil
		} else if pv.Spec.PersistentVolumeReclaimPolicy != *tc.Spec.PVReclaimPolicy {
			klog.Errorf("pv: %s's reclaimPolicy is not Retain", pvName)
			return false, nil
		}
	}

	return true, nil
}

func (oa *operatorActions) metaSyncFn(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	pdClient, cancel, err := oa.getPDClient(tc)
	if err != nil {
		klog.Errorf("Failed to create external PD client for tidb cluster %q: %v", tc.GetName(), err)
		return false, nil
	}
	defer cancel()
	var cluster *metapb.Cluster
	if cluster, err = pdClient.GetCluster(); err != nil {
		klog.Errorf("failed to get cluster from pdControl: %s/%s, error: %v", ns, tcName, err)
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
		klog.Errorf("failed to list pods for tidbcluster %s/%s, %v", ns, tcName, err)
		return false, nil
	}

outerLoop:
	for _, pod := range podList.Items {
		podName := pod.GetName()
		if pod.Labels[label.ClusterIDLabelKey] != clusterID {
			klog.Infof("tidbcluster %s/%s's pod %s's label %s not equals %s ",
				ns, tcName, podName, label.ClusterIDLabelKey, clusterID)
			return false, nil
		}

		component := pod.Labels[label.ComponentLabelKey]
		switch component {
		case label.PDLabelVal:
			var memberID string
			members, err := pdClient.GetMembers()
			if err != nil {
				klog.Errorf("failed to get members for tidbcluster %s/%s, %v", ns, tcName, err)
				return false, nil
			}
			for _, member := range members.Members {
				if member.Name == podName {
					memberID = strconv.FormatUint(member.GetMemberId(), 10)
					break
				}
			}
			if memberID == "" {
				klog.Errorf("tidbcluster: %s/%s's pod %s label [%s] is empty",
					ns, tcName, podName, label.MemberIDLabelKey)
				return false, nil
			}
			if pod.Labels[label.MemberIDLabelKey] != memberID {
				return false, fmt.Errorf("tidbcluster: %s/%s's pod %s label [%s] not equals %s",
					ns, tcName, podName, label.MemberIDLabelKey, memberID)
			}
		case label.TiKVLabelVal:
			var storeID string
			stores, err := pdClient.GetStores()
			if err != nil {
				klog.Errorf("failed to get stores for tidbcluster %s/%s, %v", ns, tcName, err)
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
				klog.Errorf("tidbcluster: %s/%s's pod %s label [%s] is empty",
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
			klog.Errorf("failed to get pvc %s/%s for pod %s/%s", ns, pvcName, ns, podName)
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
			klog.Errorf("failed to get pv for pvc %s/%s, %v", ns, pvcName, err)
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
			klog.Errorf("failed to list pods for tidbcluster %s/%s, %v", ns, tcName, err)
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

func (oa *operatorActions) podsScheduleAnnHaveDeleted(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Labels()).String(),
	}

	pvcList, err := oa.kubeCli.CoreV1().PersistentVolumeClaims(ns).List(listOptions)
	if err != nil {
		klog.Errorf("failed to list pvcs for tidb cluster %s/%s, err: %v", ns, tcName, err)
		return false, nil
	}

	for _, pvc := range pvcList.Items {
		pvcName := pvc.GetName()
		l := label.Label(pvc.Labels)
		if !(l.IsPD() || l.IsTiKV()) {
			continue
		}

		if _, exist := pvc.Annotations[label.AnnPVCPodScheduling]; exist {
			klog.Errorf("tidb cluster %s/%s pvc %s has pod scheduling annotation", ns, tcName, pvcName)
			return false, nil
		}
	}

	return true, nil
}

func (oa *operatorActions) checkReclaimPVSuccess(tc *v1alpha1.TidbCluster) (bool, error) {
	// check pv reclaim	for pd
	if err := oa.checkComponentReclaimPVSuccess(tc, label.PDLabelVal); err != nil {
		klog.Errorf(err.Error())
		return false, nil
	}

	// check pv reclaim for tikv
	if err := oa.checkComponentReclaimPVSuccess(tc, label.TiKVLabelVal); err != nil {
		klog.Errorf(err.Error())
		return false, nil
	}
	return true, nil
}

func (oa *operatorActions) checkComponentReclaimPVSuccess(tc *v1alpha1.TidbCluster, component string) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	var replica int
	switch component {
	case label.PDLabelVal:
		replica = int(tc.Spec.PD.Replicas)
	case label.TiKVLabelVal:
		replica = int(tc.Spec.TiKV.Replicas)
	default:
		return fmt.Errorf("check tidb cluster %s/%s component %s is not supported", ns, tcName, component)
	}

	pvcList, err := oa.getComponentPVCList(tc, component)
	if err != nil {
		return err
	}

	pvList, err := oa.getComponentPVList(tc, component)
	if err != nil {
		return err
	}

	if len(pvcList) != replica {
		return fmt.Errorf("tidb cluster %s/%s component %s pvc has not been reclaimed completely, expected: %d, got %d", ns, tcName, component, replica, len(pvcList))
	}

	if len(pvList) != replica {
		return fmt.Errorf("tidb cluster %s/%s component %s pv has not been reclaimed completely, expected: %d, got %d", ns, tcName, component, replica, len(pvList))
	}

	return nil
}

func (oa *operatorActions) getComponentPVCList(tc *v1alpha1.TidbCluster, component string) ([]corev1.PersistentVolumeClaim, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Component(component).Labels()).String(),
	}

	pvcList, err := oa.kubeCli.CoreV1().PersistentVolumeClaims(ns).List(listOptions)
	if err != nil {
		err := fmt.Errorf("failed to list pvcs for tidb cluster %s/%s, component: %s, err: %v", ns, tcName, component, err)
		return nil, err
	}
	return pvcList.Items, nil
}

func (oa *operatorActions) getComponentPVList(tc *v1alpha1.TidbCluster, component string) ([]corev1.PersistentVolume, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Component(component).Namespace(ns).Labels()).String(),
	}

	pvList, err := oa.kubeCli.CoreV1().PersistentVolumes().List(listOptions)
	if err != nil {
		err := fmt.Errorf("failed to list pvs for tidb cluster %s/%s, component: %s, err: %v", ns, tcName, component, err)
		return nil, err
	}
	return pvList.Items, nil
}

func (oa *operatorActions) storeLabelsIsSet(tc *v1alpha1.TidbCluster, topologyKey string) (bool, error) {
	pdClient, cancel, err := oa.getPDClient(tc)
	if err != nil {
		klog.Errorf("Failed to create external PD client for tidb cluster %q: %v", tc.GetName(), err)
		return false, nil
	}
	defer cancel()
	for _, store := range tc.Status.TiKV.Stores {
		storeID, err := strconv.ParseUint(store.ID, 10, 64)
		if err != nil {
			return false, err
		}
		storeInfo, err := pdClient.GetStore(storeID)
		if err != nil {
			return false, nil
		}
		if len(storeInfo.Store.Labels) == 0 {
			return false, nil
		}
		for _, label := range storeInfo.Store.Labels {
			if label.Key != topologyKey {
				return false, nil
			}
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
		klog.Errorf("failed to get job %s/%s, %v", ns, jobName, err)
		return false, nil
	}
	if job.Status.Succeeded < 1 {
		klog.Errorf("tidbcluster: %s/%s password setter job not finished", ns, tcName)
		return false, nil
	}

	var db *sql.DB
	dsn, cancel, err := oa.getTiDBDSN(ns, tcName, "test", clusterInfo.Password)
	if err != nil {
		klog.Errorf("failed to get TiDB DSN: %v", err)
		return false, nil
	}
	defer cancel()
	if db, err = sql.Open("mysql", dsn); err != nil {
		klog.Errorf("can't open connection to mysql: %s, %v", dsn, err)
		return false, nil
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		klog.Errorf("can't connect to mysql: %s with password %s, %v", dsn, clusterInfo.Password, err)
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
		klog.Errorf("get monitor deployment: [%s/%s] failed", ns, monitorDeploymentName)
		return false, nil
	}
	if monitorDeployment.Status.ReadyReplicas < 1 {
		klog.Infof("monitor ready replicas %d < 1", monitorDeployment.Status.ReadyReplicas)
		return false, nil
	}
	if err := oa.checkPrometheus(clusterInfo); err != nil {
		klog.Infof("check [%s/%s]'s prometheus data failed: %v", ns, monitorDeploymentName, err)
		return false, nil
	}

	if err := oa.checkGrafanaData(clusterInfo); err != nil {
		klog.Infof("check [%s/%s]'s grafana data failed: %v", ns, monitorDeploymentName, err)
		return false, nil
	}
	return true, nil
}

func (oa *operatorActions) checkTidbClusterConfigUpdated(tc *v1alpha1.TidbCluster, clusterInfo *TidbClusterConfig) (bool, error) {
	if ok := oa.checkPdConfigUpdated(tc, clusterInfo); !ok {
		return false, nil
	}
	if ok := oa.checkTiKVConfigUpdated(tc, clusterInfo); !ok {
		return false, nil
	}
	if ok := oa.checkTiDBConfigUpdated(tc, clusterInfo); !ok {
		return false, nil
	}
	return true, nil
}

func (oa *operatorActions) checkPdConfigUpdated(tc *v1alpha1.TidbCluster, clusterInfo *TidbClusterConfig) bool {
	pdClient, cancel, err := oa.getPDClient(tc)
	if err != nil {
		klog.Errorf("Failed to create external PD client for tidb cluster %q: %v", tc.GetName(), err)
		return false
	}
	defer cancel()
	config, err := pdClient.GetConfig()
	if err != nil {
		klog.Errorf("failed to get PD configuraion from tidb cluster [%s/%s]", tc.Namespace, tc.Name)
		return false
	}
	if len(clusterInfo.PDLogLevel) > 0 && clusterInfo.PDLogLevel != config.Log.Level {
		klog.Errorf("check [%s/%s] PD logLevel configuration updated failed: desired [%s], actual [%s] not equal",
			tc.Namespace,
			tc.Name,
			clusterInfo.PDLogLevel,
			config.Log.Level)
		return false
	}
	// TODO: fix #487 PD configuration update for persisted configurations
	//if clusterInfo.PDMaxReplicas > 0 && config.Replication.MaxReplicas != uint64(clusterInfo.PDMaxReplicas) {
	//	klog.Errorf("check [%s/%s] PD maxReplicas configuration updated failed: desired [%d], actual [%d] not equal",
	//		tc.Namespace,
	//		tc.Name,
	//		clusterInfo.PDMaxReplicas,
	//		config.Replication.MaxReplicas)
	//	return false
	//}
	return true
}

func (oa *operatorActions) checkTiDBConfigUpdated(tc *v1alpha1.TidbCluster, clusterInfo *TidbClusterConfig) bool {
	ordinals, err := util.GetPodOrdinals(tc, v1alpha1.TiDBMemberType)
	if err != nil {
		klog.Errorf("failed to get pod ordinals for tidb cluster %s/%s (member: %v)", tc.Namespace, tc.Name, v1alpha1.TiDBMemberType)
		return false
	}
	for i := range ordinals {
		config, err := oa.tidbControl.GetSettings(tc, int32(i))
		if err != nil {
			klog.Errorf("failed to get TiDB configuration from cluster [%s/%s], ordinal: %d, error: %v", tc.Namespace, tc.Name, i, err)
			return false
		}
		if clusterInfo.TiDBTokenLimit > 0 && uint(clusterInfo.TiDBTokenLimit) != config.TokenLimit {
			klog.Errorf("check [%s/%s] TiDB instance [%d] configuration updated failed: desired [%d], actual [%d] not equal",
				tc.Namespace, tc.Name, i, clusterInfo.TiDBTokenLimit, config.TokenLimit)
			return false
		}
	}
	return true
}

func (oa *operatorActions) checkTiKVConfigUpdated(tc *v1alpha1.TidbCluster, clusterInfo *TidbClusterConfig) bool {
	// TODO: check if TiKV configuration updated
	return true
}

func (oa *operatorActions) checkPrometheus(clusterInfo *TidbClusterConfig) error {
	ns := clusterInfo.Namespace
	tcName := clusterInfo.ClusterName
	return checkPrometheusCommon(tcName, ns, oa.fw)
}

func (oa *operatorActions) checkGrafanaData(clusterInfo *TidbClusterConfig) error {
	ns := clusterInfo.Namespace
	tcName := clusterInfo.ClusterName
	grafanaClient, err := checkGrafanaDataCommon(tcName, ns, clusterInfo.GrafanaClient, oa.fw)
	if err != nil {
		return err
	}
	if clusterInfo.GrafanaClient == nil && grafanaClient != nil {
		clusterInfo.GrafanaClient = grafanaClient
	}
	return nil
}

func getDatasourceID(addr string) (int, error) {
	u := fmt.Sprintf("http://%s/api/datasources", addr)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}

	req.SetBasicAuth(grafanaUsername, grafanaPassword)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			klog.Warningf("close response failed, err: %v", err)
		}
	}()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	datasources := []struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	}{}

	if err := json.Unmarshal(buf, &datasources); err != nil {
		return 0, err
	}

	for _, ds := range datasources {
		if ds.Name == "tidb-cluster" {
			return ds.Id, nil
		}
	}

	return 0, pingcapErrors.New("not found tidb-cluster datasource")
}

func GetD(ns, tcName, databaseName, password string) string {
	return fmt.Sprintf("root:%s@(%s-tidb.%s:4000)/%s?charset=utf8", password, tcName, ns, databaseName)
}

func releaseIsNotFound(err error) bool {
	return strings.Contains(err.Error(), "not found")
}

func notFound(res string) bool {
	return strings.Contains(res, "not found")
}

func (oa *operatorActions) cloneOperatorRepo() error {
	cmd := fmt.Sprintf("git clone %s %s", oa.cfg.OperatorRepoUrl, oa.cfg.OperatorRepoDir)
	klog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil && !strings.Contains(string(res), "already exists") {
		return fmt.Errorf("failed to clone tidb-operator repository: %v, %s", err, string(res))
	}

	return nil
}

func (oa *operatorActions) checkoutTag(tagName string) error {
	cmd := fmt.Sprintf("cd %s && git stash -u && git checkout %s && "+
		"mkdir -p %s && cp -rf charts/tidb-operator %s && "+
		"cp -rf charts/tidb-cluster %s && cp -rf charts/tidb-backup %s &&"+
		"cp -rf manifests %s",
		oa.cfg.OperatorRepoDir, tagName,
		filepath.Join(oa.cfg.ChartDir, tagName), oa.operatorChartPath(tagName),
		oa.tidbClusterChartPath(tagName), oa.backupChartPath(tagName),
		oa.manifestPath(tagName))
	if tagName != "v1.0.0" {
		cmd = cmd + fmt.Sprintf(" && cp -rf charts/tidb-drainer %s", oa.drainerChartPath(tagName))
	}
	klog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check tag: %s, %v, %s", tagName, err, string(res))
	}

	return nil
}

func (oa *operatorActions) DeployAdHocBackup(info *TidbClusterConfig) error {
	oa.EmitEvent(info, "DeployAdHocBackup")
	klog.Infof("begin to deploy adhoc backup cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)

	var tsStr string
	getTSFn := func() (bool, error) {
		var mysqlHost string
		var mysqlPort uint16
		if oa.fw != nil {
			localHost, localPort, cancel, err := portforward.ForwardOnePort(oa.fw, info.Namespace, fmt.Sprintf("svc/%s-tidb", info.ClusterName), 4000)
			if err != nil {
				klog.Errorf("failed to forward port %d for %s/%s", 4000, info.Namespace, info.ClusterName)
				return false, nil
			}
			defer cancel()
			mysqlHost = localHost
			mysqlPort = localPort
		} else {
			mysqlHost = fmt.Sprintf("%s-tidb.%s", info.ClusterName, info.Namespace)
			mysqlPort = 4000
		}
		passwdStr := ""
		if info.Password != "" {
			passwdStr = fmt.Sprintf("-p%s", info.Password)
		}
		getTSCmd := fmt.Sprintf("set -euo pipefail; mysql -u%s %s -h%s -P %d -Nse 'show master status;' | awk '{print $2}'",
			info.UserName,
			passwdStr,
			mysqlHost,
			mysqlPort,
		)
		klog.Info(getTSCmd)

		res, err := exec.Command("/bin/bash", "-c", getTSCmd).CombinedOutput()
		if err != nil {
			klog.Errorf("failed to get ts %v, %s", err, string(res))
			return false, nil
		}
		tsStr = string(res)
		return true, nil
	}

	err := wait.Poll(DefaultPollInterval, BackupAndRestorePollTimeOut, getTSFn)
	if err != nil {
		return err
	}

	sets := map[string]string{
		"name":            info.BackupName,
		"mode":            "backup",
		"user":            "root",
		"password":        info.Password,
		"storage.size":    "10Gi",
		"initialCommitTs": strings.TrimSpace(tsStr),
	}

	setString := info.BackupHelmSetString(sets)

	fullbackupName := fmt.Sprintf("%s-backup", info.ClusterName)
	cmd := fmt.Sprintf("helm install -n %s --namespace %s %s --set-string %s",
		fullbackupName, info.Namespace, oa.backupChartPath(info.OperatorTag), setString)
	klog.Infof("install adhoc deployment [%s]", cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to launch adhoc backup job: %v, %s", err, string(res))
	}

	return nil
}

func (oa *operatorActions) CheckAdHocBackup(info *TidbClusterConfig) (string, error) {
	klog.Infof("checking adhoc backup cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)

	ns := info.Namespace
	var ts string
	jobName := fmt.Sprintf("%s-%s", info.ClusterName, info.BackupName)
	fn := func() (bool, error) {
		job, err := oa.kubeCli.BatchV1().Jobs(info.Namespace).Get(jobName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get jobs %s ,%v", jobName, err)
			return false, nil
		}
		if job.Status.Succeeded == 0 {
			klog.Errorf("cluster [%s] back up job is not completed, please wait! ", info.ClusterName)
			return false, nil
		}

		listOptions := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", label.InstanceLabelKey, jobName),
		}
		podList, err := oa.kubeCli.CoreV1().Pods(ns).List(listOptions)
		if err != nil {
			klog.Errorf("failed to list pods: %v", err)
			return false, nil
		}

		var podName string
		for _, pod := range podList.Items {
			ref := pod.OwnerReferences[0]
			if ref.Kind == "Job" && ref.Name == jobName {
				podName = pod.GetName()
				break
			}
		}
		if podName == "" {
			klog.Errorf("failed to find the ad-hoc backup: %s podName", jobName)
			return false, nil
		}

		getTsCmd := fmt.Sprintf("kubectl logs -n %s %s | grep 'commitTS = ' | cut -d '=' -f2 | sed 's/ *//g'", ns, podName)
		tsData, err := exec.Command("/bin/sh", "-c", getTsCmd).CombinedOutput()
		if err != nil {
			klog.Errorf("failed to get ts of pod %s, %v", podName, err)
			return false, nil
		}
		if string(tsData) == "" {
			klog.Errorf("ts is empty pod %s", podName)
			return false, nil
		}

		ts = strings.TrimSpace(string(tsData))
		klog.Infof("ad-hoc backup ts: %s", ts)

		return true, nil
	}

	err := wait.Poll(DefaultPollInterval, BackupAndRestorePollTimeOut, fn)
	if err != nil {
		return ts, fmt.Errorf("failed to launch backup job: %v", err)
	}

	return ts, nil
}

func (oa *operatorActions) Restore(from *TidbClusterConfig, to *TidbClusterConfig) error {
	oa.EmitEvent(from, fmt.Sprintf("RestoreBackup: target: %s", to.ClusterName))
	oa.EmitEvent(to, fmt.Sprintf("RestoreBackup: source: %s", from.ClusterName))
	klog.Infof("deploying restore, the data is from cluster[%s/%s] to cluster[%s/%s]",
		from.Namespace, from.ClusterName, to.Namespace, to.ClusterName)

	sets := map[string]string{
		"name":         to.BackupName,
		"mode":         "restore",
		"user":         "root",
		"password":     to.Password,
		"storage.size": "10Gi",
	}

	setString := to.BackupHelmSetString(sets)

	restoreName := fmt.Sprintf("%s-restore", to.ClusterName)
	cmd := fmt.Sprintf("helm install -n %s --namespace %s %s --set-string %s",
		restoreName, to.Namespace, oa.backupChartPath(to.OperatorTag), setString)
	klog.Infof("install restore [%s]", cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to launch restore job: %v, %s", err, string(res))
	}

	return nil
}

func (oa *operatorActions) CheckRestore(from *TidbClusterConfig, to *TidbClusterConfig) error {
	klog.Infof("begin to check restore backup cluster[%s] namespace[%s]", from.ClusterName, from.Namespace)
	jobName := fmt.Sprintf("%s-restore-%s", to.ClusterName, from.BackupName)
	fn := func() (bool, error) {
		job, err := oa.kubeCli.BatchV1().Jobs(to.Namespace).Get(jobName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get jobs %s ,%v", jobName, err)
			return false, nil
		}
		if job.Status.Succeeded == 0 {
			klog.Errorf("cluster [%s] restore job is not completed, please wait! ", to.ClusterName)
			return false, nil
		}

		_, err = oa.DataIsTheSameAs(to, from)
		if err != nil {
			// ad-hoc restore don't check the data really, just logging
			klog.Infof("check restore: %v", err)
		}

		return true, nil
	}

	err := wait.Poll(oa.pollInterval, BackupAndRestorePollTimeOut, fn)
	if err != nil {
		return fmt.Errorf("failed to launch restore job: %v", err)
	}
	return nil
}

func (oa *operatorActions) ForceDeploy(info *TidbClusterConfig) error {
	if err := oa.CleanTidbCluster(info); err != nil {
		return err
	}

	return oa.DeployTidbCluster(info)
}

func (oa *operatorActions) DataIsTheSameAs(tc, otherInfo *TidbClusterConfig) (bool, error) {
	tableNum := otherInfo.BlockWriteConfig.TableNum

	dsn, cancel, err := oa.getTiDBDSN(tc.Namespace, tc.ClusterName, "sbtest", tc.Password)
	if err != nil {
		return false, nil
	}
	defer cancel()
	infoDb, err := sql.Open("mysql", dsn)
	if err != nil {
		return false, err
	}
	defer infoDb.Close()
	otherDsn, otherCancel, err := oa.getTiDBDSN(otherInfo.Namespace, otherInfo.ClusterName, "sbtest", otherInfo.Password)
	if err != nil {
		return false, nil
	}
	defer otherCancel()
	otherInfoDb, err := sql.Open("mysql", otherDsn)
	if err != nil {
		return false, err
	}
	defer otherInfoDb.Close()

	getCntFn := func(db *sql.DB, tableName string) (int, error) {
		var cnt int
		row := db.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s", tableName))
		err := row.Scan(&cnt)
		if err != nil {
			return cnt, fmt.Errorf("failed to scan count from %s, %v", tableName, err)
		}
		return cnt, nil
	}

	for i := 0; i < tableNum; i++ {
		var tableName string
		if i == 0 {
			tableName = "block_writer"
		} else {
			tableName = fmt.Sprintf("block_writer%d", i)
		}

		cnt, err := getCntFn(infoDb, tableName)
		if err != nil {
			return false, err
		}
		otherCnt, err := getCntFn(otherInfoDb, tableName)
		if err != nil {
			return false, err
		}

		if cnt != otherCnt {
			err := fmt.Errorf("cluster %s/%s's table %s count(*) = %d and cluster %s/%s's table %s count(*) = %d",
				tc.Namespace, tc.ClusterName, tableName, cnt,
				otherInfo.Namespace, otherInfo.ClusterName, tableName, otherCnt)
			return false, err
		}
		klog.Infof("cluster %s/%s's table %s count(*) = %d and cluster %s/%s's table %s count(*) = %d",
			tc.Namespace, tc.ClusterName, tableName, cnt,
			otherInfo.Namespace, otherInfo.ClusterName, tableName, otherCnt)
	}

	return true, nil
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
	oa.EmitEvent(info, "DeploySchedulerBackup")
	klog.Infof("begin to deploy scheduled backup")

	cron := "'*/1 * * * *'"
	sets := map[string]string{
		"clusterName":                info.ClusterName,
		"scheduledBackup.create":     "true",
		"scheduledBackup.user":       "root",
		"scheduledBackup.password":   info.Password,
		"scheduledBackup.schedule":   cron,
		"scheduledBackup.storage":    "10Gi",
		"scheduledBackup.secretName": info.BackupSecretName,
	}

	cmd, err := oa.getHelmUpgradeClusterCmd(info, sets)
	if err != nil {
		return err
	}

	klog.Infof("scheduled-backup deploy [%s]", cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to launch scheduler backup job: %v, %s", err, string(res))
	}
	return nil
}

func (oa *operatorActions) disableScheduledBackup(info *TidbClusterConfig) error {
	klog.Infof("disabling scheduled backup")

	sets := map[string]string{
		"clusterName":            info.ClusterName,
		"scheduledBackup.create": "false",
	}

	cmd, err := oa.getHelmUpgradeClusterCmd(info, sets)
	if err != nil {
		return err
	}

	klog.Infof("scheduled-backup disable [%s]", cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to disable scheduler backup job: %v, %s", err, string(res))
	}
	return nil
}

func (oa *operatorActions) CheckScheduledBackup(info *TidbClusterConfig) error {
	klog.Infof("checking scheduler backup for tidb cluster[%s/%s]", info.Namespace, info.ClusterName)

	jobName := fmt.Sprintf("%s-scheduled-backup", info.ClusterName)
	fn := func() (bool, error) {
		job, err := oa.kubeCli.BatchV1beta1().CronJobs(info.Namespace).Get(jobName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get cronjobs %s ,%v", jobName, err)
			return false, nil
		}

		jobs, err := oa.kubeCli.BatchV1().Jobs(info.Namespace).List(metav1.ListOptions{})
		if err != nil {
			klog.Errorf("failed to list jobs %s ,%v", info.Namespace, err)
			return false, nil
		}

		backupJobs := []batchv1.Job{}
		for _, j := range jobs.Items {
			if pid, found := getParentUIDFromJob(j); found && pid == job.UID {
				backupJobs = append(backupJobs, j)
			}
		}

		if len(backupJobs) == 0 {
			klog.Errorf("cluster [%s] scheduler jobs is creating, please wait!", info.ClusterName)
			return false, nil
		}

		succededJobCount := 0
		for _, j := range backupJobs {
			if j.Status.Failed > 3 {
				return false, fmt.Errorf("cluster [%s/%s] scheduled backup job failed, job: [%s] failed count is: %d",
					info.Namespace, info.ClusterName, j.Name, j.Status.Failed)
			}
			if j.Status.Succeeded > 0 {
				succededJobCount++
			}
		}

		if succededJobCount >= 3 {
			klog.Infof("cluster [%s/%s] scheduled back up job completed count: %d",
				info.Namespace, info.ClusterName, succededJobCount)
			return true, nil
		}

		klog.Infof("cluster [%s/%s] scheduled back up job is not completed, please wait! ",
			info.Namespace, info.ClusterName)
		return false, nil
	}

	err := wait.Poll(DefaultPollInterval, BackupAndRestorePollTimeOut, fn)
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
		return fmt.Errorf("scheduler job failed")
	}

	return oa.disableScheduledBackup(info)
}

func getParentUIDFromJob(j batchv1.Job) (types.UID, bool) {
	controllerRef := metav1.GetControllerOf(&j)

	if controllerRef == nil {
		return types.UID(""), false
	}

	if controllerRef.Kind != "CronJob" {
		klog.Infof("Job with non-CronJob parent, name %s namespace %s", j.Name, j.Namespace)
		return types.UID(""), false
	}

	return controllerRef.UID, true
}

func (oa *operatorActions) getBackupDir(info *TidbClusterConfig) ([]string, error) {
	scheduledPvcName := fmt.Sprintf("%s-scheduled-backup", info.ClusterName)
	backupDirPodName := info.GenerateBackupDirPodName()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupDirPodName,
			Namespace: info.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    backupDirPodName,
					Image:   "pingcap/tidb-cloud-backup:20190610",
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
		_, err := oa.kubeCli.CoreV1().Pods(info.Namespace).Get(backupDirPodName, metav1.GetOptions{})
		if !errors.IsNotFound(err) {
			return false, nil
		}
		return true, nil
	}

	err := wait.Poll(oa.pollInterval, DefaultPollTimeout, fn)

	if err != nil {
		return nil, fmt.Errorf("failed to delete pod %s, err: %v", backupDirPodName, err)
	}

	_, err = oa.kubeCli.CoreV1().Pods(info.Namespace).Create(pod)
	if err != nil && !errors.IsAlreadyExists(err) {
		klog.Errorf("cluster: [%s/%s] create get backup dir pod failed, error :%v", info.Namespace, info.ClusterName, err)
		return nil, err
	}

	fn = func() (bool, error) {
		pod, err := oa.kubeCli.CoreV1().Pods(info.Namespace).Get(backupDirPodName, metav1.GetOptions{})
		if err == nil && pod.Status.Phase == corev1.PodRunning {
			return true, nil
		} else if err != nil && !errors.IsNotFound(err) {
			return false, err
		}
		return false, nil
	}

	err = wait.Poll(oa.pollInterval, DefaultPollTimeout, fn)

	if err != nil {
		return nil, fmt.Errorf("failed to create pod %s, err: %v", backupDirPodName, err)
	}

	cmd := fmt.Sprintf("kubectl exec %s -n %s ls /data", backupDirPodName, info.Namespace)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		klog.Errorf("cluster:[%s/%s] exec :%s failed,error:%v,result:%s", info.Namespace, info.ClusterName, cmd, err, string(res))
		return nil, err
	}

	dirs := strings.Split(string(res), "\n")
	klog.Infof("dirs in pod info name [%s] dir name [%s]", scheduledPvcName, strings.Join(dirs, ","))
	return dirs, nil
}

func (tc *TidbClusterConfig) FullName() string {
	return fmt.Sprintf("%s/%s", tc.Namespace, tc.ClusterName)
}

func (oa *operatorActions) DeployIncrementalBackup(from *TidbClusterConfig, to *TidbClusterConfig, withDrainer bool, ts string) error {

	if withDrainer && to == nil {
		return fmt.Errorf("Target cluster is nil when deploying drainer")
	}
	if withDrainer {
		oa.EmitEvent(from, fmt.Sprintf("DeployIncrementalBackup: secondary: %s", to.ClusterName))
		klog.Infof("begin to deploy incremental backup, source cluster[%s/%s], target cluster [%s/%s]",
			from.Namespace, from.ClusterName, to.Namespace, to.ClusterName)
	} else {
		oa.EmitEvent(from, "Enable pump cluster")
		klog.Infof("begin to enable pump for cluster[%s/%s]",
			from.Namespace, from.ClusterName)
	}

	// v1.0.0 don't support `binlog.drainer.config`
	// https://github.com/pingcap/tidb-operator/pull/693
	isv1 := from.OperatorTag == "v1.0.0"

	sets := map[string]string{
		"binlog.pump.create":  "true",
		"binlog.pump.storage": "1Gi",
		"binlog.pump.image":   fmt.Sprintf("pingcap/tidb-binlog:%v", from.ClusterVersion),
	}

	if withDrainer {
		sets["binlog.drainer.create"] = "true"
		sets["binlog.drainer.image"] = fmt.Sprintf("pingcap/tidb-binlog:%v", from.ClusterVersion)
		if isv1 {
			sets["binlog.pump.create"] = "true"
			sets["binlog.drainer.destDBType"] = "mysql"
			sets["binlog.drainer.mysql.host"] = fmt.Sprintf("%s-tidb.%s", to.ClusterName, to.Namespace)
			sets["binlog.drainer.mysql.user"] = "root"
			sets["binlog.drainer.mysql.password"] = to.Password
			sets["binlog.drainer.mysql.port"] = "4000"
			sets["binlog.drainer.ignoreSchemas"] = ""
		} else {
			from.drainerConfig = []string{
				`detect-interval = 10`,
				`compressor = ""`,
				`[syncer]`,
				`worker-count = 16`,
				`disable-dispatch = false`,
				`ignore-schemas = "INFORMATION_SCHEMA,PERFORMANCE_SCHEMA,mysql"`,
				`safe-mode = false`,
				`txn-batch = 20`,
				`db-type = "mysql"`,
				`[syncer.to]`,
				fmt.Sprintf(`host = "%s-tidb.%s"`, to.ClusterName, to.Namespace),
				fmt.Sprintf(`user = "%s"`, "root"),
				fmt.Sprintf(`password = "%s"`, to.Password),
				fmt.Sprintf(`port = %d`, 4000),
			}
		}
	}

	if ts != "" {
		sets["binlog.drainer.initialCommitTs"] = ts
	}

	cmd, err := oa.getHelmUpgradeClusterCmd(from, sets)
	if err != nil {
		return err
	}
	klog.Infof(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to launch incremental backup job: %v, %s", err, string(res))
	}
	return nil
}

func (oa *operatorActions) CheckIncrementalBackup(info *TidbClusterConfig, withDrainer bool) error {
	klog.Infof("begin to check incremental backup cluster[%s] namespace[%s]", info.ClusterName, info.Namespace)

	pumpStatefulSetName := fmt.Sprintf("%s-pump", info.ClusterName)
	fn := func() (bool, error) {
		pumpStatefulSet, err := oa.kubeCli.AppsV1().StatefulSets(info.Namespace).Get(pumpStatefulSetName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get jobs %s ,%v", pumpStatefulSetName, err)
			return false, nil
		}
		if pumpStatefulSet.Status.Replicas != pumpStatefulSet.Status.ReadyReplicas {
			klog.Errorf("pump replicas is not ready, please wait ! %s ", pumpStatefulSetName)
			return false, nil
		}

		listOps := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(
				map[string]string{
					label.ComponentLabelKey: "pump",
					label.InstanceLabelKey:  pumpStatefulSet.Labels[label.InstanceLabelKey],
					label.NameLabelKey:      "tidb-cluster",
				},
			).String(),
		}

		pods, err := oa.kubeCli.CoreV1().Pods(info.Namespace).List(listOps)
		if err != nil {
			klog.Errorf("failed to get pods via pump labels %s ,%v", pumpStatefulSetName, err)
			return false, nil
		}

		// v1.0.0 don't have affinity test case
		// https://github.com/pingcap/tidb-operator/pull/746
		isv1 := info.OperatorTag == "v1.0.0"

		for _, pod := range pods.Items {
			if !oa.pumpHealth(info.ClusterName, info.Namespace, pod.Name, false) {
				klog.Errorf("some pods is not health %s", pumpStatefulSetName)
				return false, nil
			}

			if isv1 {
				continue
			}

			klog.Info(pod.Spec.Affinity)
			if pod.Spec.Affinity == nil || pod.Spec.Affinity.PodAntiAffinity == nil || len(pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) != 1 {
				return true, fmt.Errorf("pump pod %s/%s should have affinity set", pod.Namespace, pod.Name)
			}
			klog.Info(pod.Spec.Tolerations)
			foundKey := false
			for _, tor := range pod.Spec.Tolerations {
				if tor.Key == "node-role" {
					foundKey = true
					break
				}
			}
			if !foundKey {
				return true, fmt.Errorf("pump pod %s/%s should have tolerations set", pod.Namespace, pod.Name)
			}
		}

		if !withDrainer {
			return true, nil
		}

		drainerStatefulSetName := fmt.Sprintf("%s-drainer", info.ClusterName)
		drainerStatefulSet, err := oa.kubeCli.AppsV1().StatefulSets(info.Namespace).Get(drainerStatefulSetName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get jobs %s ,%v", pumpStatefulSetName, err)
			return false, nil
		}
		if drainerStatefulSet.Status.Replicas != drainerStatefulSet.Status.ReadyReplicas {
			klog.Errorf("drainer replicas is not ready, please wait ! %s ", pumpStatefulSetName)
			return false, nil
		}

		listOps = metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(
				map[string]string{
					label.ComponentLabelKey: "drainer",
					label.InstanceLabelKey:  drainerStatefulSet.Labels[label.InstanceLabelKey],
					label.NameLabelKey:      "tidb-cluster",
				},
			).String(),
		}

		pods, err = oa.kubeCli.CoreV1().Pods(info.Namespace).List(listOps)
		if err != nil {
			return false, nil
		}
		for _, pod := range pods.Items {
			if !oa.drainerHealth(info.ClusterName, info.Namespace, pod.Name, false) {
				klog.Errorf("some pods is not health %s", drainerStatefulSetName)
				return false, nil
			}

			if isv1 {
				continue
			}

			klog.Info(pod.Spec.Affinity)
			if pod.Spec.Affinity == nil || pod.Spec.Affinity.PodAntiAffinity == nil || len(pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) != 1 {
				return true, fmt.Errorf("drainer pod %s/%s should have spec.affinity set", pod.Namespace, pod.Name)
			}
			klog.Info(pod.Spec.Tolerations)
			foundKey := false
			for _, tor := range pod.Spec.Tolerations {
				if tor.Key == "node-role" {
					foundKey = true
					break
				}
			}
			if !foundKey {
				return true, fmt.Errorf("drainer pod %s/%s should have tolerations set", pod.Namespace, pod.Name)
			}
		}

		return true, nil
	}

	err := wait.Poll(oa.pollInterval, DefaultPollTimeout, fn)
	if err != nil {
		return fmt.Errorf("failed to check incremental backup job: %v", err)
	}
	return nil

}

func strPtr(s string) *string { return &s }

func (oa *operatorActions) RegisterWebHookAndServiceOrDie(configName, namespace, service string, context *apimachinery.CertContext) {
	if err := oa.RegisterWebHookAndService(configName, namespace, service, context); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) RegisterWebHookAndService(configName, namespace, service string, context *apimachinery.CertContext) error {
	client := oa.kubeCli
	klog.Infof("Registering the webhook via the AdmissionRegistration API")

	failurePolicy := admissionV1beta1.Fail

	_, err := client.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(&admissionV1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: configName,
		},
		Webhooks: []admissionV1beta1.ValidatingWebhook{
			{
				Name:          "check-pod-before-delete.k8s.io",
				FailurePolicy: &failurePolicy,
				Rules: []admissionV1beta1.RuleWithOperations{{
					Operations: []admissionV1beta1.OperationType{admissionV1beta1.Delete},
					Rule: admissionV1beta1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods"},
					},
				}},
				ClientConfig: admissionV1beta1.WebhookClientConfig{
					Service: &admissionV1beta1.ServiceReference{
						Namespace: namespace,
						Name:      service,
						Path:      strPtr("/pods"),
					},
					CABundle: context.SigningCert,
				},
			},
		},
	})

	if err != nil {
		klog.Errorf("registering webhook config %s with namespace %s error %v", configName, namespace, err)
		return err
	}

	// The webhook configuration is honored in 10s.
	time.Sleep(10 * time.Second)

	return nil

}

func (oa *operatorActions) CleanWebHookAndService(name string) error {
	err := oa.kubeCli.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Delete(name, nil)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete webhook config %v", err)
	}
	return nil
}

func (oa *operatorActions) CleanWebHookAndServiceOrDie(name string) {
	err := oa.CleanWebHookAndService(name)
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

type pumpStatus struct {
	StatusMap map[string]*nodeStatus `json:"StatusMap"`
}

type nodeStatus struct {
	State string `json:"state"`
}

func (oa *operatorActions) pumpHealth(tcName, ns, podName string, tlsEnabled bool) bool {
	var err error
	var addr string
	if oa.fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(oa.fw, ns, fmt.Sprintf("pod/%s", podName), 8250)
		if err != nil {
			klog.Errorf("failed to forward port %d for %s/%s", 8250, ns, podName)
			return false
		}
		defer cancel()
		addr = fmt.Sprintf("%s:%d", localHost, localPort)
	} else {
		addr = fmt.Sprintf("%s.%s-pump.%s:8250", podName, tcName, ns)
	}
	var tlsConfig *tls.Config
	scheme := "http"
	if tlsEnabled {
		tlsConfig, err = pdapi.GetTLSConfig(oa.kubeCli, pdapi.Namespace(ns), tcName, util.ClusterTLSSecretName(tcName, label.PumpLabelVal))
		if err != nil {
			return false
		}
		scheme = "https"
	}
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: tlsConfig},
	}
	pumpHealthURL := fmt.Sprintf("%s://%s/status", scheme, addr)
	res, err := client.Get(pumpHealthURL)
	if err != nil {
		klog.Errorf("cluster:[%s] call %s failed,error:%v", tcName, pumpHealthURL, err)
		return false
	}
	if res.StatusCode >= 400 {
		klog.Errorf("Error response %v", res.StatusCode)
		return false
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		klog.Errorf("cluster:[%s] read response body failed,error:%v", tcName, err)
		return false
	}
	healths := pumpStatus{}
	err = json.Unmarshal(body, &healths)
	if err != nil {
		klog.Errorf("cluster:[%s] unmarshal failed,error:%v", tcName, err)
		return false
	}
	for _, status := range healths.StatusMap {
		if status.State != "online" {
			klog.Errorf("cluster:[%s] pump's state is not online", tcName)
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

func (oa *operatorActions) drainerHealth(tcName, ns, podName string, tlsEnabled bool) bool {
	var body []byte
	var err error
	var addr string
	if oa.fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(oa.fw, ns, fmt.Sprintf("pod/%s", podName), 8249)
		if err != nil {
			klog.Errorf("failed to forward port %d for %s/%s", 8249, ns, podName)
			return false
		}
		defer cancel()
		addr = fmt.Sprintf("%s:%d", localHost, localPort)
	} else {
		addr = fmt.Sprintf("%s.%s-drainer.%s:8249", podName, tcName, ns)
	}
	var tlsConfig *tls.Config
	scheme := "http"
	if tlsEnabled {
		tlsConfig, err = pdapi.GetTLSConfig(oa.kubeCli, pdapi.Namespace(ns), tcName, util.ClusterTLSSecretName(tcName, "drainer"))
		if err != nil {
			return false
		}
		scheme = "https"
	}
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: tlsConfig},
	}
	drainerHealthURL := fmt.Sprintf("%s://%s/status", scheme, addr)
	res, err := client.Get(drainerHealthURL)
	if err != nil {
		klog.Errorf("cluster:[%s] call %s failed,error:%v", tcName, drainerHealthURL, err)
		return false
	}
	if res.StatusCode >= 400 {
		klog.Errorf("Error response %v", res.StatusCode)
		return false
	}
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		klog.Errorf("cluster:[%s] read response body failed,error:%v", tcName, err)
		return false
	}
	healths := drainerStatus{}
	err = json.Unmarshal(body, &healths)
	if err != nil {
		klog.Errorf("cluster:[%s] unmarshal failed,error:%v", tcName, err)
		return false
	}
	return len(healths.PumpPos) > 0
}

func (oa *operatorActions) EmitEvent(info *TidbClusterConfig, message string) {
	oa.lock.Lock()
	defer oa.lock.Unlock()

	klog.Infof("Event: %s", message)

	if !oa.eventWorkerRunning {
		return
	}

	if len(oa.clusterEvents) == 0 {
		return
	}

	ev := event{
		message: message,
		ts:      time.Now().UnixNano() / int64(time.Millisecond),
	}

	if info == nil {
		for k := range oa.clusterEvents {
			ce := oa.clusterEvents[k]
			ce.events = append(ce.events, ev)
		}
		return
	}

	ce, ok := oa.clusterEvents[info.String()]
	if !ok {
		return
	}
	ce.events = append(ce.events, ev)

	// sleep a while to avoid overlapping time
	time.Sleep(10 * time.Second)
}

func (oa *operatorActions) RunEventWorker() {
	oa.lock.Lock()
	oa.eventWorkerRunning = true
	oa.lock.Unlock()
	klog.Infof("Event worker started")
	wait.Forever(oa.eventWorker, 10*time.Second)
}

func (oa *operatorActions) eventWorker() {
	oa.lock.Lock()
	defer oa.lock.Unlock()

	for key, clusterEv := range oa.clusterEvents {
		retryEvents := make([]event, 0)
		for _, ev := range clusterEv.events {
			ns := clusterEv.ns
			clusterName := clusterEv.clusterName
			grafanaURL := fmt.Sprintf("http://%s-grafana.%s:3000", clusterName, ns)
			client, err := metrics.NewClient(grafanaURL, grafanaUsername, grafanaPassword)
			if err != nil {
				// If parse grafana URL failed, this error cannot be recovered by retrying, so send error msg and panic
				slack.NotifyAndPanic(fmt.Errorf("failed to parse grafana URL so can't new grafana client: %s, %v", grafanaURL, err))
			}

			anno := metrics.Annotation{
				Text:                ev.message,
				TimestampInMilliSec: ev.ts,
				Tags: []string{
					statbilityTestTag,
					fmt.Sprintf("clusterName: %s", clusterName),
					fmt.Sprintf("namespace: %s", ns),
				},
			}
			if err := client.AddAnnotation(anno); err != nil {
				klog.V(4).Infof("cluster:[%s/%s] error recording event: %s, reason: %v",
					ns, clusterName, ev.message, err)
				retryEvents = append(retryEvents, ev)
				continue
			}
			klog.Infof("cluster: [%s/%s] recoding event: %s", ns, clusterName, ev.message)
		}

		ce := oa.clusterEvents[key]
		ce.events = retryEvents
	}
}

func (oa *operatorActions) getHelmUpgradeClusterCmd(info *TidbClusterConfig, set map[string]string) (string, error) {
	cmd := fmt.Sprintf("helm upgrade %s %s --set-string %s",
		info.ClusterName, oa.tidbClusterChartPath(info.OperatorTag), info.TidbClusterHelmSetString(set))
	svFilePath, err := info.BuildSubValues(oa.tidbClusterChartPath(info.OperatorTag))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(" %s --values %s", cmd, svFilePath), nil
}

func (oa *operatorActions) checkManualPauseComponent(info *TidbClusterConfig, component string) error {

	var tc *v1alpha1.TidbCluster
	var setName string
	var set *v1.StatefulSet
	var err error
	ns := info.Namespace

	fn := func() (bool, error) {

		if tc, err = oa.cli.PingcapV1alpha1().TidbClusters(ns).Get(info.ClusterName, metav1.GetOptions{}); err != nil {
			klog.Infof("failed to get tidbcluster: [%s/%s], %v", ns, info.ClusterName, err)
			return false, nil
		}

		switch component {
		case label.TiDBLabelVal:
			podName := fmt.Sprintf("%s-%d", controller.TiDBMemberName(tc.Name), 1)
			setName = controller.TiDBMemberName(info.ClusterName)
			tidbPod, err := oa.kubeCli.CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
			if err != nil {
				klog.Infof("fail to get pod in CheckManualPauseCompoent tidb [%s/%s]", ns, podName)
				return false, nil
			}

			if tidbPod.Labels[v1.ControllerRevisionHashLabelKey] == tc.Status.TiDB.StatefulSet.UpdateRevision &&
				tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
				if member, ok := tc.Status.TiDB.Members[tidbPod.Name]; !ok || !member.Health {
					klog.Infof("wait for tidb pod [%s/%s] ready member health %t ok %t", ns, podName, member.Health, ok)
				} else {
					return true, nil
				}
			} else {
				klog.Infof("tidbset is not in upgrade phase or pod is not upgrade done [%s/%s]", ns, podName)
			}

			return false, nil
		case label.TiKVLabelVal:
			podName := fmt.Sprintf("%s-%d", controller.TiKVMemberName(tc.Name), 1)
			setName = controller.TiKVMemberName(info.ClusterName)
			tikvPod, err := oa.kubeCli.CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
			if err != nil {
				klog.Infof("fail to get pod in CheckManualPauseCompoent tikv [%s/%s]", ns, podName)
				return false, nil
			}

			if tikvPod.Labels[v1.ControllerRevisionHashLabelKey] == tc.Status.TiKV.StatefulSet.UpdateRevision &&
				tc.Status.TiKV.Phase == v1alpha1.UpgradePhase {
				var tikvStore *v1alpha1.TiKVStore
				for _, store := range tc.Status.TiKV.Stores {
					if store.PodName == podName {
						tikvStore = &store
						break
					}
				}
				if tikvStore == nil || tikvStore.State != v1alpha1.TiKVStateUp {
					klog.Infof("wait for tikv pod [%s/%s] ready store state %s", ns, podName, tikvStore.State)
				} else {
					return true, nil
				}
			} else {
				klog.Infof("tikvset is not in upgrade phase or pod is not upgrade done [%s/%s]", ns, podName)
			}

			return false, nil
		default:
			return false, fmt.Errorf("invalid component %s", component)
		}
	}

	// wait for the tidb or tikv statefulset is upgraded to the protected one
	if err = wait.Poll(DefaultPollInterval, 30*time.Minute, fn); err != nil {
		return fmt.Errorf("fail to upgrade to annotation %s pod, err: %v", component, err)
	}

	time.Sleep(30 * time.Second)

	if set, err = oa.tcStsGetter.StatefulSets(ns).Get(setName, metav1.GetOptions{}); err != nil {
		return fmt.Errorf("failed to get statefulset: [%s/%s], %v", ns, setName, err)
	}

	if *set.Spec.UpdateStrategy.RollingUpdate.Partition < 1 {
		return fmt.Errorf("pause partition is not correct in upgrade phase [%s/%s] partition %d annotation %d",
			ns, setName, *set.Spec.UpdateStrategy.RollingUpdate.Partition, 1)
	}

	return nil
}

func (oa *operatorActions) CheckUpgradeComplete(info *TidbClusterConfig) error {
	ns, tcName := info.Namespace, info.ClusterName
	if err := wait.PollImmediate(15*time.Second, 30*time.Minute, func() (done bool, err error) {
		tc, err := oa.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("checkUpgradeComplete, [%s/%s] cannot get tidbcluster, %v", ns, tcName, err)
			return false, nil
		}
		if tc.Status.PD.Phase == v1alpha1.UpgradePhase {
			klog.Errorf("checkUpgradeComplete, [%s/%s] PD is still upgrading", ns, tcName)
			return false, nil
		}
		if tc.Status.TiKV.Phase == v1alpha1.UpgradePhase {
			klog.Errorf("checkUpgradeComplete, [%s/%s] TiKV is still upgrading", ns, tcName)
			return false, nil
		}
		if tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
			klog.Errorf("checkUpgradeComplete, [%s/%s] TiDB is still upgrading", ns, tcName)
			return false, nil
		}
		return true, nil
	}); err != nil {
		klog.Errorf("failed to wait upgrade complete [%s/%s], %v", ns, tcName, err)
		return err
	}
	return nil
}

func (oa *operatorActions) CheckUpgradeCompleteOrDie(info *TidbClusterConfig) {
	if err := oa.CheckUpgradeComplete(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CheckInitSQL(info *TidbClusterConfig) error {
	ns, tcName := info.Namespace, info.ClusterName
	if err := wait.PollImmediate(10*time.Second, DefaultPollTimeout, func() (done bool, err error) {
		dsn, cancel, err := oa.getTiDBDSN(ns, tcName, "e2e", info.Password)
		if err != nil {
			return false, nil
		}
		defer cancel()
		infoDb, err := sql.Open("mysql", dsn)
		if err != nil {
			return false, nil
		}
		infoDb.Close()

		return true, nil
	}); err != nil {
		klog.Errorf("failed to check init sql complete [%s/%s], %v", ns, tcName, err)
		return err
	}
	return nil
}

func (oa *operatorActions) CheckInitSQLOrDie(info *TidbClusterConfig) {
	if err := oa.CheckInitSQL(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) pumpMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	ssName := controller.PumpMemberName(tcName)

	ss, err := oa.tcStsGetter.StatefulSets(ns).Get(ssName, metav1.GetOptions{})
	if err != nil {
		klog.Infof("failed to get statefulset: %s/%s, %v", ns, ssName, err)
		return false, nil
	}

	if ss.Status.CurrentRevision != ss.Status.UpdateRevision {
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), ss) {
		return false, nil
	}

	// check all pump replicas are online
	for i := 0; i < int(*ss.Spec.Replicas); i++ {
		podName := fmt.Sprintf("%s-%d", ssName, i)
		if !oa.pumpHealth(tc.Name, tc.Namespace, podName, tc.IsTLSClusterEnabled()) {
			klog.Infof("%s is not health yet", podName)
			return false, nil
		}
	}
	return true, nil
}

func (oa *operatorActions) WaitForTidbClusterReady(tc *v1alpha1.TidbCluster, timeout, pollInterval time.Duration) error {
	if tc == nil {
		return fmt.Errorf("tidbcluster is nil, cannot call WaitForTidbClusterReady")
	}
	return wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		var local *v1alpha1.TidbCluster
		var err error
		if local, err = oa.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{}); err != nil {
			klog.Errorf("failed to get tidbcluster: %s/%s, %v", tc.Namespace, tc.Name, err)
			return false, nil
		}

		if b, err := oa.pdMembersReadyFn(local); !b && err == nil {
			return false, nil
		}
		if b, err := oa.tikvMembersReadyFn(local); !b && err == nil {
			return false, nil
		}
		if b, err := oa.tidbMembersReadyFn(local); !b && err == nil {
			return false, nil
		}
		if tc.Spec.TiFlash != nil {
			if b, err := oa.tiflashMembersReadyFn(local); !b && err == nil {
				klog.Errorf("tiflash  members not ready: %s/%s, %v", tc.Namespace, tc.Name, err)
				return false, nil
			}
			klog.Infof("tiflash  members ready: %s/%s, %v", tc.Namespace, tc.Name, err)
		}
		if tc.Spec.Pump != nil {
			if b, err := oa.pumpMembersReadyFn(local); !b && err == nil {
				return false, nil
			}
		}
		return true, nil
	})
}

var dummyCancel = func() {}

func (oa *operatorActions) getPDClient(tc *v1alpha1.TidbCluster) (pdapi.PDClient, context.CancelFunc, error) {
	if oa.fw != nil {
		return proxiedpdclient.NewProxiedPDClientFromTidbCluster(oa.kubeCli, oa.fw, tc)
	}
	return controller.GetPDClient(oa.pdControl, tc), dummyCancel, nil
}

func (oa *operatorActions) getTiDBDSN(ns, tcName, databaseName, password string) (string, context.CancelFunc, error) {
	if oa.fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(oa.fw, ns, fmt.Sprintf("svc/%s", controller.TiDBMemberName(tcName)), 4000)
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("root:%s@(%s:%d)/%s?charset=utf8", password, localHost, localPort, databaseName), cancel, nil
	}
	return fmt.Sprintf("root:%s@(%s-tidb.%s:4000)/%s?charset=utf8", password, tcName, ns, databaseName), dummyCancel, nil
}

func StartValidatingAdmissionWebhookServerOrDie(context *apimachinery.CertContext, namespaces ...string) {
	sCert, err := tls.X509KeyPair(context.Cert, context.Key)
	if err != nil {
		panic(err)
	}

	versionCli, kubeCli, _, _, _ := client.NewCliOrDie()
	wh := webhook.NewWebhook(kubeCli, versionCli, namespaces)
	http.HandleFunc("/pods", wh.ServePods)
	server := &http.Server{
		Addr: ":443",
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{sCert},
		},
	}
	if err := server.ListenAndServeTLS("", ""); err != nil {
		sendErr := slack.SendErrMsg(err.Error())
		if sendErr != nil {
			klog.Error(sendErr)
		}
		panic(fmt.Sprintf("failed to start webhook server %v", err))
	}
}

func blockWriterPodName(info *TidbClusterConfig) string {
	return fmt.Sprintf("%s-blockwriter", info.ClusterName)
}
