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
	"os"
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
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	utildiscovery "github.com/pingcap/tidb-operator/pkg/util/discovery"
	e2eutil "github.com/pingcap/tidb-operator/tests/e2e/util"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/e2e/util/proxiedpdclient"
	"github.com/pingcap/tidb-operator/tests/e2e/util/proxiedtidbclient"
	utilstatefulset "github.com/pingcap/tidb-operator/tests/e2e/util/statefulset"
	"github.com/pingcap/tidb-operator/tests/pkg/apimachinery"
	"github.com/pingcap/tidb-operator/tests/pkg/blockwriter"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	"github.com/pingcap/tidb-operator/tests/pkg/metrics"
	"github.com/pingcap/tidb-operator/tests/pkg/webhook"
	"github.com/pingcap/tidb-operator/tests/slack"
	admissionV1beta1 "k8s.io/api/admissionregistration/v1beta1"
	apps "k8s.io/api/apps/v1"
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
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	aggregatorclientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/log"
	"k8s.io/kubernetes/test/e2e/framework/pod"
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
	fw portforward.PortForward, f *framework.Framework) *OperatorActions {

	var tcStsGetter typedappsv1.StatefulSetsGetter
	if operatorConfig != nil && operatorConfig.Enabled(features.AdvancedStatefulSet) {
		tcStsGetter = helper.NewHijackClient(kubeCli, asCli).AppsV1()
	} else {
		tcStsGetter = kubeCli.AppsV1()
	}
	secretLister := GetSecretListerWithCacheSynced(kubeCli, 1*time.Second)

	oa := &OperatorActions{
		framework:    f,
		cli:          cli,
		kubeCli:      kubeCli,
		pdControl:    pdapi.NewDefaultPDControl(secretLister),
		asCli:        asCli,
		aggrCli:      aggrCli,
		apiExtCli:    apiExtCli,
		tcStsGetter:  tcStsGetter,
		pollInterval: pollInterval,
		cfg:          cfg,
		fw:           fw,
		crdUtil:      NewCrdTestUtil(cli, kubeCli, asCli, tcStsGetter),
		secretLister: secretLister,
	}
	if fw != nil {
		kubeCfg, err := framework.LoadConfig()
		framework.ExpectNoError(err, "failed to load config")
		oa.tidbControl = proxiedtidbclient.NewProxiedTiDBClient(fw, kubeCfg.TLSClientConfig.CAData)
	} else {
		oa.tidbControl = controller.NewDefaultTiDBControl(secretLister)
	}
	oa.clusterEvents = make(map[string]*clusterEvent)
	return oa
}

const (
	DefaultPollTimeout          time.Duration = 5 * time.Minute
	DefaultPollInterval         time.Duration = 5 * time.Second
	BackupAndRestorePollTimeOut time.Duration = 10 * time.Minute
	grafanaUsername                           = "admin"
	grafanaPassword                           = "admin"
	operartorChartName                        = "tidb-operator"
	tidbClusterChartName                      = "tidb-cluster"
	backupChartName                           = "tidb-backup"
	drainerChartName                          = "tidb-drainer"
	statbilityTestTag                         = "stability"
)

type OperatorActions struct {
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
	secretLister       corelisterv1.SecretLister
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
	StsWebhookEnabled         bool
	DefaultingEnabled         bool
	ValidatingEnabled         bool
	Cabundle                  string
	BackupImage               string
	AutoFailover              *bool
	AppendReleaseSuffix       bool
	// Additional STRING values, set via --set-string flag.
	StringValues map[string]string
	Selector     []string
}
type BlockWriterConfig struct {
	Namespace        string
	ClusterName      string
	OperatorTag      string
	Password         string
	blockWriterPod   *corev1.Pod
	BlockWriteConfig blockwriter.Config
}

func (bw *BlockWriterConfig) String() string {
	return fmt.Sprintf("%s/%s", bw.Namespace, bw.ClusterName)
}

type SourceTidbClusterConfig struct {
	Namespace      string
	ClusterName    string
	OperatorTag    string
	ClusterVersion string
}

