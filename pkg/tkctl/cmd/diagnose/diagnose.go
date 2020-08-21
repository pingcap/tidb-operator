// Copyright 2019 PingCAP, Inc.
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

package diagnose

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/tkctl/readable"

	"k8s.io/kubernetes/pkg/api/legacyscheme"

	"k8s.io/kubernetes/pkg/printers"
	printersinternal "k8s.io/kubernetes/pkg/printers/internalversion"

	"github.com/ghodss/yaml"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/spf13/cobra"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubernetes/pkg/apis/apps"
	api "k8s.io/kubernetes/pkg/apis/core"
)

const (
	diagnoseLongDesc = `
		Export a tidb cluster diagnostic information of a specified cluster.
		
		You may omit --tidbcluster option by running 'tkc use <clusterName>'.
`
	diagnoseExample = `
		# specify a tidb cluster to use
		tkctl diagnose

		# diagnose specify tidb cluster information
		tkctl diagnose -t demo-cluster
`
	diagnoseUsage = `expected 'diagnose -t CLUSTER_NAME' for the diagnose command or
using 'tkctl use to set tidb cluster first.'`
)

type diagnoseInfoOptions struct {
	namespace       string
	tidbClusterName string

	tcCli   *versioned.Clientset
	kubeCli *kubernetes.Clientset

	listOptions metav1.ListOptions

	logPath       string
	since         time.Duration
	byteReadLimit int64
	printer       printers.ResourcePrinter
	tidbPrinter   printers.ResourcePrinter

	genericclioptions.IOStreams
}

// NewDiagnoseInfoOptions returns a diagnoseInfoOptions.
func NewDiagnoseInfoOptions(streams genericclioptions.IOStreams) *diagnoseInfoOptions {
	return &diagnoseInfoOptions{
		IOStreams: streams,
	}
}

// NewCmdDiagnoseInfo creates the diagnose command to export diagnose information of specify tidb cluster.
func NewCmdDiagnoseInfo(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewDiagnoseInfoOptions(streams)

	cmd := &cobra.Command{
		Use:     "diagnose",
		Short:   "export a tidb cluster diagnostic information",
		Long:    diagnoseLongDesc,
		Example: diagnoseExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(tkcContext, cmd, args))
			cmdutil.CheckErr(o.Run())
		},
	}

	cmd.Flags().StringVar(&o.logPath, "path", "", "The log path to dump.")
	cmd.Flags().DurationVar(&o.since, "since", time.Duration(1)*time.Hour, "Return logs newer than a relative duration like 1m, or 3h.")
	cmd.Flags().Int64Var(&o.byteReadLimit, "byteReadLimit", 500000, "The maximum number of bytes dump log.")
	cmdutil.CheckErr(cmd.MarkFlagRequired("path"))
	return cmd
}

func (o *diagnoseInfoOptions) Complete(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {
	clientConfig, err := tkcContext.ToTkcClientConfig()
	if err != nil {
		return err
	}

	if tidbClusterName, ok := clientConfig.TidbClusterName(); ok {
		o.tidbClusterName = tidbClusterName
	} else {
		return cmdutil.UsageErrorf(cmd, diagnoseUsage)
	}

	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return err
	}
	o.namespace = namespace

	restConfig, err := clientConfig.RestConfig()
	if err != nil {
		return err
	}

	tcCli, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	o.tcCli = tcCli

	kubeCli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	o.kubeCli = kubeCli

	o.printer = NewPrinter()

	tidbPrinter, err := NewTiDBPrinter()
	if err != nil {
		return err
	}
	o.tidbPrinter = tidbPrinter

	o.listOptions = metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s in (%s)", label.InstanceLabelKey, o.tidbClusterName, label.ComponentLabelKey,
			strings.Join([]string{label.TiDBLabelVal, label.TiKVLabelVal, label.PDLabelVal}, ",")),
	}

	return nil
}

