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
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	tngmname "github.com/pingcap/tidb-operator/pkg/manager/tidbngmonitoring"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	utildiscovery "github.com/pingcap/tidb-operator/pkg/util/discovery"
	e2eutil "github.com/pingcap/tidb-operator/tests/e2e/util"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/e2e/util/proxiedtidbclient"
	utilstatefulset "github.com/pingcap/tidb-operator/tests/e2e/util/statefulset"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	"github.com/pingcap/tidb-operator/tests/pkg/metrics"
	"github.com/pingcap/tidb-operator/tests/slack"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	aggregatorclientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/log"
)

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

func (oa *OperatorActions) RunKubectlOrDie(args ...string) string {
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

func (oa *OperatorActions) ReplaceCRDOrDie(info *OperatorConfig) {
	files, err := oa.crdFiles(info)
	framework.ExpectNoError(err, "failed to get CRD files")

	for _, file := range files {
		framework.RunKubectl("", "create", "-f", file) // exec `create` command to create new CRD
		oa.RunKubectlOrDie("replace", "-f", file)
	}

	oa.waitForCRDsReady()
}

func (oa *OperatorActions) CreateCRDOrDie(info *OperatorConfig) {
	files, err := oa.crdFiles(info)
	framework.ExpectNoError(err, "failed to get CRD files")

	for _, file := range files {
		oa.RunKubectlOrDie("create", "-f", file)
	}

	oa.waitForCRDsReady()
}

func (oa *OperatorActions) CreateReleasedCRDOrDie(version string) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/pingcap/tidb-operator/%s/manifests/crd.yaml", version)

	var lastErr error
	err := wait.PollImmediate(time.Second*10, time.Minute, func() (bool, error) {
		_, err := framework.RunKubectl("", "create", "-f", url)
		if err != nil {
			lastErr = err
			return false, nil
		}
		return true, nil
	})
	framework.ExpectNoError(err, "failed to create CRD of version %s: %s", version, lastErr)

	oa.waitForCRDsReady()
}

func (oa *OperatorActions) crdFiles(info *OperatorConfig) ([]string, error) {
	supportV1, err := utildiscovery.IsAPIGroupVersionSupported(oa.kubeCli.Discovery(), "apiextensions.k8s.io/v1")
	if err != nil {
		return nil, err
	}

	crdFile := oa.manifestPath("e2e/crd.yaml")
	astsCRDFile := oa.manifestPath("e2e/advanced-statefulset-crd.v1.yaml")
	if !supportV1 {
		crdFile = oa.manifestPath("e2e/crd_v1beta1.yaml")
		astsCRDFile = oa.manifestPath("e2e/advanced-statefulset-crd.v1beta1.yaml")
	}

	files := []string{
		crdFile,
	}
	if info.Enabled(features.AdvancedStatefulSet) {
		files = append(files, astsCRDFile)
	}

	return files, nil
}