func (oi *OperatorConfig) OperatorHelmSetBoolean() string {
	set := map[string]bool{
		"admissionWebhook.create":                      oi.WebhookEnabled,
		"admissionWebhook.validation.statefulSets":     oi.StsWebhookEnabled,
		"admissionWebhook.mutation.pingcapResources":   oi.DefaultingEnabled,
		"admissionWebhook.validation.pingcapResources": oi.ValidatingEnabled,
		"appendReleaseSuffix":                          oi.AppendReleaseSuffix,
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
	if len(oi.Selector) > 0 {
		selector := fmt.Sprintln(strings.Join(oi.Selector, ","))
		set["controllerManager.selector"] = selector
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

func (oa *OperatorActions) runKubectlOrDie(args ...string) string {
	cmd := "kubectl"
	log.Logf("Running '%s %s'", cmd, strings.Join(args, " "))
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		log.Failf("Failed to run '%s %s'\nCombined output: %q\nError: %v", cmd, strings.Join(args, " "), string(out), err)
	}
	log.Logf("Combined output: %q", string(out))
	return string(out)
}

func (oa *OperatorActions) CleanCRDOrDie() {
	crdList, err := oa.apiExtCli.ApiextensionsV1beta1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	framework.ExpectNoError(err, "failed to list CRD")
	for _, crd := range crdList.Items {
		if !strings.HasSuffix(crd.Name, ".pingcap.com") {
			framework.Logf("CRD %q ignored", crd.Name)
			continue
		}
		framework.Logf("Deleting CRD %q", crd.Name)
		err = oa.apiExtCli.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(context.TODO(), crd.Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err, "failed to delete CRD %q", crd.Name)
		// Even if DELETE API request succeeds, the CRD object may still exists
		// in ap server. We should wait for it to be gone.
		err = e2eutil.WaitForCRDNotFound(oa.apiExtCli, crd.Name)
		framework.ExpectNoError(err, "failed to wait for CRD %q deleted", crd.Name)
	}
}

func (oa *OperatorActions) CreateOrReplaceCRD(isCRDV1Supported bool) {
	files := map[string]struct{}{}
	if isCRDV1Supported {
		crdList, err := oa.apiExtCli.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
		framework.ExpectNoError(err, "failed to list CRD")
		for _, crd := range crdList.Items {
			if !strings.HasSuffix(crd.Name, ".pingcap.com") {
				framework.Logf("CRD %q ignored", crd.Name)
				continue
			}
			files[crd.Spec.Group+"_"+crd.Spec.Names.Plural+".yaml"] = struct{}{}
		}
		oa.createOrReplaceCRD("v1", files)
	} else {
		crdList, err := oa.apiExtCli.ApiextensionsV1beta1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
		framework.ExpectNoError(err, "failed to list CRD")
		for _, crd := range crdList.Items {
			if !strings.HasSuffix(crd.Name, ".pingcap.com") {
				framework.Logf("CRD %q ignored", crd.Name)
				continue
			}
			files[crd.Spec.Group+"_"+crd.Spec.Names.Plural+".yaml"] = struct{}{}
		}
		oa.createOrReplaceCRD("v1beta1", files)
	}
}

func (oa *OperatorActions) createOrReplaceCRD(version string, files map[string]struct{}) {
	filepath.Walk(oa.manifestPath(filepath.Join("e2e/crd", version)), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if _, ok := files[info.Name()]; ok {
			oa.runKubectlOrDie("replace", "-f", path)
		} else {
			oa.runKubectlOrDie("create", "-f", path)
		}
		return nil
	})
}

// InstallCRDOrDie install CRDs and wait for them to be established in Kubernetes.
func (oa *OperatorActions) InstallCRDOrDie(info *OperatorConfig) {
	isSupported, err := utildiscovery.IsAPIGroupVersionSupported(oa.kubeCli.Discovery(), "apiextensions.k8s.io/v1")
	if err != nil {
		log.Failf(err.Error())
	}
	if info.Enabled(features.AdvancedStatefulSet) {
		if isSupported {
			oa.runKubectlOrDie("apply", "-f", oa.manifestPath("e2e/advanced-statefulset-crd.v1.yaml"))
		} else {
			oa.runKubectlOrDie("apply", "-f", oa.manifestPath("e2e/advanced-statefulset-crd.v1beta1.yaml"))
		}
	}
	// replace crd to avoid problem of too big annotation
	oa.CreateOrReplaceCRD(isSupported)
	oa.runKubectlOrDie("apply", "-f", oa.manifestPath("e2e/data-resource-crd.yaml"))
	log.Logf("Wait for all CRDs are established")
	e2eutil.WaitForCRDsEstablished(oa.apiExtCli, labels.Everything())
	// workaround for https://github.com/kubernetes/kubernetes/issues/65517
	log.Logf("force sync kubectl cache")
	cmdArgs := []string{"sh", "-c", "rm -rf ~/.kube/cache ~/.kube/http-cache"}
	if _, err := exec.Command(cmdArgs[0], cmdArgs[1:]...).CombinedOutput(); err != nil {
		log.Failf("Failed to run '%s': %v", strings.Join(cmdArgs, " "), err)
	}
}

func (oa *OperatorActions) DeployReleasedCRDOrDie(version string) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/pingcap/tidb-operator/%s/manifests/crd.yaml", version)
	err := wait.PollImmediate(time.Second*10, time.Minute, func() (bool, error) {
		_, err := framework.RunKubectl("", "apply", "-f", url)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	framework.ExpectNoError(err, "failed to apply CRD of version %s", version)
	log.Logf("Wait for all CRDs are established")
	e2eutil.WaitForCRDsEstablished(oa.apiExtCli, labels.Everything())
	// workaround for https://github.com/kubernetes/kubernetes/issues/65517
	log.Logf("force sync kubectl cache")
	cmdArgs := []string{"sh", "-c", "rm -rf ~/.kube/cache ~/.kube/http-cache"}
	_, err = exec.Command(cmdArgs[0], cmdArgs[1:]...).CombinedOutput()
	if err != nil {
		log.Failf("Failed to run '%s': %v", strings.Join(cmdArgs, " "), err)
	}
}

func (oa *OperatorActions) DeployOperator(info *OperatorConfig) error {
	log.Logf("deploying tidb-operator %s", info.ReleaseName)

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

	cmd := fmt.Sprintf(`helm install %s %s --namespace %s %s --set-string %s`,
		info.ReleaseName,
		oa.operatorChartPath(info.Tag),
		info.Namespace,
		info.OperatorHelmSetBoolean(),
		info.OperatorHelmSetString(nil))
	log.Logf(cmd)

	exec.Command("/bin/sh", "-c", fmt.Sprintf("kubectl create namespace %s", info.Namespace)).Run()
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to deploy operator: %v, %s", err, string(res))
	}
	log.Logf("deploy operator response: %v\n", string(res))

	log.Logf("Wait for all apiesrvices are available")
	return e2eutil.WaitForAPIServicesAvaiable(oa.aggrCli, labels.Everything())
}

func (oa *OperatorActions) DeployOperatorOrDie(info *OperatorConfig) {
	if err := oa.DeployOperator(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func CleanOperator(info *OperatorConfig) error {
	log.Logf("cleaning tidb-operator %s", info.ReleaseName)

	cmd := fmt.Sprintf("helm uninstall %s --namespace %s",
		info.ReleaseName,
		info.Namespace)
	log.Logf(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()

	if err == nil || !releaseIsNotFound(err) {
		return nil
	}

	return fmt.Errorf("failed to clear operator: %v, %s", err, string(res))
}

func (oa *OperatorActions) CleanOperatorOrDie(info *OperatorConfig) {
	if err := CleanOperator(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *OperatorActions) UpgradeOperator(info *OperatorConfig) error {
	log.Logf("upgrading tidb-operator %s", info.ReleaseName)

	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Labels()).String(),
	}
	pods1, err := oa.kubeCli.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), listOptions)
	if err != nil {
		log.Logf("failed to get pods in all namespaces with selector: %+v", listOptions)
		return err
	}

	if info.Tag != "e2e" {
		if err := oa.checkoutTag(info.Tag); err != nil {
			log.Logf("failed to checkout tag: %s", info.Tag)
			return err
		}
	}

	cmd := fmt.Sprintf("helm upgrade %s %s --namespace %s %s --set-string %s --wait",
		info.ReleaseName,
		oa.operatorChartPath(info.Tag),
		info.Namespace,
		info.OperatorHelmSetBoolean(),
		info.OperatorHelmSetString(nil))
	log.Logf("running helm upgrade command: %s", cmd)

	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to upgrade operator to: %s, %v, %s", info.Image, err, string(res))
	}

	log.Logf("Wait for all apiesrvices are available")
	err = e2eutil.WaitForAPIServicesAvaiable(oa.aggrCli, labels.Everything())
	if err != nil {
		return err
	}

	if info.Tag == "e2e" {
		return nil
	}

	// ensure pods unchanged when upgrading operator
	waitFn := func() (done bool, err error) {
		pods2, err := oa.kubeCli.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), listOptions)
		if err != nil {
			log.Logf("ERROR: %v", err)
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

func (oa *OperatorActions) DeployDMMySQLOrDie(ns string) {
	if err := DeployDMMySQL(oa.kubeCli, ns); err != nil {
		slack.NotifyAndPanic(err)
	}

	if err := CheckDMMySQLReady(oa.fw, ns); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *OperatorActions) DeployDMTiDBOrDie() {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: DMTiDBNamespace,
		},
	}
	_, err := oa.kubeCli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		slack.NotifyAndPanic(err)
	}

	tc := fixture.GetTidbCluster(DMTiDBNamespace, DMTiDBName, utilimage.TiDBLatest)
	tc.Spec.PD.Replicas = 1
	tc.Spec.TiKV.Replicas = 1
	tc.Spec.TiDB.Replicas = 1
	if _, err := oa.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(context.TODO(), tc, metav1.CreateOptions{}); err != nil {
		slack.NotifyAndPanic(err)
	}

	if err := oa.WaitForTidbClusterReady(tc, 30*time.Minute, 30*time.Second); err != nil {
		slack.NotifyAndPanic(err)
	}
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
		log.Logf("%s", string(pods1Yaml))
		log.Logf("%s", string(pods2Yaml))
		log.Logf("%v", pods1UIDs)
		log.Logf("%v", pods2UIDs)
		log.Logf("pods unchanged after operator upgraded")
		return nil
	}

	log.Logf("%s", string(pods1Yaml))
	log.Logf("%s", string(pods2Yaml))
	log.Logf("%v", pods1UIDs)
	log.Logf("%v", pods2UIDs)
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

func (oa *OperatorActions) UpgradeOperatorOrDie(info *OperatorConfig) {
	if err := oa.UpgradeOperator(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *OperatorActions) getBlockWriterPod(bpc *BlockWriterConfig, database string) *corev1.Pod {

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: bpc.Namespace,
			Name:      blockWriterPodNameInfo(bpc),
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
						fmt.Sprintf("--namespace=%s", bpc.Namespace),
						fmt.Sprintf("--cluster-name=%s", bpc.ClusterName),
						fmt.Sprintf("--database=%s", database),
						fmt.Sprintf("--password=%s", bpc.Password),
						fmt.Sprintf("--table-num=%d", bpc.BlockWriteConfig.TableNum),
						fmt.Sprintf("--concurrency=%d", bpc.BlockWriteConfig.Concurrency),
						fmt.Sprintf("--batch-size=%d", bpc.BlockWriteConfig.BatchSize),
						fmt.Sprintf("--raw-size=%d", bpc.BlockWriteConfig.RawSize),
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}
	if bpc.OperatorTag != "e2e" {
		pod.Spec.Containers[0].ImagePullPolicy = corev1.PullAlways
	}
	return pod
}