func (o *diagnoseInfoOptions) Run() error {
	if err := createPathIfNotExist(o.logPath); err != nil {
		return err
	}

	resourceFile, err := os.Create(filepath.Join(o.logPath, "resources"))
	if err != nil {
		return err
	}
	defer func() {
		cmdutil.CheckErr(resourceFile.Close())
	}()

	rWriter := bufio.NewWriter(resourceFile)
	defer func() {
		cmdutil.CheckErr(rWriter.Flush())
	}()

	tc, err := o.tcCli.PingcapV1alpha1().
		TidbClusters(o.namespace).
		Get(o.tidbClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	tc.SetGroupVersionKind(controller.ControllerKind)

	// dump tidb cluster object information.
	if err := NewTiDBClusterDumper(tc, o.tidbPrinter).Dump(o.logPath, rWriter); err != nil {
		return err
	}

	// dump stateful information by a particular tidb cluster.
	if err := NewTiDBClusterStatefulDumper(tc, o.kubeCli, o.printer).Dump(o.logPath, rWriter); err != nil {
		return err
	}

	// dump pvc information by a particular tidb cluster.
	if err := NewPvcDumper(o.kubeCli, tc, o.listOptions, o.printer).Dump(o.logPath, rWriter); err != nil {
		return err
	}

	// dump services information by a particular tidb cluster.
	if err := NewSvcDumper(o.kubeCli, tc, o.listOptions, o.printer).Dump(o.logPath, rWriter); err != nil {
		return err
	}

	// dump configmaps information by a particular tidb cluster.
	if err := NewConfigMapDumper(o.kubeCli, tc, o.listOptions, o.printer).Dump(o.logPath, rWriter); err != nil {
		return err
	}

	podList, err := o.kubeCli.CoreV1().Pods(o.namespace).List(o.listOptions)
	if err != nil {
		return err
	}

	if _, err := rWriter.Write([]byte("----------------pods---------------\n")); err != nil {
		return err
	}

	// dump detail information and logs of pods.
	pods := api.PodList{}
	for _, pod := range podList.Items {
		if err := NewPodDumper(o.kubeCli, pod, int64(o.since.Seconds()), o.byteReadLimit).Dump(o.logPath, rWriter); err != nil {
			return err
		}

		p, err := convertToInternalObj(&pod, "")
		if err != nil {
			return err
		}
		pods.Items = append(pods.Items, *(p.(*api.Pod)))
	}

	return o.printer.PrintObj(&pods, rWriter)
}

// tidbClusterDumper generates information about a tidbclusters object.
type tidbClusterDumper struct {
	tc      *v1alpha1.TidbCluster
	printer printers.ResourcePrinter
}

// NewTiDBClusterDummper returns a tidbClusterDumper.
func NewTiDBClusterDumper(tc *v1alpha1.TidbCluster, printer printers.ResourcePrinter) *tidbClusterDumper {
	return &tidbClusterDumper{
		tc:      tc,
		printer: printer,
	}
}

// Dump dumps the details of tidbclusters from a particular namespace.
func (d *tidbClusterDumper) Dump(logPath string, resourceWriter io.Writer) error {
	logFile, err := os.Create(filepath.Join(logPath, fmt.Sprintf("%s-%s-tidbcluster-info.yaml", d.tc.Name, d.tc.Namespace)))
	if err != nil {
		return err
	}
	defer func() {
		cmdutil.CheckErr(logFile.Close())
	}()

	if _, err := resourceWriter.Write([]byte("----------------tidbclusters---------------\n")); err != nil {
		return err
	}

	if err = d.printer.PrintObj(d.tc, resourceWriter); err != nil {
		return err
	}

	data, err := yaml.Marshal(d.tc)
	if err != nil {
		return err
	}

	return writeString(logFile, string(data))
}

// tidbClusterStatefulDumper generates information about a statefulset by a specify tidbcluster.
type tidbClusterStatefulDumper struct {
	kubeCli *kubernetes.Clientset
	tc      *v1alpha1.TidbCluster
	printer printers.ResourcePrinter
}

// NewTiDBClusterStatefulDumper returns a tidbClusterStatefulDumper
func NewTiDBClusterStatefulDumper(tc *v1alpha1.TidbCluster, kubeCli *kubernetes.Clientset, printer printers.ResourcePrinter) *tidbClusterStatefulDumper {
	return &tidbClusterStatefulDumper{
		tc:      tc,
		kubeCli: kubeCli,
		printer: printer,
	}
}

// Dump dumps the details of statefulsets from a particular tidb cluster.
func (d *tidbClusterStatefulDumper) Dump(logPath string, resourceWriter io.Writer) error {
	logFile, err := os.Create(filepath.Join(logPath, fmt.Sprintf("%s-%s-statefulsets-info.yaml", d.tc.Name, d.tc.Namespace)))
	if err != nil {
		return err
	}

	defer func() {
		cmdutil.CheckErr(logFile.Close())
	}()

	if _, err := resourceWriter.Write([]byte("----------------statefulset---------------\n")); err != nil {
		return err
	}

	sts := apps.StatefulSetList{}
	for _, sn := range []string{
		controller.TiDBMemberName(d.tc.Name),
		controller.TiKVMemberName(d.tc.Name),
		controller.PDMemberName(d.tc.Name),
	} {
		ps, err := d.kubeCli.AppsV1().StatefulSets(d.tc.Namespace).Get(sn, metav1.GetOptions{})
		if err != nil {
			return err
		}
		ps.SetGroupVersionKind(appv1.SchemeGroupVersion.WithKind("StatefulSet"))

		if err = writeString(logFile, "#"+sn+"\n"); err != nil {
			return err
		}

		body, err := yaml.Marshal(ps)
		if err != nil {
			return err
		}

		if err = writeString(logFile, string(body)); err != nil {
			return err
		}

		if err = writeString(logFile, "\n"); err != nil {
			return err
		}

		s, err := convertToInternalObj(ps, "apps")
		if err != nil {
			return err
		}
		sts.Items = append(sts.Items, *(s.(*apps.StatefulSet)))
	}

	return d.printer.PrintObj(&sts, resourceWriter)
}

// pvcDumper generates information about pvc.
type pvcDumper struct {
	kubeCli *kubernetes.Clientset
	tc      *v1alpha1.TidbCluster
	options metav1.ListOptions
	printer printers.ResourcePrinter
}

// NewPvcDumper returns a pvcDumper.
func NewPvcDumper(kubeCli *kubernetes.Clientset, tc *v1alpha1.TidbCluster, options metav1.ListOptions, printer printers.ResourcePrinter) *pvcDumper {
	return &pvcDumper{
		tc:      tc,
		kubeCli: kubeCli,
		options: options,
		printer: printer,
	}
}

// Dump dumps the details of pvc from a particular tidb cluster.
func (d *pvcDumper) Dump(logPath string, resourceWriter io.Writer) error {
	logFile, err := os.Create(filepath.Join(logPath, fmt.Sprintf("%s-%s-pvc-info.yaml", d.tc.Name, d.tc.Namespace)))
	if err != nil {
		return err
	}

	defer func() {
		cmdutil.CheckErr(logFile.Close())
	}()

	if _, err := resourceWriter.Write([]byte("----------------pvc---------------\n")); err != nil {
		return err
	}

	pvcList, err := d.kubeCli.CoreV1().PersistentVolumeClaims(d.tc.Namespace).List(d.options)
	if err != nil {
		return err
	}

	pvcs := api.PersistentVolumeClaimList{}
	for _, pvc := range pvcList.Items {
		pvc.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"))

		body, err := yaml.Marshal(pvc)
		if err != nil {
			return err
		}
		if err = writeString(logFile, string(body)); err != nil {
			return err
		}

		s, err := convertToInternalObj(&pvc, "")
		if err != nil {
			return err
		}

		pvcs.Items = append(pvcs.Items, *(s.(*api.PersistentVolumeClaim)))
	}

	return d.printer.PrintObj(&pvcs, resourceWriter)
}

// svcDumper generates information about service.
type svcDumper struct {
	kubeCli *kubernetes.Clientset
	tc      *v1alpha1.TidbCluster
	options metav1.ListOptions
	printer printers.ResourcePrinter
}

// NewSvcDumper returns a pvcDumper.
func NewSvcDumper(kubeCli *kubernetes.Clientset, tc *v1alpha1.TidbCluster, options metav1.ListOptions, printer printers.ResourcePrinter) *svcDumper {
	return &svcDumper{
		tc:      tc,
		kubeCli: kubeCli,
		options: options,
		printer: printer,
	}
}

// Dump dumps the details of service from a particular tidb cluster.
func (d *svcDumper) Dump(logPath string, resourceWriter io.Writer) error {
	logFile, err := os.Create(filepath.Join(logPath, fmt.Sprintf("%s-%s-svc-info.yaml", d.tc.Name, d.tc.Namespace)))
	if err != nil {
		return err
	}

	defer func() {
		cmdutil.CheckErr(logFile.Close())
	}()

	if _, err := resourceWriter.Write([]byte("----------------service---------------\n")); err != nil {
		return err
	}

	svcList, err := d.kubeCli.CoreV1().Services(d.tc.Namespace).List(d.options)
	if err != nil {
		return err
	}

	svcs := api.ServiceList{}
	for _, svc := range svcList.Items {
		svc.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Service"))

		body, err := yaml.Marshal(svc)
		if err != nil {
			return err
		}
		if err = writeString(logFile, string(body)); err != nil {
			return err
		}

		s, err := convertToInternalObj(&svc, "")
		if err != nil {
			return err
		}

		svcs.Items = append(svcs.Items, *(s.(*api.Service)))
	}

	return d.printer.PrintObj(&svcs, resourceWriter)
}

// configMapDumper generates information about service.
type configMapDumper struct {
	kubeCli *kubernetes.Clientset
	tc      *v1alpha1.TidbCluster
	options metav1.ListOptions
	printer printers.ResourcePrinter
}

// NewConfigMapDumper returns a pvcDumper.
func NewConfigMapDumper(kubeCli *kubernetes.Clientset, tc *v1alpha1.TidbCluster, options metav1.ListOptions, printer printers.ResourcePrinter) *configMapDumper {
	return &configMapDumper{
		tc:      tc,
		kubeCli: kubeCli,
		options: options,
		printer: printer,
	}
}

// Dump dumps the details of configmap from a particular tidb cluster.
func (d *configMapDumper) Dump(logPath string, resourceWriter io.Writer) error {
	logFile, err := os.Create(filepath.Join(logPath, fmt.Sprintf("%s-%s-configmap-info.yaml", d.tc.Name, d.tc.Namespace)))
	if err != nil {
		return err
	}

	defer func() {
		cmdutil.CheckErr(logFile.Close())
	}()

	if _, err := resourceWriter.Write([]byte("----------------configmap---------------\n")); err != nil {
		return err
	}

	cfgList, err := d.kubeCli.CoreV1().ConfigMaps(d.tc.Namespace).List(d.options)
	if err != nil {
		return err
	}

	cfgs := api.ConfigMapList{}
	for _, cfg := range cfgList.Items {
		cfg.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))

		body, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		if err = writeString(logFile, string(body)); err != nil {
			return err
		}

		s, err := convertToInternalObj(&cfg, "")
		if err != nil {
			return err
		}

		cfgs.Items = append(cfgs.Items, *(s.(*api.ConfigMap)))
	}

	return d.printer.PrintObj(&cfgs, resourceWriter)
}

