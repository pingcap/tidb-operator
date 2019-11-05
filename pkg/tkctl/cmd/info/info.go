// Copyright 2019. PingCAP, Inc.
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

package info

import (
	"fmt"
	"io"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/pingcap/tidb-operator/pkg/tkctl/readable"
	"github.com/pingcap/tidb-operator/pkg/tkctl/util"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

const (
	infoLongDesc = `
		Get tidb cluster information of a specified cluster.

		You can omit --tidbcluster=<name> option by running 'tkc use <clusterName>',
`
	infoExample = `
		# get current tidb cluster info (set by tkc use)
		tkctl info

		# get specified tidb cluster info
		tkctl info -t another-cluster
`
	infoUsage = `expected 'info -t CLUSTER_NAME' for the info command or 
using 'tkctl use' to set tidb cluster first.
`
)

// InfoOptions contains the input to the list command.
type InfoOptions struct {
	TidbClusterName string
	Namespace       string

	TcCli   *versioned.Clientset
	KubeCli *kubernetes.Clientset

	genericclioptions.IOStreams
}

// NewInfoOptions returns a InfoOptions
func NewInfoOptions(streams genericclioptions.IOStreams) *InfoOptions {
	return &InfoOptions{
		IOStreams: streams,
	}
}

// NewCmdInfo creates the info command which info the tidb component detail
func NewCmdInfo(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewInfoOptions(streams)

	cmd := &cobra.Command{
		Use:     "info",
		Short:   "Show tidb cluster information.",
		Long:    infoLongDesc,
		Example: infoExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(tkcContext, cmd, args))
			cmdutil.CheckErr(o.Run())
		},
		SuggestFor: []string{"inspect", "explain"},
	}

	return cmd
}

func (o *InfoOptions) Complete(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {

	clientConfig, err := tkcContext.ToTkcClientConfig()
	if err != nil {
		return err
	}

	if tidbClusterName, ok := clientConfig.TidbClusterName(); ok {
		o.TidbClusterName = tidbClusterName
	} else {
		return cmdutil.UsageErrorf(cmd, infoUsage)
	}

	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return err
	}
	o.Namespace = namespace

	restConfig, err := clientConfig.RestConfig()
	if err != nil {
		return err
	}
	tcCli, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	o.TcCli = tcCli
	kubeCli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	o.KubeCli = kubeCli

	return nil
}

func (o *InfoOptions) Run() error {

	tc, err := o.TcCli.PingcapV1alpha1().
		TidbClusters(o.Namespace).
		Get(o.TidbClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	svcName := util.GetTidbServiceName(tc.Name)
	svc, err := o.KubeCli.CoreV1().Services(o.Namespace).Get(svcName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	podList, err := o.KubeCli.CoreV1().Pods(o.Namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s", label.InstanceLabelKey, tc.Name, label.ComponentLabelKey, "tidb"),
	})
	if err != nil {
		return err
	}
	msg, err := renderTidbCluster(tc, svc, podList)
	if err != nil {
		return err
	}
	fmt.Fprint(o.Out, msg)
	return nil
}

// go template is lacking type checking and hard to maintain, in this
// case we just render manually
func renderTidbCluster(tc *v1alpha1.TidbCluster, svc *v1.Service, podList *v1.PodList) (string, error) {
	return readable.TabbedString(func(out io.Writer) error {
		w := readable.NewPrefixWriter(out)
		w.WriteLine(readable.LEVEL_0, "Name:\t%s", tc.Name)
		w.WriteLine(readable.LEVEL_0, "Namespace:\t%s", tc.Namespace)
		w.WriteLine(readable.LEVEL_0, "CreationTimestamp:\t%s", tc.CreationTimestamp)
		w.WriteLine(readable.LEVEL_0, "Overview:")
		{
			w.WriteLine(readable.LEVEL_1, "\tPhase\tReady\tDesired\tCPU\tMemory\tStorage\tVersion")
			w.WriteLine(readable.LEVEL_1, "\t-----\t-----\t-------\t---\t------\t-------\t-------")
			w.Write(readable.LEVEL_1, "PD:\t")
			{
				w.Write(readable.LEVEL_0, "%s\t", tc.Status.PD.Phase)
				w.Write(readable.LEVEL_0, "%d\t", tc.Status.PD.StatefulSet.ReadyReplicas)
				w.Write(readable.LEVEL_0, "%d\t", tc.Status.PD.StatefulSet.Replicas)
				w.Write(readable.LEVEL_0, "%s\t", tc.Spec.PD.Requests.CPU)
				w.Write(readable.LEVEL_0, "%s\t", tc.Spec.PD.Requests.Memory)
				w.Write(readable.LEVEL_0, "%s\t", tc.Spec.PD.Requests.Storage)
				w.Write(readable.LEVEL_0, "%s\t\n", tc.Spec.PD.Image)
			}
			w.Write(readable.LEVEL_1, "TiKV:\t")
			{
				w.Write(readable.LEVEL_0, "%s\t", tc.Status.TiKV.Phase)
				w.Write(readable.LEVEL_0, "%d\t", tc.Status.TiKV.StatefulSet.ReadyReplicas)
				w.Write(readable.LEVEL_0, "%d\t", tc.Status.TiKV.StatefulSet.Replicas)
				w.Write(readable.LEVEL_0, "%s\t", tc.Spec.TiKV.Requests.CPU)
				w.Write(readable.LEVEL_0, "%s\t", tc.Spec.TiKV.Requests.Memory)
				w.Write(readable.LEVEL_0, "%s\t", tc.Spec.TiKV.Requests.Storage)
				w.Write(readable.LEVEL_0, "%s\t\n", tc.Spec.TiKV.Image)
			}
			w.Write(readable.LEVEL_1, "TiDB\t")
			{
				w.Write(readable.LEVEL_0, "%s\t", tc.Status.TiDB.Phase)
				w.Write(readable.LEVEL_0, "%d\t", tc.Status.TiDB.StatefulSet.ReadyReplicas)
				w.Write(readable.LEVEL_0, "%d\t", tc.Status.TiDB.StatefulSet.Replicas)
				w.Write(readable.LEVEL_0, "%s\t", tc.Spec.TiDB.Requests.CPU)
				w.Write(readable.LEVEL_0, "%s\t", tc.Spec.TiDB.Requests.Memory)
				w.Write(readable.LEVEL_0, "%s\t", tc.Spec.TiDB.Requests.Storage)
				w.Write(readable.LEVEL_0, "%s\t\n", tc.Spec.TiDB.Image)
			}
		}
		w.WriteLine(readable.LEVEL_0, "Endpoints(%s):", svc.Spec.Type)
		if svc.Spec.Type == v1.ServiceTypeNodePort {
			var nodePort int32
			for _, port := range svc.Spec.Ports {
				// FIXME: magic name
				if port.Name == "mysql-client" {
					nodePort = port.NodePort
					break
				}
			}
			if nodePort > 0 {
				for _, pod := range podList.Items {
					if pod.Status.Phase == v1.PodRunning {
						w.WriteLine(readable.LEVEL_1, "- %s:%d", pod.Status.HostIP, nodePort)
					}
				}
			} else {
				w.WriteLine(readable.LEVEL_1, "no suitable port")
			}
		} else {
			w.WriteLine(readable.LEVEL_1, "Cluster IP:\t%s", svc.Spec.ClusterIP)
		}
		return nil
	})
}