func (oa *OperatorActions) BeginInsertDataToInfo(bwc *BlockWriterConfig) error {
	oa.EmitEventInfo(bwc, fmt.Sprintf("BeginInsertData: concurrency: %d", bwc.BlockWriteConfig.Concurrency))

	bwpod := oa.getBlockWriterPod(bwc, "sbtest")
	bwpod, err := oa.kubeCli.CoreV1().Pods(bwc.Namespace).Create(context.TODO(), bwpod, metav1.CreateOptions{})
	if err != nil {
		log.Logf("ERROR: %v", err)
		return err
	}
	bwc.blockWriterPod = bwpod
	err = pod.WaitForPodRunningInNamespace(oa.kubeCli, bwpod)
	if err != nil {
		return err
	}
	log.Logf("begin insert Data in pod[%s/%s]", bwpod.Namespace, bwpod.Name)
	return nil
}

func (oa *OperatorActions) BeginInsertDataToOrDieInfo(bwc *BlockWriterConfig) {
	err := oa.BeginInsertDataToInfo(bwc)
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

//
func (oa *OperatorActions) StopInsertDataToInfo(bwc *BlockWriterConfig) {
	if bwc.blockWriterPod == nil {
		return
	}
	oa.EmitEventInfo(bwc, "StopInsertData")

	err := wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		pod := bwc.blockWriterPod
		err = oa.kubeCli.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
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
	bwc.blockWriterPod = nil
}

func (oa *OperatorActions) manifestPath(tag string) string {
	return filepath.Join(oa.cfg.ManifestDir, tag)
}

func (oa *OperatorActions) chartPath(name string, tag string) string {
	return filepath.Join(oa.cfg.ChartDir, tag, name)
}

func (oa *OperatorActions) operatorChartPath(tag string) string {
	return oa.chartPath(operartorChartName, tag)
}

func (oa *OperatorActions) tidbClusterChartPath(tag string) string {
	return oa.chartPath(tidbClusterChartName, tag)
}

func (oa *OperatorActions) backupChartPath(tag string) string {
	return oa.chartPath(backupChartName, tag)
}

func (oa *OperatorActions) drainerChartPath(tag string) string {
	return oa.chartPath(drainerChartName, tag)
}

func (oa *OperatorActions) setPartitionAnnotation(namespace, tcName, component string, ordinal int) error {
	// add annotation to pause statefulset upgrade process
	cmd := fmt.Sprintf("kubectl annotate tc %s -n %s tidb.pingcap.com/%s-partition=%d --overwrite",
		tcName, namespace, component, ordinal)
	log.Logf("%s", cmd)
	output, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("fail to set annotation for [%s/%s], component: %s, partition: %d, err: %v, output: %s", namespace, tcName, component, ordinal, err, string(output))
	}
	return nil
}

// getMemberContainer gets member container
func getMemberContainer(kubeCli kubernetes.Interface, stsGetter typedappsv1.StatefulSetsGetter, namespace, tcName, component string) (*corev1.Container, bool) {
	sts, err := stsGetter.StatefulSets(namespace).Get(context.TODO(), fmt.Sprintf("%s-%s", tcName, component), metav1.GetOptions{})
	if err != nil {
		log.Logf("ERROR: failed to get sts for component %s of cluster %s/%s", component, namespace, tcName)
		return nil, false
	}
	return getStsContainer(kubeCli, sts, component)
}

func getStsContainer(kubeCli kubernetes.Interface, sts *apps.StatefulSet, containerName string) (*corev1.Container, bool) {
	listOption := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(sts.Spec.Selector.MatchLabels).String(),
	}
	podList, err := kubeCli.CoreV1().Pods(sts.Namespace).List(context.TODO(), listOption)
	if err != nil {
		log.Logf("ERROR: fail to get pods for container %s of sts %s/%s", containerName, sts.Namespace, sts.Name)
		return nil, false
	}
	if len(podList.Items) == 0 {
		log.Logf("ERROR: no pods found for component %s of cluster %s/%s", containerName, sts.Namespace, sts.Name)
		return nil, false
	}
	pod := podList.Items[0]
	if len(pod.Spec.Containers) == 0 {
		log.Logf("ERROR: no containers found for component %s of cluster %s/%s", containerName, sts.Namespace, sts.Name)
		return nil, false
	}
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			return &container, true
		}
	}
	return nil, false
}