// podDumper generates information about pods and the replication controllers that
// create them.
type podDumper struct {
	kubeCli *kubernetes.Clientset
	pod     v1.Pod

	sinceSeconds  int64
	byteReadLimit int64
}

// NewPodDumper returns a podDumper.
func NewPodDumper(kubeCli *kubernetes.Clientset, pod v1.Pod, sinceSeconds int64, byteReadLimit int64) *podDumper {
	pod.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Pod"))
	return &podDumper{
		kubeCli:       kubeCli,
		pod:           pod,
		sinceSeconds:  sinceSeconds,
		byteReadLimit: byteReadLimit,
	}
}

// Dump dumps detail information and logs of pod from a particular pod.
func (d *podDumper) Dump(logPath string, resourceWriter io.Writer) error {
	path := fmt.Sprintf("%s%cpod-infos", logPath, filepath.Separator)
	if err := createPathIfNotExist(path); err != nil {
		return err
	}

	if err := d.DumpDetail(path); err != nil {
		return err
	}

	path = fmt.Sprintf("%s%cpod-logs", logPath, filepath.Separator)
	if err := createPathIfNotExist(path); err != nil {
		return err
	}

	return d.DumpLog(path)
}

// Dump dumps the details of a named Pod from a particular namespace.
func (d *podDumper) DumpDetail(logPath string) error {
	logFile, err := os.Create(filepath.Join(logPath, fmt.Sprintf("%s-%s.yaml", d.pod.Name, d.pod.Namespace)))
	if err != nil {
		return err
	}
	defer func() {
		cmdutil.CheckErr(logFile.Close())
	}()

	body, err := yaml.Marshal(d.pod)
	if err != nil {
		return err
	}

	return writeString(logFile, string(body))
}