func (oa *OperatorActions) waitForCRDsReady() {
	framework.Logf("Wait for all CRDs are established")
	e2eutil.WaitForCRDsEstablished(oa.apiExtCli, labels.Everything())

	// workaround for https://github.com/kubernetes/kubernetes/issues/65517
	framework.Logf("force sync kubectl cache")
	cmdArgs := []string{"sh", "-c", "rm -rf ~/.kube/cache ~/.kube/http-cache"}
	if _, err := exec.Command(cmdArgs[0], cmdArgs[1:]...).CombinedOutput(); err != nil {
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

type memberCheckContext struct {
	skip           bool
	stsName        string
	expectedImage  string
	services       []string
	status         v1alpha1.ComponentStatus
	checkComponent func(obj metav1.Object, sts *v1.StatefulSet) error
}

func (oa *OperatorActions) IsMembersReady(obj metav1.Object, component v1alpha1.MemberType) error {
	ns := obj.GetNamespace()

	var (
		err error
		ctx *memberCheckContext
	)

	switch cluster := obj.(type) {
	case *v1alpha1.TidbCluster:
		ctx, err = oa.memberCheckContextForTC(cluster, component)
	case *v1alpha1.DMCluster:
		ctx, err = oa.memberCheckContextForDC(cluster, component)
	case *v1alpha1.TidbNGMonitoring:
		ctx, err = oa.memberCheckContextForTNGM(cluster, component)
	default:
		err = fmt.Errorf("unsupported object type: %T", obj)
	}
	if err != nil {
		return err
	}

	stsName := ctx.stsName
	stsID := fmt.Sprintf("%s/%s", ns, ctx.stsName)

	if ctx.skip {
		log.Logf("component %s is not exist in spec, so skip", component)
		return nil
	}

	// check statefulset
	sts, err := oa.tcStsGetter.StatefulSets(ns).Get(context.TODO(), stsName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get sts %q: %s", stsID, err)
	}

	if sts.Status.CurrentRevision != sts.Status.UpdateRevision {
		return fmt.Errorf("sts %q is not ready, currentRevision: %s, updateRevision: %s", stsID, sts.Status.CurrentRevision, sts.Status.UpdateRevision)
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(oa.kubeCli, oa.asCli), sts) {
		return fmt.Errorf("not all pods are running and ready for sts %q", stsID)
	}

	// check the status of component
	err = ctx.checkComponent(obj, sts)
	if err != nil {
		return fmt.Errorf("%s members are not ready: %s", component, err)
	}

	// check containers
	containers, err := utilstatefulset.GetMemberContainersFromSts(oa.kubeCli, oa.tcStsGetter, ns, stsName, component)
	if err != nil {
		return fmt.Errorf("failed to get containers: %s", err)
	}
	for _, container := range containers {
		if container.Image != ctx.expectedImage {
			return fmt.Errorf("a container[%s] image is not expected, expected: %s, actual: %s", container.Name, ctx.expectedImage, container.Image)
		}
	}

	// check service
	for _, svc := range ctx.services {
		_, err := oa.kubeCli.CoreV1().Services(ns).Get(context.TODO(), svc, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get service: %s/%s", ns, svc)
		}
	}

	return nil
}

func (oa *OperatorActions) memberCheckContextForTC(tc *v1alpha1.TidbCluster, component v1alpha1.MemberType) (*memberCheckContext, error) {
	name := tc.GetName()
	stsName := fmt.Sprintf("%s-%s", name, component)

	var (
		skip           bool
		expectedImage  string
		services       []string
		checkComponent func(tc *v1alpha1.TidbCluster, sts *v1.StatefulSet) error
	)

	// check the status of component
	switch component {
	case v1alpha1.PDMemberType:
		skip = tc.Spec.PD == nil
		expectedImage = tc.PDImage()
		services = []string{controller.PDMemberName(name), controller.PDPeerMemberName(name)}
		checkComponent = oa.isPDMembersReady
	case v1alpha1.TiDBMemberType:
		skip = tc.Spec.TiDB == nil
		expectedImage = tc.TiDBImage()
		services = []string{controller.TiDBMemberName(name), controller.TiDBPeerMemberName(name)}
		checkComponent = oa.isTiDBMembersReady
	case v1alpha1.TiKVMemberType:
		skip = tc.Spec.TiKV == nil
		expectedImage = tc.TiKVImage()
		services = []string{controller.TiKVPeerMemberName(name)}
		checkComponent = oa.isTiKVMembersReady
	case v1alpha1.TiFlashMemberType:
		skip = tc.Spec.TiFlash == nil
		expectedImage = tc.TiFlashImage()
		services = []string{controller.TiFlashPeerMemberName(name)}
		checkComponent = oa.isTiFlashMembersReady
	case v1alpha1.TiCDCMemberType:
		skip = tc.Spec.TiCDC == nil
		expectedImage = tc.TiCDCImage()
		services = []string{controller.TiCDCPeerMemberName(name)}
		checkComponent = oa.isTiCDCMembersReady
	case v1alpha1.PumpMemberType:
		skip = tc.Spec.Pump == nil
		if !skip {
			expectedImage = *tc.PumpImage()
		}
		services = []string{controller.PumpMemberName(name), controller.PumpPeerMemberName(name)}
		checkComponent = oa.isPumpMembersReady
	default:
		return nil, fmt.Errorf("unknown component %s", component)
	}

	ctx := &memberCheckContext{
		skip:          skip,
		stsName:       stsName,
		expectedImage: expectedImage,
		services:      services,
		status:        tc.ComponentStatus(component),
		checkComponent: func(obj metav1.Object, sts *v1.StatefulSet) error {
			tc := obj.(*v1alpha1.TidbCluster)
			return checkComponent(tc, sts)
		},
	}

	return ctx, nil
}

func (oa *OperatorActions) memberCheckContextForDC(dc *v1alpha1.DMCluster, component v1alpha1.MemberType) (*memberCheckContext, error) {
	name := dc.GetName()
	stsName := fmt.Sprintf("%s-%s", name, component)

	var (
		skip           bool
		expectedImage  string
		services       []string
		checkComponent func(dc *v1alpha1.DMCluster, sts *v1.StatefulSet) error
	)

	// check the status of component
	switch component {
	case v1alpha1.DMMasterMemberType:
		skip = false
		expectedImage = dc.MasterImage()
		services = []string{controller.DMMasterMemberName(name), controller.DMMasterPeerMemberName(name)}
		checkComponent = oa.isDMMasterMembersReady
	case v1alpha1.DMWorkerMemberType:
		skip = dc.Spec.Worker == nil
		expectedImage = dc.WorkerImage()
		services = []string{controller.DMWorkerPeerMemberName(name)}
		checkComponent = oa.isDMWorkerMembersReady
	default:
		return nil, fmt.Errorf("unknown component %s", component)
	}

	ctx := &memberCheckContext{
		skip:          skip,
		stsName:       stsName,
		expectedImage: expectedImage,
		services:      services,
		status:        dc.ComponentStatus(component),
		checkComponent: func(obj metav1.Object, sts *v1.StatefulSet) error {
			dc := obj.(*v1alpha1.DMCluster)
			return checkComponent(dc, sts)
		},
	}

	return ctx, nil
}

func (oa *OperatorActions) memberCheckContextForTNGM(tngm *v1alpha1.TidbNGMonitoring, component v1alpha1.MemberType) (*memberCheckContext, error) {
	name := tngm.GetName()
	stsName := fmt.Sprintf("%s-%s", name, component)

	var (
		skip           bool
		expectedImage  string
		services       []string
		checkComponent func(tngm *v1alpha1.TidbNGMonitoring, sts *v1.StatefulSet) error
	)

	switch component {
	case v1alpha1.NGMonitoringMemberType:
		skip = false
		expectedImage = tngm.NGMonitoringImage()
		services = []string{tngmname.NGMonitoringHeadlessServiceName(name)}
		checkComponent = oa.isNGMMembersReady
	default:
		return nil, fmt.Errorf("unknown component %s", component)
	}

	ctx := &memberCheckContext{
		skip:          skip,
		stsName:       stsName,
		expectedImage: expectedImage,
		services:      services,
		status:        nil,
		checkComponent: func(obj metav1.Object, sts *v1.StatefulSet) error {
			tngm := obj.(*v1alpha1.TidbNGMonitoring)
			return checkComponent(tngm, sts)
		},
	}

	return ctx, nil
}

func (oa *OperatorActions) isPDMembersReady(tc *v1alpha1.TidbCluster, sts *v1.StatefulSet) error {
	if tc.Status.PD.StatefulSet == nil {
		return fmt.Errorf("sts in tc status is nil")
	}

	failureCount := len(tc.Status.PD.FailureMembers)
	replicas := tc.Spec.PD.Replicas + int32(failureCount)

	if *sts.Spec.Replicas != replicas {
		return fmt.Errorf("sts.spec.Replicas(%d) != %d", *sts.Spec.Replicas, replicas)
	}
	if sts.Status.ReadyReplicas != tc.Spec.PD.Replicas {
		return fmt.Errorf("sts.status.ReadyReplicas(%d) != %d", sts.Status.ReadyReplicas, tc.Spec.PD.Replicas)
	}
	if len(tc.Status.PD.Members) != int(tc.Spec.PD.Replicas) {
		return fmt.Errorf("sts.status.PD.Members count(%d) != %d", len(tc.Status.PD.Members), tc.Spec.PD.Replicas)
	}
	if sts.Status.ReadyReplicas != sts.Status.Replicas {
		return fmt.Errorf("sts.status.ReadyReplicas(%d) != sts.status.Replicas(%d)", sts.Status.ReadyReplicas, sts.Status.Replicas)
	}

	for _, member := range tc.Status.PD.Members {
		if !member.Health {
			return fmt.Errorf("pd member(%s/%s) is not health", member.ID, member.Name)
		}
	}

	return nil
}

func (oa *OperatorActions) isTiDBMembersReady(tc *v1alpha1.TidbCluster, sts *v1.StatefulSet) error {
	if tc.Status.TiDB.StatefulSet == nil {
		return fmt.Errorf("sts in tc status is nil")
	}

	failureCount := len(tc.Status.TiDB.FailureMembers)
	replicas := tc.Spec.TiDB.Replicas + int32(failureCount)

	if *sts.Spec.Replicas != replicas {
		return fmt.Errorf("sts.spec.Replicas(%d) != %d", *sts.Spec.Replicas, replicas)
	}
	if sts.Status.ReadyReplicas != tc.Spec.TiDB.Replicas {
		return fmt.Errorf("sts.status.ReadyReplicas(%d) != %d", sts.Status.ReadyReplicas, tc.Spec.TiDB.Replicas)
	}
	if len(tc.Status.TiDB.Members) != int(tc.Spec.TiDB.Replicas) {
		return fmt.Errorf("tc.status.TiKV.Stores.count(%d) != %d", len(tc.Status.TiDB.Members), tc.Spec.TiDB.Replicas)
	}
	if sts.Status.ReadyReplicas != sts.Status.Replicas {
		return fmt.Errorf("sts.status.ReadyReplicas(%d) != sts.status.Replicas(%d)", sts.Status.ReadyReplicas, sts.Status.Replicas)
	}

	for _, member := range tc.Status.TiDB.Members {
		if !member.Health {
			return fmt.Errorf("tidb member(%s) is unhealthy", member.Name)
		}
	}

	return nil
}

func (oa *OperatorActions) isTiKVMembersReady(tc *v1alpha1.TidbCluster, sts *v1.StatefulSet) error {
	if tc.Status.TiKV.StatefulSet == nil {
		return fmt.Errorf("sts in tc status is nil")
	}

	failureCount := len(tc.Status.TiKV.FailureStores)
	replicas := tc.Spec.TiKV.Replicas + int32(failureCount)

	if *sts.Spec.Replicas != replicas {
		return fmt.Errorf("sts.spec.Replicas(%d) != %d", *sts.Spec.Replicas, replicas)
	}
	if sts.Status.ReadyReplicas != replicas {
		return fmt.Errorf("sts.status.ReadyReplicas(%d) != %d", sts.Status.ReadyReplicas, replicas)
	}
	if len(tc.Status.TiKV.Stores) != int(replicas) {
		return fmt.Errorf("tc.status.TiKV.Stores.count(%d) != %d", len(tc.Status.TiKV.Stores), replicas)
	}
	if sts.Status.ReadyReplicas != sts.Status.Replicas {
		return fmt.Errorf("sts.status.ReadyReplicas(%d) != sts.status.Replicas(%d)", sts.Status.ReadyReplicas, sts.Status.Replicas)
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.State != v1alpha1.TiKVStateUp {
			return fmt.Errorf("tikv store(%s) is not %q", store.ID, v1alpha1.TiKVStateUp)
		}
	}

	return nil
}

func (oa *OperatorActions) isTiFlashMembersReady(tc *v1alpha1.TidbCluster, sts *v1.StatefulSet) error {
	if tc.Status.TiFlash.StatefulSet == nil {
		return fmt.Errorf("sts in tc status is nil")
	}

	failureCount := len(tc.Status.TiFlash.FailureStores)
	replicas := tc.Spec.TiFlash.Replicas + int32(failureCount)

	if *sts.Spec.Replicas != replicas {
		return fmt.Errorf("sts.spec.Replicas(%d) != %d", *sts.Spec.Replicas, replicas)
	}
	if sts.Status.ReadyReplicas != replicas {
		return fmt.Errorf("sts.status.ReadyReplicas(%d) != %d", sts.Status.ReadyReplicas, replicas)
	}
	if len(tc.Status.TiFlash.Stores) != int(replicas) {
		return fmt.Errorf("tc.status.TiFlash.Stores.count(%d) != %d", len(tc.Status.TiFlash.Stores), replicas)
	}
	if sts.Status.ReadyReplicas != sts.Status.Replicas {
		return fmt.Errorf("sts.status.ReadyReplicas(%d) != sts.status.Replicas(%d)", sts.Status.ReadyReplicas, sts.Status.Replicas)
	}

	for _, store := range tc.Status.TiFlash.Stores {
		if store.State != v1alpha1.TiKVStateUp {
			return fmt.Errorf("tiflash store(%s) is not %q", store.ID, v1alpha1.TiKVStateUp)
		}
	}

	return nil
}

func (oa *OperatorActions) isTiCDCMembersReady(tc *v1alpha1.TidbCluster, sts *v1.StatefulSet) error {
	if tc.Status.TiCDC.StatefulSet == nil {
		return fmt.Errorf("sts in tc status is nil")
	}

	if len(tc.Status.TiCDC.Captures) != int(tc.Spec.TiCDC.Replicas) {
		return fmt.Errorf("tc.status.TiCDC.Captures.count(%d) != %d", len(tc.Status.TiCDC.Captures), tc.Spec.TiCDC.Replicas)
	}

	return nil
}

func (oa *OperatorActions) isPumpMembersReady(tc *v1alpha1.TidbCluster, sts *v1.StatefulSet) error {
	tcName := tc.GetName()
	pumpStsName := controller.PumpMemberName(tcName)

	if tc.Status.Pump.StatefulSet == nil {
		return fmt.Errorf("sts in tc status is nil")
	}

	// check all pump replicas are online
	for i := 0; i < int(*sts.Spec.Replicas); i++ {
		podName := fmt.Sprintf("%s-%d", pumpStsName, i)
		if !oa.pumpIsHealthy(tc.Name, tc.Namespace, podName, tc.IsTLSClusterEnabled()) {
			return fmt.Errorf("%s is not healthy", podName)
		}
	}

	return nil
}

func (oa *OperatorActions) isDMMasterMembersReady(dc *v1alpha1.DMCluster, sts *v1.StatefulSet) error {
	if dc.Status.Master.StatefulSet == nil {
		return fmt.Errorf("sts in dc status is nil")
	}

	if !dc.MasterAllPodsStarted() {
		return fmt.Errorf("not all master pods started, desired(%d) != started(%d)", dc.MasterStsDesiredReplicas(), dc.MasterStsActualReplicas())
	}

	if !dc.MasterAllMembersReady() {
		return fmt.Errorf("not all master members are healthy")
	}

	return nil
}

func (oa *OperatorActions) isDMWorkerMembersReady(dc *v1alpha1.DMCluster, sts *v1.StatefulSet) error {
	if dc.Status.Worker.StatefulSet == nil {
		return fmt.Errorf("sts in dc status is nil")
	}

	if !dc.WorkerAllPodsStarted() {
		return fmt.Errorf("not all worker pods started, desired(%d) != started(%d)",
			dc.WorkerStsDesiredReplicas(), dc.WorkerStsActualReplicas())
	}

	if !dc.WorkerAllMembersReady() {
		return fmt.Errorf("some worker members are offline")
	}

	return nil
}

func (oa *OperatorActions) isNGMMembersReady(tngm *v1alpha1.TidbNGMonitoring, sts *v1.StatefulSet) error {
	if tngm.Status.NGMonitoring.StatefulSet == nil {
		return fmt.Errorf("sts in tngm status is nil")
	}
	return nil
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

// FIXME: this duplicates with WaitForTidbClusterReady in crd_test_utils.go, and all functions in it
// TODO: sync with e2e doc
func (oa *OperatorActions) WaitForTidbClusterReady(tc *v1alpha1.TidbCluster, timeout, pollInterval time.Duration) error {
	if tc == nil {
		return fmt.Errorf("tidbcluster is nil, cannot call WaitForTidbClusterReady")
	}
	var checkErr, err error
	var local *v1alpha1.TidbCluster
	tcID := fmt.Sprintf("%s/%s", tc.Namespace, tc.Name)
	err = wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		if local, err = oa.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(context.TODO(), tc.Name, metav1.GetOptions{}); err != nil {
			checkErr = fmt.Errorf("failed to get TidbCluster: %q, %v", tcID, err)
			return false, nil
		}

		components := []v1alpha1.MemberType{
			v1alpha1.PDMemberType,
			v1alpha1.TiKVMemberType,
			v1alpha1.TiDBMemberType,
			v1alpha1.TiDBMemberType,
			v1alpha1.TiFlashMemberType,
			v1alpha1.PumpMemberType,
			v1alpha1.TiCDCMemberType,
		}

		for _, component := range components {
			if err := oa.IsMembersReady(local, component); err != nil {
				checkErr = fmt.Errorf("%s members for tc %q are not ready: %v", component, tcID, err)
				return false, nil
			}
		}

		log.Logf("TidbCluster %q is ready", tcID)
		return true, nil
	})

	if err == wait.ErrWaitTimeout {
		err = checkErr
	}

	return err
}

func (oa *OperatorActions) WaitForTidbClusterInitRandomPassword(tc *v1alpha1.TidbCluster, fw portforward.PortForward, timeout, pollInterval time.Duration) error {
	var checkErr, err error
	ns := tc.Namespace
	tcName := tc.Name
	var localHost string
	var localPort uint16
	var cancel context.CancelFunc
	if fw != nil {
		localHost, localPort, cancel, err = portforward.ForwardOnePort(fw, ns, fmt.Sprintf("svc/%s", controller.TiDBMemberName(tcName)), uint16(tc.Spec.TiDB.GetServicePort()))
		if err != nil {
			return err
		}
		defer cancel()
	}
	err = wait.Poll(timeout, pollInterval, func() (done bool, err error) {
		randomPasswordTc, err := oa.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(context.TODO(), tcName, metav1.GetOptions{})
		if err != nil {
			checkErr = fmt.Errorf("failed to get TidbCluster[%s:%s], error:%v", ns, tcName, err)
			return false, nil
		}
		if randomPasswordTc.Status.TiDB.PasswordInitialized != nil && *randomPasswordTc.Status.TiDB.PasswordInitialized {
			secretName := controller.TiDBInitSecret(randomPasswordTc.Name)
			passwordSecret, err := oa.kubeCli.CoreV1().Secrets(ns).Get(context.TODO(), secretName, metav1.GetOptions{})
			if err != nil {
				checkErr = fmt.Errorf("failed to get secret %s for cluster %s/%s, err: %s", secretName, ns, tcName, err)
				return false, nil
			}
			password := string(passwordSecret.Data[constants.TidbRootKey])
			var dsn string
			if fw != nil {
				dsn = fmt.Sprintf("root:%s@tcp(%s:%d)/?charset=utf8mb4,utf8&multiStatements=true", password, localHost, localPort)
			} else {
				dsn = util.GetDSN(randomPasswordTc, password)
			}
			klog.Errorf("tc[%s:%s] random password dsn:%s", tc.Namespace, tc.Name, dsn)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err = util.OpenDB(ctx, dsn)
			if err != nil {
				if ctx.Err() != nil {
					checkErr = fmt.Errorf("can't connect to the TiDB service of the TiDB cluster [%s:%s], error: %s, context error: %s", ns, randomPasswordTc.Name, err, ctx.Err())
				} else {
					checkErr = fmt.Errorf("can't connect to the TiDB service of the TiDB cluster [%s:%s], error: %s", ns, randomPasswordTc.Name, err)
				}
				return false, err
			}
			return true, nil
		}
		return false, nil
	})
	if err == wait.ErrWaitTimeout {
		return checkErr
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

		components := []v1alpha1.MemberType{
			v1alpha1.DMMasterMemberType,
			v1alpha1.DMWorkerMemberType,
		}

		for _, component := range components {
			if err := oa.IsMembersReady(local, component); err != nil {
				checkErr = fmt.Errorf("%s members for dc %q are not ready: %v", component, dc.Name, err)
				return false, nil
			}
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

func (oa *OperatorActions) WaitForTiDBNGMonitoringReady(tngm *v1alpha1.TidbNGMonitoring, timeout, pollInterval time.Duration) error {
	var checkErr error

	err := wait.PollImmediate(pollInterval, timeout, func() (done bool, err error) {
		ns := tngm.Namespace
		name := tngm.Name
		locator := fmt.Sprintf("%s/%s", tngm.Namespace, tngm.Name)

		local, err := oa.cli.PingcapV1alpha1().TidbNGMonitorings(ns).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			checkErr = fmt.Errorf("get tngm %q failed: %s", locator, err)
			return false, nil
		}

		components := []v1alpha1.MemberType{
			v1alpha1.NGMonitoringMemberType,
		}

		for _, component := range components {
			if err := oa.IsMembersReady(local, component); err != nil {
				checkErr = fmt.Errorf("%s members for tngm %q are not ready: %v", component, locator, err)
				return false, nil
			}
		}

		log.Logf("TiDBNGMonitoring %q: all components are ready", locator)
		return true, nil
	})

	if err == wait.ErrWaitTimeout {
		err = checkErr
	}

	return err
}

// WaitForTidbComponentsReady will wait for tidbcluster components to be ready except those specified in componentsFilter.
func (oa *OperatorActions) WaitForTidbComponentsReady(tc *v1alpha1.TidbCluster, componentsFilter map[v1alpha1.MemberType]struct{}, timeout, pollInterval time.Duration) error {
	if tc == nil {
		return fmt.Errorf("tidbcluster is nil")
	}
	var checkErr, err error
	var local *v1alpha1.TidbCluster
	tcID := fmt.Sprintf("%s/%s", tc.Namespace, tc.Name)
	err = wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		if local, err = oa.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(context.TODO(), tc.Name, metav1.GetOptions{}); err != nil {
			checkErr = fmt.Errorf("failed to get TidbCluster: %q, %v", tcID, err)
			return false, nil
		}

		components := []v1alpha1.MemberType{
			v1alpha1.PDMemberType,
			v1alpha1.TiKVMemberType,
			v1alpha1.TiDBMemberType,
			v1alpha1.TiDBMemberType,
			v1alpha1.TiFlashMemberType,
			v1alpha1.PumpMemberType,
			v1alpha1.TiCDCMemberType,
		}

		for _, component := range components {
			if _, ok := componentsFilter[component]; ok {
				continue
			}

			if err := oa.IsMembersReady(local, component); err != nil {
				checkErr = fmt.Errorf("%s members for tc %q are not ready: %v", component, tcID, err)
				return false, nil
			}
		}

		log.Logf("Components in %q are ready", tcID)
		return true, nil
	})

	if err == wait.ErrWaitTimeout {
		err = checkErr
	}

	return err
}