func (oa *OperatorActions) pdMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	if tc.Spec.PD == nil {
		log.Logf("no pd in tc spec, skip")
		return true, nil
	}
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	pdSetName := controller.PDMemberName(tcName)
	tcID := fmt.Sprintf("%s/%s", ns, tcName)
	pdStsID := fmt.Sprintf("%s/%s", ns, pdSetName)

	pdSet, err := oa.tcStsGetter.StatefulSets(ns).Get(context.TODO(), pdSetName, metav1.GetOptions{})
	if err != nil {
		log.Logf("failed to get StatefulSet: %q, %v", pdStsID, err)
		return false, nil
	}

	if pdSet.Status.CurrentRevision != pdSet.Status.UpdateRevision {
		log.Logf("pd sts .Status.CurrentRevision (%s) != .Status.UpdateRevision (%s)", pdSet.Status.CurrentRevision, pdSet.Status.UpdateRevision)
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), pdSet) {
		return false, nil
	}

	if tc.Status.PD.StatefulSet == nil {
		log.Logf("TidbCluster: %q .status.PD.StatefulSet is nil", tcID)
		return false, nil
	}
	failureCount := len(tc.Status.PD.FailureMembers)
	replicas := tc.Spec.PD.Replicas + int32(failureCount)
	if *pdSet.Spec.Replicas != replicas {
		log.Logf("StatefulSet: %q .spec.Replicas(%d) != %d", pdStsID, *pdSet.Spec.Replicas, replicas)
		return false, nil
	}
	if pdSet.Status.ReadyReplicas != tc.Spec.PD.Replicas {
		log.Logf("StatefulSet: %q .status.ReadyReplicas(%d) != %d", pdStsID, pdSet.Status.ReadyReplicas, tc.Spec.PD.Replicas)
		return false, nil
	}
	if len(tc.Status.PD.Members) != int(tc.Spec.PD.Replicas) {
		log.Logf("TidbCluster: %q .status.PD.Members count(%d) != %d", tcID, len(tc.Status.PD.Members), tc.Spec.PD.Replicas)
		return false, nil
	}
	if pdSet.Status.ReadyReplicas != pdSet.Status.Replicas {
		log.Logf("StatefulSet: %q .status.ReadyReplicas(%d) != .status.Replicas(%d)", pdStsID, pdSet.Status.ReadyReplicas, pdSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(oa.kubeCli, oa.tcStsGetter, ns, tc.Name, label.PDLabelVal)
	if !found {
		log.Logf("StatefulSet: %q not found containers[name=pd] or pod %s-0", pdStsID, pdSetName)
		return false, nil
	}

	if tc.PDImage() != c.Image {
		log.Logf("StatefulSet: %q .spec.template.spec.containers[name=pd].image(%s) != %s", pdStsID, c.Image, tc.PDImage())
		return false, nil
	}

	for _, member := range tc.Status.PD.Members {
		if !member.Health {
			log.Logf("TidbCluster: %q pd member(%s/%s) is not health", tcID, member.ID, member.Name)
			return false, nil
		}
	}

	pdServiceName := controller.PDMemberName(tcName)
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(context.TODO(), pdServiceName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get service: %s/%s", ns, pdServiceName)
		return false, nil
	}
	pdPeerServiceName := controller.PDPeerMemberName(tcName)
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(context.TODO(), pdPeerServiceName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get peer service: %s/%s", ns, pdPeerServiceName)
		return false, nil
	}

	log.Logf("pd members are ready for tc %q", tcID)
	return true, nil
}

func (oa *OperatorActions) tikvMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	if tc.Spec.TiKV == nil {
		log.Logf("no tikv in tc spec, skip")
		return true, nil
	}
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	tikvSetName := controller.TiKVMemberName(tcName)
	tcID := fmt.Sprintf("%s/%s", ns, tcName)
	tikvStsID := fmt.Sprintf("%s/%s", ns, tikvSetName)

	tikvSet, err := oa.tcStsGetter.StatefulSets(ns).Get(context.TODO(), tikvSetName, metav1.GetOptions{})
	if err != nil {
		log.Logf("failed to get StatefulSet: %q, %v", tikvStsID, err)
		return false, nil
	}

	if tikvSet.Status.CurrentRevision != tikvSet.Status.UpdateRevision {
		log.Logf("tikv sts .Status.CurrentRevision (%s) != .Status.UpdateRevision (%s)", tikvSet.Status.CurrentRevision, tikvSet.Status.UpdateRevision)
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), tikvSet) {
		return false, nil
	}

	if tc.Status.TiKV.StatefulSet == nil {
		log.Logf("TidbCluster: %q .status.TiKV.StatefulSet is nil", tcID)
		return false, nil
	}
	failureCount := len(tc.Status.TiKV.FailureStores)
	replicas := tc.Spec.TiKV.Replicas + int32(failureCount)
	if *tikvSet.Spec.Replicas != replicas {
		log.Logf("StatefulSet: %q .spec.Replicas(%d) != %d", tikvStsID, *tikvSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tikvSet.Status.ReadyReplicas != replicas {
		log.Logf("StatefulSet: %q .status.ReadyReplicas(%d) != %d", tikvStsID, tikvSet.Status.ReadyReplicas, replicas)
		return false, nil
	}
	if len(tc.Status.TiKV.Stores) != int(replicas) {
		log.Logf("TidbCluster: %q .status.TiKV.Stores.count(%d) != %d", tcID, len(tc.Status.TiKV.Stores), replicas)
		return false, nil
	}
	if tikvSet.Status.ReadyReplicas != tikvSet.Status.Replicas {
		log.Logf("StatefulSet: %q .status.ReadyReplicas(%d) != .status.Replicas(%d)", tikvStsID, tikvSet.Status.ReadyReplicas, tikvSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(oa.kubeCli, oa.tcStsGetter, ns, tc.Name, label.TiKVLabelVal)
	if !found {
		log.Logf("StatefulSet: %q not found containers[name=tikv] or pod %s-0", tikvStsID, tikvSetName)
		return false, nil
	}

	if tc.TiKVImage() != c.Image {
		log.Logf("StatefulSet: %q .spec.template.spec.containers[name=tikv].image(%s) != %s", tikvStsID, c.Image, tc.TiKVImage())
		return false, nil
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.State != v1alpha1.TiKVStateUp {
			log.Logf("TidbCluster: %q .status.TiKV store(%s) state != %s", tcID, store.ID, v1alpha1.TiKVStateUp)
			return false, nil
		}
	}

	tikvPeerServiceName := controller.TiKVPeerMemberName(tcName)
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(context.TODO(), tikvPeerServiceName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get peer service: %s/%s", ns, tikvPeerServiceName)
		return false, nil
	}

	log.Logf("tikv members are ready for tc %q", tcID)
	return true, nil
}

func (oa *OperatorActions) tiflashMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	if tc.Spec.TiFlash == nil {
		log.Logf("no tiflash in tc spec, skip")
		return true, nil
	}
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	tiflashSetName := controller.TiFlashMemberName(tcName)
	tcID := fmt.Sprintf("%s/%s", ns, tcName)
	tiflashStsID := fmt.Sprintf("%s/%s", ns, tiflashSetName)

	tiflashSet, err := oa.tcStsGetter.StatefulSets(ns).Get(context.TODO(), tiflashSetName, metav1.GetOptions{})
	if err != nil {
		log.Logf("failed to get StatefulSet: %q, %v", tiflashStsID, err)
		return false, nil
	}

	if tiflashSet.Status.CurrentRevision != tiflashSet.Status.UpdateRevision {
		log.Logf("tiflash sts .Status.CurrentRevision (%s) != .Status.UpdateRevision (%s)", tiflashSet.Status.CurrentRevision, tiflashSet.Status.UpdateRevision)
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), tiflashSet) {
		return false, nil
	}

	if tc.Status.TiFlash.StatefulSet == nil {
		log.Logf("TidbCluster: %q .status.TiFlash.StatefulSet is nil", tcID)
		return false, nil
	}
	failureCount := len(tc.Status.TiFlash.FailureStores)
	replicas := tc.Spec.TiFlash.Replicas + int32(failureCount)
	if *tiflashSet.Spec.Replicas != replicas {
		log.Logf("TiFlash StatefulSet: %q .spec.Replicas(%d) != %d", tiflashStsID, *tiflashSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tiflashSet.Status.ReadyReplicas != replicas {
		log.Logf("TiFlash StatefulSet: %q .status.ReadyReplicas(%d) != %d", tiflashStsID, tiflashSet.Status.ReadyReplicas, replicas)
		return false, nil
	}
	if len(tc.Status.TiFlash.Stores) != int(replicas) {
		log.Logf("TidbCluster: %q .status.TiFlash.Stores.count(%d) != %d", tcID, len(tc.Status.TiFlash.Stores), replicas)
		return false, nil
	}
	if tiflashSet.Status.ReadyReplicas != tiflashSet.Status.Replicas {
		log.Logf("StatefulSet: %q .status.ReadyReplicas(%d) != .status.Replicas(%d)", tiflashStsID, tiflashSet.Status.ReadyReplicas, tiflashSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(oa.kubeCli, oa.tcStsGetter, ns, tc.Name, label.TiFlashLabelVal)
	if !found {
		log.Logf("StatefulSet: %q not found containers[name=tiflash] or pod %s-0", tiflashStsID, tiflashSetName)
		return false, nil
	}

	if tc.TiFlashImage() != c.Image {
		log.Logf("StatefulSet: %q .spec.template.spec.containers[name=tiflash].image(%s) != %s", tiflashStsID, c.Image, tc.TiFlashImage())
		return false, nil
	}

	for _, store := range tc.Status.TiFlash.Stores {
		if store.State != v1alpha1.TiKVStateUp {
			log.Logf("TiFlash TidbCluster: %q's store(%s) state != %s", tcID, store.ID, v1alpha1.TiKVStateUp)
			return false, nil
		}
	}

	tiflashPeerServiceName := controller.TiFlashPeerMemberName(tcName)
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(context.TODO(), tiflashPeerServiceName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get peer service: %s/%s", ns, tiflashPeerServiceName)
		return false, nil
	}

	log.Logf("tiflash members are ready for tc %q", tcID)
	return true, nil
}

func (oa *OperatorActions) tidbMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	if tc.Spec.TiDB == nil {
		log.Logf("no tidb in tc spec, skip")
		return true, nil
	}
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	tidbSetName := controller.TiDBMemberName(tcName)
	tcID := fmt.Sprintf("%s/%s", ns, tcName)
	tidbStsID := fmt.Sprintf("%s/%s", ns, tidbSetName)

	tidbSet, err := oa.tcStsGetter.StatefulSets(ns).Get(context.TODO(), tidbSetName, metav1.GetOptions{})
	if err != nil {
		log.Logf("failed to get StatefulSet: %q, %v", tidbStsID, err)
		return false, nil
	}

	if tidbSet.Status.CurrentRevision != tidbSet.Status.UpdateRevision {
		log.Logf("tidb sts .Status.CurrentRevision (%s) != .Status.UpdateRevision (%s)", tidbSet.Status.CurrentRevision, tidbSet.Status.UpdateRevision)
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), tidbSet) {
		return false, nil
	}

	if tc.Status.TiDB.StatefulSet == nil {
		log.Logf("TidbCluster: %q .status.TiDB.StatefulSet is nil", tcID)
		return false, nil
	}
	failureCount := len(tc.Status.TiDB.FailureMembers)
	replicas := tc.Spec.TiDB.Replicas + int32(failureCount)
	if *tidbSet.Spec.Replicas != replicas {
		log.Logf("StatefulSet: %q .spec.Replicas(%d) != %d", tidbStsID, *tidbSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tidbSet.Status.ReadyReplicas != tc.Spec.TiDB.Replicas {
		log.Logf("StatefulSet: %q .status.ReadyReplicas(%d) != %d", tidbStsID, tidbSet.Status.ReadyReplicas, tc.Spec.TiDB.Replicas)
		return false, nil
	}
	if len(tc.Status.TiDB.Members) != int(tc.Spec.TiDB.Replicas) {
		log.Logf("TidbCluster: %q .status.TiDB.Members count(%d) != %d", tcID, len(tc.Status.TiDB.Members), tc.Spec.TiDB.Replicas)
		return false, nil
	}
	if tidbSet.Status.ReadyReplicas != tidbSet.Status.Replicas {
		log.Logf("StatefulSet: %q .status.ReadyReplicas(%d) != .status.Replicas(%d)", tidbStsID, tidbSet.Status.ReadyReplicas, tidbSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(oa.kubeCli, oa.tcStsGetter, ns, tc.Name, label.TiDBLabelVal)
	if !found {
		log.Logf("StatefulSet: %q not found containers[name=tidb] or pod %s-0", tidbStsID, tidbSetName)
		return false, nil
	}

	if tc.TiDBImage() != c.Image {
		log.Logf("StatefulSet: %q .spec.template.spec.containers[name=tidb].image(%s) != %s", tidbStsID, c.Image, tc.TiDBImage())
		return false, nil
	}

	tidbServiceName := controller.TiDBMemberName(tcName)
	if _, err = oa.kubeCli.CoreV1().Services(ns).Get(context.TODO(), tidbServiceName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get service: %s/%s", ns, tidbServiceName)
		return false, nil
	}
	tidbPeerServiceName := controller.TiDBPeerMemberName(tcName)
	if _, err = oa.kubeCli.CoreV1().Services(ns).Get(context.TODO(), tidbPeerServiceName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get peer service: %s/%s", ns, tidbPeerServiceName)
		return false, nil
	}

	log.Logf("tidb members are ready for tc %q", tcID)
	return true, nil
}

func (oa *OperatorActions) dmMasterMembersReadyFn(dc *v1alpha1.DMCluster) bool {
	dcName := dc.GetName()
	ns := dc.GetNamespace()
	masterSetName := controller.DMMasterMemberName(dcName)

	masterSet, err := oa.tcStsGetter.StatefulSets(ns).Get(context.TODO(), masterSetName, metav1.GetOptions{})
	if err != nil {
		log.Logf("failed to get statefulset: %s/%s, %v", ns, masterSetName, err)
		return false
	}

	if masterSet.Status.CurrentRevision != masterSet.Status.UpdateRevision {
		log.Logf("dm master .Status.CurrentRevision (%s) != .Status.UpdateRevision (%s)", masterSet.Status.CurrentRevision, masterSet.Status.UpdateRevision)
		return false
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), masterSet) {
		return false
	}

	if dc.Status.Master.StatefulSet == nil {
		log.Logf("DmCluster: %s/%s .status.Master.StatefulSet is nil", ns, dcName)
		return false
	}

	if !dc.MasterAllPodsStarted() {
		log.Logf("DmCluster: %s/%s not all master pods started, desired(%d) != started(%d)",
			ns, dcName, dc.MasterStsDesiredReplicas(), dc.MasterStsActualReplicas())
		return false
	}

	if !dc.MasterAllMembersReady() {
		log.Logf("DmCluster: %s/%s not all master members are healthy", ns, dcName)
		return false
	}

	c, found := getMemberContainer(oa.kubeCli, oa.tcStsGetter, ns, dcName, label.DMMasterLabelVal)
	if !found {
		log.Logf("statefulset: %s/%s not found containers[name=dm-master] or pod %s-0",
			ns, masterSetName, masterSetName)
		return false
	}

	if dc.MasterImage() != c.Image {
		log.Logf("statefulset: %s/%s .spec.template.spec.containers[name=dm-master].image(%s) != %s",
			ns, masterSetName, c.Image, dc.MasterImage())
		return false
	}

	masterServiceName := controller.DMMasterMemberName(dcName)
	masterPeerServiceName := controller.DMMasterPeerMemberName(dcName)
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(context.TODO(), masterServiceName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get service: %s/%s", ns, masterServiceName)
		return false
	}
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(context.TODO(), masterPeerServiceName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get peer service: %s/%s", ns, masterPeerServiceName)
		return false
	}

	return true
}

func (oa *OperatorActions) dmMasterMembersDeleted(ns, dcName string) bool {
	stsName := controller.DMMasterMemberName(dcName)
	_, err := oa.tcStsGetter.StatefulSets(ns).Get(context.TODO(), stsName, metav1.GetOptions{})
	if !errors.IsNotFound(err) {
		return false
	}
	svcName := controller.DMMasterMemberName(dcName)
	_, err = oa.kubeCli.CoreV1().Services(ns).Get(context.TODO(), svcName, metav1.GetOptions{})
	if !errors.IsNotFound(err) {
		return false
	}
	peerSvcName := controller.DMMasterPeerMemberName(dcName)
	_, err = oa.kubeCli.CoreV1().Services(ns).Get(context.TODO(), peerSvcName, metav1.GetOptions{})
	return errors.IsNotFound(err)
}

func (oa *OperatorActions) dmWorkerMembersDeleted(ns, dcName string) bool {
	stsName := controller.DMWorkerMemberName(dcName)
	_, err := oa.tcStsGetter.StatefulSets(ns).Get(context.TODO(), stsName, metav1.GetOptions{})
	if !errors.IsNotFound(err) {
		return false
	}
	peerSvcName := controller.DMWorkerPeerMemberName(dcName)
	_, err = oa.kubeCli.CoreV1().Services(ns).Get(context.TODO(), peerSvcName, metav1.GetOptions{})
	return errors.IsNotFound(err)
}

// TODO: try to simplify the code with dmMasterMembersReadyFn.
func (oa *OperatorActions) dmWorkerMembersReadyFn(dc *v1alpha1.DMCluster) bool {
	dcName := dc.GetName()
	ns := dc.GetNamespace()
	workerSetName := controller.DMWorkerMemberName(dcName)

	workerSet, err := oa.tcStsGetter.StatefulSets(ns).Get(context.TODO(), workerSetName, metav1.GetOptions{})
	if err != nil {
		log.Logf("failed to get statefulset: %s/%s, %v", ns, workerSetName, err)
		return false
	}

	if workerSet.Status.CurrentRevision != workerSet.Status.UpdateRevision {
		log.Logf("dm worker .Status.CurrentRevision (%s) != .Status.UpdateRevision (%s)", workerSet.Status.CurrentRevision, workerSet.Status.UpdateRevision)
		return false
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), workerSet) {
		return false
	}

	if dc.Status.Worker.StatefulSet == nil {
		log.Logf("DmCluster: %s/%s .status.Worker.StatefulSet is nil", ns, dcName)
		return false
	}

	if !dc.WorkerAllPodsStarted() {
		log.Logf("DmCluster: %s/%s not all worker pods started, desired(%d) != started(%d)",
			ns, dcName, dc.WorkerStsDesiredReplicas(), dc.WorkerStsActualReplicas())
		return false
	}

	if !dc.WorkerAllMembersReady() {
		log.Logf("DmCluster: %s/%s some worker members are offline", ns, dcName)
		return false
	}

	c, found := getMemberContainer(oa.kubeCli, oa.tcStsGetter, ns, dcName, label.DMWorkerLabelVal)
	if !found {
		log.Logf("statefulset: %s/%s not found containers[name=dm-worker] or pod %s-0",
			ns, workerSetName, workerSetName)
		return false
	}

	if dc.WorkerImage() != c.Image {
		log.Logf("statefulset: %s/%s .spec.template.spec.containers[name=dm-worker].image(%s) != %s",
			ns, workerSetName, c.Image, dc.WorkerImage())
		return false
	}

	workerPeerServiceName := controller.DMWorkerPeerMemberName(dcName)
	if _, err := oa.kubeCli.CoreV1().Services(ns).Get(context.TODO(), workerPeerServiceName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get peer service: %s/%s", ns, workerPeerServiceName)
		return false
	}

	return true
}

func (oa *OperatorActions) reclaimPolicySyncFn(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Labels(),
		).String(),
	}
	var pvcList *corev1.PersistentVolumeClaimList
	var err error
	if pvcList, err = oa.kubeCli.CoreV1().PersistentVolumeClaims(ns).List(context.TODO(), listOptions); err != nil {
		log.Logf("failed to list pvs for tidbcluster %s/%s, %v", ns, tcName, err)
		return false, nil
	}

	for _, pvc := range pvcList.Items {
		pvName := pvc.Spec.VolumeName
		if pv, err := oa.kubeCli.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{}); err != nil {
			log.Logf("failed to get pv: %s, error: %v", pvName, err)
			return false, nil
		} else if pv.Spec.PersistentVolumeReclaimPolicy != *tc.Spec.PVReclaimPolicy {
			log.Logf("pv: %s's reclaimPolicy is not Retain", pvName)
			return false, nil
		}
	}

	return true, nil
}

func (oa *OperatorActions) metaSyncFn(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	pdClient, cancel, err := oa.getPDClient(tc)
	if err != nil {
		log.Logf("failed to create external PD client for tidb cluster %q: %v", tc.GetName(), err)
		return false, nil
	}
	defer cancel()
	var cluster *metapb.Cluster
	if cluster, err = pdClient.GetCluster(); err != nil {
		log.Logf("failed to get cluster from pdControl: %s/%s, error: %v", ns, tcName, err)
		return false, nil
	}

	clusterID := strconv.FormatUint(cluster.Id, 10)
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Labels(),
		).String(),
	}

	var podList *corev1.PodList
	if podList, err = oa.kubeCli.CoreV1().Pods(ns).List(context.TODO(), listOptions); err != nil {
		log.Logf("failed to list pods for tidbcluster %s/%s, %v", ns, tcName, err)
		return false, nil
	}

outerLoop:
	for _, pod := range podList.Items {
		podName := pod.GetName()
		if pod.Labels[label.ClusterIDLabelKey] != clusterID {
			log.Logf("tidbcluster %s/%s's pod %s's label %s not equals %s ",
				ns, tcName, podName, label.ClusterIDLabelKey, clusterID)
			return false, nil
		}

		component := pod.Labels[label.ComponentLabelKey]
		switch component {
		case label.PDLabelVal:
			var memberID string
			members, err := pdClient.GetMembers()
			if err != nil {
				log.Logf("failed to get members for tidbcluster %s/%s, %v", ns, tcName, err)
				return false, nil
			}
			for _, member := range members.Members {
				if member.Name == podName {
					memberID = strconv.FormatUint(member.GetMemberId(), 10)
					break
				}
			}
			if memberID == "" {
				log.Logf("tidbcluster: %s/%s's pod %s label [%s] is empty",
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
				log.Logf("failed to get stores for tidbcluster %s/%s, %v", ns, tcName, err)
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
				log.Logf("tidbcluster: %s/%s's pod %s label [%s] is empty",
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
		if pvc, err = oa.kubeCli.CoreV1().PersistentVolumeClaims(ns).Get(context.TODO(), pvcName, metav1.GetOptions{}); err != nil {
			log.Logf("failed to get pvc %s/%s for pod %s/%s", ns, pvcName, ns, podName)
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
		if pv, err = oa.kubeCli.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{}); err != nil {
			log.Logf("failed to get pv for pvc %s/%s, %v", ns, pvcName, err)
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

func (oa *OperatorActions) schedulerHAFn(tc *v1alpha1.TidbCluster) (bool, error) {
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
		if podList, err = oa.kubeCli.CoreV1().Pods(ns).List(context.TODO(), listOptions); err != nil {
			log.Logf("failed to list pods for tidbcluster %s/%s, %v", ns, tcName, err)
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

func (oa *OperatorActions) podsScheduleAnnHaveDeleted(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Labels()).String(),
	}

	pvcList, err := oa.kubeCli.CoreV1().PersistentVolumeClaims(ns).List(context.TODO(), listOptions)
	if err != nil {
		log.Logf("failed to list pvcs for tidb cluster %s/%s, err: %v", ns, tcName, err)
		return false, nil
	}

	for _, pvc := range pvcList.Items {
		pvcName := pvc.GetName()
		l := label.Label(pvc.Labels)
		if !(l.IsPD() || l.IsTiKV()) {
			continue
		}

		if _, exist := pvc.Annotations[label.AnnPVCPodScheduling]; exist {
			log.Logf("tidb cluster %s/%s pvc %s has pod scheduling annotation", ns, tcName, pvcName)
			return false, nil
		}
	}

	return true, nil
}

func (oa *OperatorActions) checkReclaimPVSuccess(tc *v1alpha1.TidbCluster) (bool, error) {
	// check pv reclaim	for pd
	if err := oa.checkComponentReclaimPVSuccess(tc, label.PDLabelVal); err != nil {
		log.Logf("%v", err)
		return false, nil
	}

	// check pv reclaim for tikv
	if err := oa.checkComponentReclaimPVSuccess(tc, label.TiKVLabelVal); err != nil {
		log.Logf("%v", err)
		return false, nil
	}
	return true, nil
}

func (oa *OperatorActions) checkComponentReclaimPVSuccess(tc *v1alpha1.TidbCluster, component string) error {
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

func (oa *OperatorActions) getComponentPVCList(tc *v1alpha1.TidbCluster, component string) ([]corev1.PersistentVolumeClaim, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Component(component).Labels()).String(),
	}

	pvcList, err := oa.kubeCli.CoreV1().PersistentVolumeClaims(ns).List(context.TODO(), listOptions)
	if err != nil {
		err := fmt.Errorf("failed to list pvcs for tidb cluster %s/%s, component: %s, err: %v", ns, tcName, component, err)
		return nil, err
	}
	return pvcList.Items, nil
}

func (oa *OperatorActions) getComponentPVList(tc *v1alpha1.TidbCluster, component string) ([]corev1.PersistentVolume, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Component(component).Namespace(ns).Labels()).String(),
	}

	pvList, err := oa.kubeCli.CoreV1().PersistentVolumes().List(context.TODO(), listOptions)
	if err != nil {
		err := fmt.Errorf("failed to list pvs for tidb cluster %s/%s, component: %s, err: %v", ns, tcName, component, err)
		return nil, err
	}
	return pvList.Items, nil
}

func (oa *OperatorActions) storeLabelsIsSet(tc *v1alpha1.TidbCluster, topologyKey string) (bool, error) {
	pdClient, cancel, err := oa.getPDClient(tc)
	if err != nil {
		log.Logf("failed to create external PD client for tidb cluster %q: %v", tc.GetName(), err)
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
			log.Logf("close response failed, err: %v", err)
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

func (oa *OperatorActions) cloneOperatorRepo() error {
	cmd := fmt.Sprintf("git clone %s %s", oa.cfg.OperatorRepoUrl, oa.cfg.OperatorRepoDir)
	log.Logf(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil && !strings.Contains(string(res), "already exists") {
		return fmt.Errorf("failed to clone tidb-operator repository: %v, %s", err, string(res))
	}

	return nil
}

func (oa *OperatorActions) checkoutTag(tagName string) error {
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
	log.Logf("running checkout tag commands: %s", cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check tag: %s, %v, %s", tagName, err, string(res))
	}

	return nil
}

func (oa *OperatorActions) DataIsTheSameAsInfo(bwc, otherBwc *BlockWriterConfig) (bool, error) {
	tableNum := otherBwc.BlockWriteConfig.TableNum

	dsn, cancel, err := oa.getTiDBDSN(bwc.Namespace, bwc.ClusterName, "sbtest", bwc.Password)
	if err != nil {
		return false, nil
	}
	defer cancel()
	infoDb, err := sql.Open("mysql", dsn)
	if err != nil {
		return false, err
	}
	defer infoDb.Close()
	otherDsn, otherCancel, err := oa.getTiDBDSN(otherBwc.Namespace, otherBwc.ClusterName, "sbtest", otherBwc.Password)
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
				bwc.Namespace, bwc.ClusterName, tableName, cnt,
				otherBwc.Namespace, otherBwc.ClusterName, tableName, otherCnt)
			return false, err
		}
		log.Logf("cluster %s/%s's table %s count(*) = %d and cluster %s/%s's table %s count(*) = %d",
			bwc.Namespace, bwc.ClusterName, tableName, cnt,
			otherBwc.Namespace, otherBwc.ClusterName, tableName, otherCnt)
	}

	return true, nil
}

func releaseIsExist(err error) bool {
	return strings.Contains(err.Error(), "already exists")
}

func getParentUIDFromJob(j batchv1.Job) (types.UID, bool) {
	controllerRef := metav1.GetControllerOf(&j)

	if controllerRef == nil {
		return types.UID(""), false
	}

	if controllerRef.Kind != "CronJob" {
		log.Logf("Job with non-CronJob parent, name %s namespace %s", j.Name, j.Namespace)
		return types.UID(""), false
	}

	return controllerRef.UID, true
}

func strPtr(s string) *string { return &s }

func (oa *OperatorActions) RegisterWebHookAndServiceOrDie(configName, namespace, service string, context *apimachinery.CertContext) {
	if err := oa.RegisterWebHookAndService(configName, namespace, service, context); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *OperatorActions) RegisterWebHookAndService(configName, namespace, service string, ctx *apimachinery.CertContext) error {
	client := oa.kubeCli
	log.Logf("Registering the webhook via the AdmissionRegistration API")

	failurePolicy := admissionV1beta1.Fail

	_, err := client.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(context.TODO(), &admissionV1beta1.ValidatingWebhookConfiguration{
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
					CABundle: ctx.SigningCert,
				},
			},
		},
	}, metav1.CreateOptions{})

	if err != nil {
		log.Logf("registering webhook config %s with namespace %s error %v", configName, namespace, err)
		return err
	}

	// The webhook configuration is honored in 10s.
	time.Sleep(10 * time.Second)

	return nil

}

func (oa *OperatorActions) CleanWebHookAndService(name string) error {
	err := oa.kubeCli.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete webhook config %v", err)
	}
	return nil
}

