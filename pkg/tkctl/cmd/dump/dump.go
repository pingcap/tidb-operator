package dump

import (
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

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/pkg/apis/apps"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/core/helper"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	coreclient "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/internalversion"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/printers"
	"k8s.io/kubernetes/pkg/printers/internalversion"
)

const (
	dumpLongDesc = `
		Dump a tidb cluster information of a specified cluster.
		
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

	tcCli       *versioned.Clientset
	kubeCli     *kubernetes.Clientset
	internalCli *internalclientset.Clientset
	dynamicCli  dynamic.Interface

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
	cmd.MarkFlagRequired("path")
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

	internalCli, err := internalclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	o.internalCli = internalCli

	dynamicCli, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	o.dynamicCli = dynamicCli

	return nil
}

func (o *dumpInfoOptions) Run() error {
	if err := createPathIfNotExist(o.logPath); err != nil {
		return err
	}

	tc, err := o.tcCli.PingcapV1alpha1().
		TidbClusters(o.namespace).
		Get(o.tidbClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// dump tidb cluster object information.
	if err := NewTiDBClusterDescriber(o.internalCli, o.dynamicCli, tc).Dump(o.logPath); err != nil {
		return err
	}

	// dump stateful information by a particular tidb cluster.
	if err := NewTiDBClusterStatefulDescriber(*o.internalCli, tc).Dump(o.logPath); err != nil {
		return err
	}

	podList, err := o.kubeCli.CoreV1().Pods(o.namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s in (%s)", label.InstanceLabelKey, tc.Name, label.ComponentLabelKey,
			strings.Join([]string{label.TiDBLabelVal, label.TiKVLabelVal, label.PDLabelVal}, ",")),
	})
	if err != nil {
		return err
	}

	// dump detail information and logs of pods.
	for _, pod := range podList.Items {
		if err := NewPodDescriber(o.kubeCli, o.internalCli, pod, int64(o.since.Seconds()), o.byteReadLimit).Dump(o.logPath); err != nil {
			return err
		}
	}

	return nil
}

// tidbClusterDescriber generates information about a tidbclusters object.
type tidbClusterDescriber struct {
	internalCli *internalclientset.Clientset
	dynamicCli  dynamic.Interface

	mapping *meta.RESTMapping
	tc      *v1alpha1.TidbCluster
}

// NewTiDBClusterDummper returns a tidbClusterDumper.
func NewTiDBClusterDescriber(internalCli *internalclientset.Clientset, dynamicCli dynamic.Interface, tc *v1alpha1.TidbCluster) *tidbClusterDescriber {
	mapping := &meta.RESTMapping{
		Resource: schema.GroupVersionResource{
			Group:    "pingcap.com",
			Version:  "v1alpha1",
			Resource: "tidbclusters",
		},
	}

	return &tidbClusterDescriber{
		internalCli: internalCli,
		dynamicCli:  dynamicCli,
		mapping:     mapping,
		tc:          tc,
	}
}

// Dump dump the details of tidbclusters from a particular namespace.
func (d *tidbClusterDescriber) Dump(logPath string) error {
	tcd := internalversion.GenericDescriberFor(d.mapping, d.dynamicCli, d.internalCli.Core())

	logFile, err := os.Create(filepath.Join(logPath, fmt.Sprintf("%s-%s-tidbcluster-info.log", d.tc.Name, d.tc.Namespace)))
	if err != nil {
		return err
	}
	defer logFile.Close()

	body, err := tcd.Describe(d.tc.Namespace, d.tc.Name, printers.DescriberSettings{ShowEvents: true})
	if err != nil {
		return err
	}

	return writeString(logFile, body)
}

// tidbClusterStatefulDescriber generates information about a statefulset by a specify tidbcluster.
type tidbClusterStatefulDescriber struct {
	internalCli internalclientset.Clientset
	tc          *v1alpha1.TidbCluster
}

// NewTiDBClusterStatefulDescriber returns a tidbClusterStatefulDescriber
func NewTiDBClusterStatefulDescriber(internalCli internalclientset.Clientset, tc *v1alpha1.TidbCluster) *tidbClusterStatefulDescriber {
	return &tidbClusterStatefulDescriber{
		internalCli: internalCli,
		tc:          tc,
	}
}

// Dump dump the details of statefulsets from a particular tidb cluster.
func (d *tidbClusterStatefulDescriber) Dump(logPath string) error {
	logFile, err := os.Create(filepath.Join(logPath, fmt.Sprintf("%s-%s-statefulsets-info.log", d.tc.Name, d.tc.Namespace)))
	if err != nil {
		return err
	}
	defer logFile.Close()

	for _, sn := range []string{
		controller.TiDBMemberName(d.tc.Name),
		controller.TiKVMemberName(d.tc.Name),
		controller.PDMemberName(d.tc.Name),
	} {
		body, err := d.Describe(sn, printers.DescriberSettings{ShowEvents: true})
		if err != nil {
			return err
		}

		if err = writeString(logFile, body); err != nil {
			return err
		}
	}

	return nil
}

// Describe will attempt to print statefulset object to a string.
func (d *tidbClusterStatefulDescriber) Describe(name string, describerSettings printers.DescriberSettings) (string, error) {
	ps, err := d.internalCli.Apps().StatefulSets(d.tc.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	selector, err := metav1.LabelSelectorAsSelector(ps.Spec.Selector)
	if err != nil {
		return "", err
	}

	pc := d.internalCli.Core().Pods(d.tc.Namespace)
	running, waiting, succeeded, failed, err := getPodStatusForController(pc, selector, ps.UID)
	if err != nil {
		return "", err
	}

	var events *api.EventList
	if describerSettings.ShowEvents {
		events, _ = d.internalCli.Core().Events(d.tc.Namespace).Search(legacyscheme.Scheme, ps)
	}

	return describeStatefulSet(ps, selector, events, running, waiting, succeeded, failed)
}

func describeStatefulSet(ps *apps.StatefulSet, selector labels.Selector, events *api.EventList, running, waiting, succeeded, failed int) (string, error) {
	return readable.TabbedString(func(out io.Writer) error {
		w := internalversion.NewPrefixWriter(out)
		w.Write(internalversion.LEVEL_0, "Name:\t%s\n", ps.ObjectMeta.Name)
		w.Write(internalversion.LEVEL_0, "Namespace:\t%s\n", ps.ObjectMeta.Namespace)
		w.Write(internalversion.LEVEL_0, "CreationTimestamp:\t%s\n", ps.CreationTimestamp.Time.Format(time.RFC1123Z))
		w.Write(internalversion.LEVEL_0, "Selector:\t%s\n", selector)
		readable.PrintLabelsMultiline(w, "Labels", ps.Labels)
		readable.PrintAnnotationsMultiline(w, "Annotations", ps.Annotations)
		w.Write(internalversion.LEVEL_0, "Replicas:\t%d desired | %d total\n", ps.Spec.Replicas, ps.Status.Replicas)
		w.Write(internalversion.LEVEL_0, "Update Strategy:\t%s\n", ps.Spec.UpdateStrategy.Type)
		if ps.Spec.UpdateStrategy.RollingUpdate != nil {
			ru := ps.Spec.UpdateStrategy.RollingUpdate
			if ru.Partition != 0 {
				w.Write(internalversion.LEVEL_1, "Partition:\t%d\n", ru.Partition)
			}
		}

		w.Write(internalversion.LEVEL_0, "Pods Status:\t%d Running / %d Waiting / %d Succeeded / %d Failed\n", running, waiting, succeeded, failed)
		internalversion.DescribePodTemplate(&ps.Spec.Template, w)
		describeVolumeClaimTemplates(ps.Spec.VolumeClaimTemplates, w)
		if events != nil {
			internalversion.DescribeEvents(events, w)
		}

		return nil
	})
}

func describeVolumeClaimTemplates(templates []api.PersistentVolumeClaim, w internalversion.PrefixWriter) {
	if len(templates) == 0 {
		w.Write(readable.LEVEL_0, "Volume Claims:\t<none>\n")
		return
	}
	w.Write(readable.LEVEL_0, "Volume Claims:\n")
	for _, pvc := range templates {
		w.Write(readable.LEVEL_1, "Name:\t%s\n", pvc.Name)
		w.Write(readable.LEVEL_1, "StorageClass:\t%s\n", helper.GetPersistentVolumeClaimClass(&pvc))
		readable.PrintLabelsMultilineWithIndent(w, "  ", "Labels", "\t", pvc.Labels, sets.NewString())
		readable.PrintLabelsMultilineWithIndent(w, "  ", "Annotations", "\t", pvc.Annotations, sets.NewString())
		if capacity, ok := pvc.Spec.Resources.Requests[api.ResourceStorage]; ok {
			w.Write(readable.LEVEL_1, "Capacity:\t%s\n", capacity.String())
		} else {
			w.Write(readable.LEVEL_1, "Capacity:\t%s\n", "<default>")
		}
		w.Write(readable.LEVEL_1, "Access Modes:\t%s\n", pvc.Spec.AccessModes)
	}
}

func getPodStatusForController(c coreclient.PodInterface, selector labels.Selector, uid types.UID) (running, waiting, succeeded, failed int, err error) {
	options := metav1.ListOptions{LabelSelector: selector.String()}
	rcPods, err := c.List(options)
	if err != nil {
		return
	}
	for _, pod := range rcPods.Items {
		controllerRef := metav1.GetControllerOf(&pod)
		// Skip pods that are orphans or owned by other controllers.
		if controllerRef == nil || controllerRef.UID != uid {
			continue
		}
		switch pod.Status.Phase {
		case api.PodRunning:
			running++
		case api.PodPending:
			waiting++
		case api.PodSucceeded:
			succeeded++
		case api.PodFailed:
			failed++
		}
	}
	return
}

// podDescriber generates information about pods and the replication controllers that
// create them.
type podDescriber struct {
	kubeCli     *kubernetes.Clientset
	internalCli *internalclientset.Clientset
	pod         v1.Pod

	sinceSeconds  int64
	byteReadLimit int64
}

// NewPodDescriber returns a podDescriber.
func NewPodDescriber(kubeCli *kubernetes.Clientset, internalCli *internalclientset.Clientset, pod v1.Pod, sinceSeconds int64, byteReadLimit int64) *podDescriber {
	return &podDescriber{
		kubeCli:       kubeCli,
		internalCli:   internalCli,
		pod:           pod,
		sinceSeconds:  sinceSeconds,
		byteReadLimit: byteReadLimit,
	}
}

// Dump dump detail information and logs of pod from a particular pod.
func (d *podDescriber) Dump(logPath string) error {
	path := fmt.Sprintf("%s%cpodsinfo", logPath, filepath.Separator)
	if err := createPathIfNotExist(path); err != nil {
		return err
	}
	if err := d.DumpDetail(path); err != nil {
		return err
	}

	path = fmt.Sprintf("%s%clogs", logPath, filepath.Separator)
	if err := createPathIfNotExist(path); err != nil {
		return err
	}
	if err := d.DumpLog(path); err != nil {
		return err
	}

	return nil
}

// Dump dump the details of a named Pod from a particular namespace.
func (d *podDescriber) DumpDetail(logPath string) error {
	pd := internalversion.PodDescriber{d.internalCli}
	podDetail, err := pd.Describe(d.pod.Namespace, d.pod.Name, printers.DescriberSettings{ShowEvents: true})
	if err != nil {
		return err
	}

	logFile, err := os.Create(filepath.Join(logPath, fmt.Sprintf("%s-%s.log", d.pod.Name, d.pod.Namespace)))
	if err != nil {
		return err
	}
	defer logFile.Close()

	if err := writeString(logFile, podDetail); err != nil {
		return err
	}

	return nil

}

// Dump dump dump the logs for the last terminated container and current running container. If info about the container is not available then a specific
// error is returned.
func (d *podDescriber) DumpLog(logPath string) error {
	for _, c := range d.pod.Spec.Containers {
		if err := d.DumpContainerLog(filepath.Join(logPath, fmt.Sprintf("%s-%s-%s.log", d.pod.Name, c.Name, d.pod.Namespace)), c.Name, false); err != nil {
			return err
		}

		// ignore error for previous container log
		d.DumpContainerLog(filepath.Join(logPath, fmt.Sprintf("%s-%s-%s-p.log", d.pod.Name, c.Name, d.pod.Namespace)), c.Name, true)
	}

	return nil
}

// DumpContainerLog dump logs for particular pod and container. Previous indicates to read archived logs created by log rotation or container crash
func (d *podDescriber) DumpContainerLog(logPath string, containerName string, previous bool) error {
	body, err := getLogStream(d.kubeCli, d.pod, mapToLogOptions(containerName, d.sinceSeconds, d.byteReadLimit, previous))
	if err != nil {
		return err
	}
	defer body.Close()

	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer logFile.Close()

	written, err := io.Copy(logFile, body)
	if err != nil {
		return err
	}

	if written == 0 {
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
