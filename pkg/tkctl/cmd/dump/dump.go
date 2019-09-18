package dump

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/pingcap/tidb-operator/pkg/tkctl/readable"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	dumpLongDesc = `
		Export a tidb cluster diagnostic information of a specified cluster.
		
		You may omit --tidbcluster option by running 'tkc use <clusterName>'.
`
	dumpExample = `
		# specify a tidb cluster to use
		tkctl dump

		# dump specify tidb cluster information
		tkctl dump -t demo-cluster
`
	dumpUsage = `expected 'dump -t CLUSTER_NAME' for the dump command or
using 'tkctl use to set tidb cluster first.'`
)

type dumpInfoOptions struct {
	kubeContext     string
	namespace       string
	tidbClusterName string

	tcCli   *versioned.Clientset
	kubeCli *kubernetes.Clientset

	logPath       string
	since         time.Duration
	byteReadLimit int64

	genericclioptions.IOStreams
}

// NewDumpInfoOptions returns a DumpInfoOptions
func NewDumpInfoOptions(streams genericclioptions.IOStreams) *dumpInfoOptions {
	return &dumpInfoOptions{
		IOStreams: streams,
	}
}

// NewCmdDumpInfo creates the dump command to dump specify tidb cluster information
func NewCmdDumpInfo(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewDumpInfoOptions(streams)

	cmd := &cobra.Command{
		Use:     "dump",
		Short:   "dump tidb cluster log and kubernetes object information",
		Long:    dumpLongDesc,
		Example: dumpExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(tkcContext, cmd, args))
			cmdutil.CheckErr(o.Run())
		},
	}

	cmd.Flags().StringVar(&o.logPath, "path", "", "The log path to dump.")
	cmd.Flags().DurationVar(&o.since, "since", 3600000000000, "Return logs newer than a relative duration like 1m, or 3h.")
	cmd.Flags().Int64Var(&o.byteReadLimit, "byteReadLimit", 500000, "The maximum number of bytes dump log.")
	cmdutil.CheckErr(cmd.MarkFlagRequired("path"))
	return cmd
}

func (o *dumpInfoOptions) Complete(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {
	clientConfig, err := tkcContext.ToTkcClientConfig()
	if err != nil {
		return err
	}

	if tidbClusterName, ok := clientConfig.TidbClusterName(); ok {
		o.tidbClusterName = tidbClusterName
	} else {
		return cmdutil.UsageErrorf(cmd, dumpUsage)
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

	return nil
}

func (o *dumpInfoOptions) Run() error {
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
	if err := NewTiDBClusterDumper(tc).Dump(o.logPath, rWriter); err != nil {
		return err
	}

	// dump stateful information by a particular tidb cluster.
	if err := NewTiDBClusterStatefulDumper(tc, o.kubeCli).Dump(o.logPath, rWriter); err != nil {
		return err
	}

	podList, err := o.kubeCli.CoreV1().Pods(o.namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s in (%s)", label.InstanceLabelKey, tc.Name, label.ComponentLabelKey,
			strings.Join([]string{label.TiDBLabelVal, label.TiKVLabelVal, label.PDLabelVal}, ",")),
	})
	if err != nil {
		return err
	}

	if _, err := rWriter.Write([]byte("----------------pods---------------\n")); err != nil {
		return err
	}

	// dump detail information and logs of pods.
	for _, pod := range podList.Items {
		if err := NewPodDumper(o.kubeCli, pod, int64(o.since.Seconds()), o.byteReadLimit).Dump(o.logPath, rWriter); err != nil {
			return err
		}
	}

	return nil
}

// tidbClusterDumper generates information about a tidbclusters object.
type tidbClusterDumper struct {
	tc *v1alpha1.TidbCluster
}

// NewTiDBClusterDummper returns a tidbClusterDumper.
func NewTiDBClusterDumper(tc *v1alpha1.TidbCluster) *tidbClusterDumper {
	return &tidbClusterDumper{
		tc: tc,
	}
}

// Dump dump the details of tidbclusters from a particular namespace.
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

	if err = DumpObj(d.tc, resourceWriter); err != nil {
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
}

// NewTiDBClusterStatefulDumper returns a tidbClusterStatefulDumper
func NewTiDBClusterStatefulDumper(tc *v1alpha1.TidbCluster, kubeCli *kubernetes.Clientset) *tidbClusterStatefulDumper {
	return &tidbClusterStatefulDumper{
		tc:      tc,
		kubeCli: kubeCli,
	}
}

// Dump dump the details of statefulsets from a particular tidb cluster.
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

	for _, sn := range []string{
		controller.TiDBMemberName(d.tc.Name),
		controller.TiKVMemberName(d.tc.Name),
		controller.PDMemberName(d.tc.Name),
	} {
		ps, err := d.kubeCli.AppsV1().StatefulSets(d.tc.Namespace).Get(sn, metav1.GetOptions{})
		if err != nil {
			return err
		}
		ps.SetGroupVersionKind(apps.SchemeGroupVersion.WithKind("StatefulSet"))

		if err = writeString(logFile, "#"+sn+"\n"); err != nil {
			return err
		}

		if err = DumpObj(ps, resourceWriter); err != nil {
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
	}

	return nil
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

// Dump dump detail information and logs of pod from a particular pod.
func (d *podDumper) Dump(logPath string, resourceWriter io.Writer) error {
	if err := DumpObj(&d.pod, resourceWriter); err != nil {
		return err
	}

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

// Dump dump the details of a named Pod from a particular namespace.
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

// Dump dump dump the logs for the last terminated container and current running container. If info about the container is not available then a specific
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

// DumpContainerLog dump logs for particular pod and container. Previous indicates to read archived logs created by log rotation or container crash
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

// DumpObj print wide output to the file.
func DumpObj(obj runtime.Object, writer io.Writer) error {
	// create kubectl HumanReadablePrinter
	printFlags := &readable.PrintFlags{
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
		OutputFormat:       "wide",
	}
	printer, err := printFlags.ToPrinter(false, false)
	if err != nil {
		return err
	}

	return printer.PrintObj(obj, writer)
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