func (oa *OperatorActions) CleanWebHookAndServiceOrDie(name string) {
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

func (oa *OperatorActions) pumpIsHealthy(tcName, ns, podName string, tlsEnabled bool) bool {
	var err error
	var addr string
	if oa.fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(oa.fw, ns, fmt.Sprintf("pod/%s", podName), 8250)
		if err != nil {
			log.Logf("failed to forward port %d for %s/%s", 8250, ns, podName)
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
		tlsConfig, err = pdapi.GetTLSConfig(GetSecretListerWithCacheSynced(oa.kubeCli, 1*time.Second), pdapi.Namespace(ns), util.ClusterTLSSecretName(tcName, label.PumpLabelVal))
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
		log.Logf("cluster:[%s] call %s failed,error:%v", tcName, pumpHealthURL, err)
		return false
	}
	if res.StatusCode >= 400 {
		log.Logf("Error response %v", res.StatusCode)
		return false
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Logf("cluster:[%s] read response body failed,error:%v", tcName, err)
		return false
	}
	healths := pumpStatus{}
	err = json.Unmarshal(body, &healths)
	if err != nil {
		log.Logf("cluster:[%s] unmarshal failed,error:%v", tcName, err)
		return false
	}
	for _, status := range healths.StatusMap {
		if status.State != "online" {
			log.Logf("cluster:[%s] pump's state is not online", tcName)
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

func (oa *OperatorActions) drainerHealth(tcName, ns, podName string, tlsEnabled bool) bool {
	var body []byte
	var err error
	var addr string
	if oa.fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(oa.fw, ns, fmt.Sprintf("pod/%s", podName), 8249)
		if err != nil {
			log.Logf("failed to forward port %d for %s/%s", 8249, ns, podName)
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
		tlsConfig, err = pdapi.GetTLSConfig(GetSecretListerWithCacheSynced(oa.kubeCli, 1*time.Second), pdapi.Namespace(ns), util.ClusterTLSSecretName(tcName, "drainer"))
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
		log.Logf("cluster:[%s] call %s failed,error:%v", tcName, drainerHealthURL, err)
		return false
	}
	if res.StatusCode >= 400 {
		log.Logf("Error response %v", res.StatusCode)
		return false
	}
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		log.Logf("cluster:[%s] read response body failed,error:%v", tcName, err)
		return false
	}
	healths := drainerStatus{}
	err = json.Unmarshal(body, &healths)
	if err != nil {
		log.Logf("cluster:[%s] unmarshal failed,error:%v", tcName, err)
		return false
	}
	return len(healths.PumpPos) > 0
}

func (oa *OperatorActions) EmitEventInfo(info *BlockWriterConfig, message string) {
	oa.lock.Lock()
	defer oa.lock.Unlock()

	log.Logf("Event: %s", message)

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

func (oa *OperatorActions) RunEventWorker() {
	oa.lock.Lock()
	oa.eventWorkerRunning = true
	oa.lock.Unlock()
	log.Logf("Event worker started")
	wait.Forever(oa.eventWorker, 10*time.Second)
}

func (oa *OperatorActions) eventWorker() {
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
				log.Logf("cluster:[%s/%s] error recording event: %s, reason: %v",
					ns, clusterName, ev.message, err)
				retryEvents = append(retryEvents, ev)
				continue
			}
			log.Logf("cluster: [%s/%s] recoding event: %s", ns, clusterName, ev.message)
		}

		ce := oa.clusterEvents[key]
		ce.events = retryEvents
	}
}

func (oa *OperatorActions) cdcMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	if tc.Spec.TiCDC == nil {
		return true, nil
	}
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	cdcSetName := controller.TiCDCMemberName(tcName)
	tcID := fmt.Sprintf("%s/%s", ns, tcName)
	cdcStsID := fmt.Sprintf("%s/%s", ns, cdcSetName)

	cdcSet, err := oa.tcStsGetter.StatefulSets(ns).Get(context.TODO(), cdcSetName, metav1.GetOptions{})
	if err != nil {
		log.Logf("failed to get StatefulSet: %q, %v", cdcStsID, err)
		return false, nil
	}

	if cdcSet.Status.CurrentRevision != cdcSet.Status.UpdateRevision {
		log.Logf("cdc sts .Status.CurrentRevision (%s) != .Status.UpdateRevision (%s)", cdcSet.Status.CurrentRevision, cdcSet.Status.UpdateRevision)
		return false, nil
	}

	c, found := getMemberContainer(oa.kubeCli, oa.tcStsGetter, ns, tc.Name, label.TiCDCLabelVal)
	if !found {
		log.Logf("StatefulSet: %q not found containers[name=ticdc] or pod %s-0", cdcStsID, cdcSetName)
		return false, nil
	}

	if tc.TiCDCImage() != c.Image {
		log.Logf("StatefulSet: %q .spec.template.spec.containers[name=ticdc].image(%s) != %s", cdcStsID, c.Image, tc.TiCDCImage())
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), cdcSet) {
		return false, nil
	}

	log.Logf("cdc members are ready for tc %q", tcID)
	return true, nil
}