// Dump dumps the logs for the last terminated container and current running container. If info about the container is not available then a specific
// error is returned.
func (d *podDumper) DumpLog(logPath string) error {
	for _, c := range d.pod.Spec.Containers {
		if err := d.DumpContainerLog(filepath.Join(logPath, fmt.Sprintf("%s-%s-%s.log", d.pod.Name, c.Name, d.pod.Namespace)), c.Name, false); err != nil {
			return err
		}

		d.DumpContainerLog(filepath.Join(logPath, fmt.Sprintf("%s-%s-%s-p.log", d.pod.Name, c.Name, d.pod.Namespace)), c.Name, true)
	}

	return nil
}

// DumpContainerLog dumps logs for particular pod and container. Previous indicates to read archived logs created by log rotation or container crash
func (d *podDumper) DumpContainerLog(logPath string, containerName string, previous bool) error {
	body, err := getLogStream(d.kubeCli, d.pod, mapToLogOptions(containerName, d.sinceSeconds, d.byteReadLimit, previous))
	if err != nil {
		return err
	}
	defer body.Close()

	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer func() {
		cmdutil.CheckErr(logFile.Close())
	}()

	written, err := io.Copy(logFile, body)
	if err != nil {
		return err
	}

	if written == 0 {
		// ignore error
		os.Remove(logPath)
	}

	return nil
}