func (oa *OperatorActions) pumpMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	if tc.Spec.Pump == nil {
		log.Logf("no pump in tc spec, skip")
		return true, nil
	}
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	pumpSetName := controller.PumpMemberName(tcName)
	tcID := fmt.Sprintf("%s/%s", ns, tcName)
	pumpStsID := fmt.Sprintf("%s/%s", ns, pumpSetName)

	pumpSet, err := oa.tcStsGetter.StatefulSets(ns).Get(context.TODO(), pumpSetName, metav1.GetOptions{})
	if err != nil {
		log.Logf("failed to get StatefulSet: %q, %v", pumpStsID, err)
		return false, nil
	}

	if pumpSet.Status.CurrentRevision != pumpSet.Status.UpdateRevision {
		log.Logf("pump sts .Status.CurrentRevision (%s) != .Status.UpdateRevision (%s)", pumpSet.Status.CurrentRevision, pumpSet.Status.UpdateRevision)
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), pumpSet) {
		return false, nil
	}

	// check all pump replicas are online
	for i := 0; i < int(*pumpSet.Spec.Replicas); i++ {
		podName := fmt.Sprintf("%s-%d", pumpSetName, i)
		if !oa.pumpIsHealthy(tc.Name, tc.Namespace, podName, tc.IsTLSClusterEnabled()) {
			log.Logf("%s is not health yet", podName)
			return false, nil
		}
	}

	log.Logf("pump members are ready for tc %q", tcID)
	return true, nil
}

// FIXME: this duplicates with WaitForTidbClusterReady in crd_test_utils.go, and all functions in it
// TODO: sync with e2e doc
func (oa *OperatorActions) WaitForTidbClusterReady(tc *v1alpha1.TidbCluster, timeout, pollInterval time.Duration) error {
	if tc == nil {
		return fmt.Errorf("tidbcluster is nil, cannot call WaitForTidbClusterReady")
	}
	var checkErr error
	err := wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		var local *v1alpha1.TidbCluster
		var err error
		tcID := fmt.Sprintf("%s/%s", tc.Namespace, tc.Name)

		if local, err = oa.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(context.TODO(), tc.Name, metav1.GetOptions{}); err != nil {
			checkErr = fmt.Errorf("failed to get TidbCluster: %q, %v", tcID, err)
			return false, nil
		}

		if b, err := oa.pdMembersReadyFn(local); !b && err == nil {
			checkErr = fmt.Errorf("pd members are not ready for tc %q", tcID)
			return false, nil
		}

		if b, err := oa.tikvMembersReadyFn(local); !b && err == nil {
			checkErr = fmt.Errorf("tikv members are not ready for tc %q", tcID)
			return false, nil
		}

		if b, err := oa.tidbMembersReadyFn(local); !b && err == nil {
			checkErr = fmt.Errorf("tidb members are not ready for tc %q", tcID)
			return false, nil
		}

		if b, err := oa.tiflashMembersReadyFn(local); !b && err == nil {
			checkErr = fmt.Errorf("tiflash members are not ready for tc %q", tcID)
			return false, nil
		}

		if b, err := oa.pumpMembersReadyFn(local); !b && err == nil {
			checkErr = fmt.Errorf("pump members are not ready for tc %q", tcID)
			return false, nil
		}

		if b, err := oa.cdcMembersReadyFn(local); !b && err == nil {
			checkErr = fmt.Errorf("cdc members are not ready for tc %q", tcID)
			return false, nil
		}

		log.Logf("TidbCluster %q is ready", tcID)
		return true, nil
	})

	if err == wait.ErrWaitTimeout {
		err = checkErr
	}

	return err
}