// mapToLogOptions Maps the log selection to the corresponding api object
// Read limits are set to avoid out of memory issues
func mapToLogOptions(container string, sinceSeconds int64, byteReadLimit int64, previous bool) *v1.PodLogOptions {
	return &v1.PodLogOptions{
		Container:    container,
		Follow:       false,
		Previous:     previous,
		Timestamps:   true,
		SinceSeconds: &sinceSeconds,
		LimitBytes:   &byteReadLimit,
	}
}

// getLogStream returns a stream to the log file which can be piped directly to the response. This avoids out of memory
// issues. Previous indicates to read archived logs created by log rotation or container crash
func getLogStream(kubeCli *kubernetes.Clientset, pod v1.Pod, logOptions *v1.PodLogOptions) (io.ReadCloser, error) {
	return kubeCli.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, logOptions).Stream()
}

// NewPrinter creates a common HumanReadablePrinter.
func NewPrinter() printers.ResourcePrinter {
	printer := printers.NewTablePrinter(printers.PrintOptions{
		WithKind:      false,
		Wide:          true,
		WithNamespace: false,
	})
	// AddHandlers adds print handlers for default Kubernetes types dealing with internal versions.
	tableGenerator := printers.NewTableGenerator().With(printersinternal.AddHandlers)
	return readable.NewLocalPrinter(printer, tableGenerator, printers.GenerateOptions{
		Wide: true,
	})
}

// NewTiDBPrinter creates a TiDB object printer
func NewTiDBPrinter() (printers.ResourcePrinter, error) {
	printFlags := &readable.PrintFlags{
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
		OutputFormat:       "wide",
	}
	return printFlags.ToPrinter(false, false)
}

func writeString(w io.Writer, body string) error {
	_, err := io.WriteString(w, body)
	return err
}

func createPathIfNotExist(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

// convertToInternalObj will convert v1 object to internal version object.
func convertToInternalObj(obj runtime.Object, group string) (runtime.Object, error) {
	sch := legacyscheme.Scheme

	p, err := sch.UnsafeConvertToVersion(obj, schema.GroupVersion{Group: group, Version: runtime.APIVersionInternal})
	if err != nil {
		return nil, err
	}
	return p, nil
}