func (oa *OperatorActions) WaitForDmClusterReady(dc *v1alpha1.DMCluster, timeout, pollInterval time.Duration) error {
	if dc == nil {
		return fmt.Errorf("DmCluster is nil, cannot call WaitForDmClusterReady")
	}
	var checkErr error
	err := wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		var (
			local *v1alpha1.DMCluster
			err   error
		)
		if local, err = oa.cli.PingcapV1alpha1().DMClusters(dc.Namespace).Get(context.TODO(), dc.Name, metav1.GetOptions{}); err != nil {
			checkErr = fmt.Errorf("failed to get DmCluster: %s/%s, %v", dc.Namespace, dc.Name, err)
			return false, nil
		}

		if b := oa.dmMasterMembersReadyFn(local); !b {
			checkErr = fmt.Errorf("dm master members are not ready for dc %q", dc.Name)
			return false, nil
		}
		if b := oa.dmWorkerMembersReadyFn(local); !b {
			checkErr = fmt.Errorf("dm worker memebers are not ready for dc %q", dc.Name)
			return false, nil
		}

		return true, nil
	})
	if err == wait.ErrWaitTimeout {
		return checkErr
	}
	return err
}

func (oa *OperatorActions) WaitForDmClusterDeleted(ns, dcName string, timeout, pollInterval time.Duration) error {
	var checkErr error
	err := wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		if b := oa.dmMasterMembersDeleted(ns, dcName); !b {
			checkErr = fmt.Errorf("dm master members are not deleted for dc %q", dcName)
			return false, nil
		}
		if b := oa.dmWorkerMembersDeleted(ns, dcName); !b {
			checkErr = fmt.Errorf("dm worker members are not deleted for dc %q", dcName)
			return false, nil
		}
		return true, nil
	})
	if err == wait.ErrWaitTimeout {
		return checkErr
	}
	return err
}

var dummyCancel = func() {}

func (oa *OperatorActions) getPDClient(tc *v1alpha1.TidbCluster) (pdapi.PDClient, context.CancelFunc, error) {
	if oa.fw != nil {
		return proxiedpdclient.NewProxiedPDClientFromTidbCluster(oa.fw, oa.secretLister, tc)
	}
	return controller.GetPDClient(oa.pdControl, tc), dummyCancel, nil
}

func (oa *OperatorActions) getTiDBDSN(ns, tcName, databaseName, password string) (string, context.CancelFunc, error) {
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
			log.Logf(sendErr.Error())
		}
		panic(fmt.Sprintf("failed to start webhook server %v", err))
	}
}

func blockWriterPodNameInfo(bpc *BlockWriterConfig) string {
	return fmt.Sprintf("%s-blockwriter", bpc.ClusterName)
}
