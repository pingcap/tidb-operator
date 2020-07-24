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

package upinfo

import (
	"fmt"
	"io"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/pingcap/tidb-operator/pkg/tkctl/readable"
	tkctlUtil "github.com/pingcap/tidb-operator/pkg/tkctl/util"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/spf13/cobra"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

const (
	upinfoLongDesc = `
		Get tidb cluster component upgrade info.

		You can omit --tidbcluster=<name> option by running 'tkc use <clusterName>',
`
	upinfoExample = `
		# get current tidb cluster info (set by tkctl user)
		tkctl upinfo

		# get specified tidb cluster component upgrade info
		tkctl upinfo -t another-cluster
`
	infoUsage = `expected 'upinfo -t CLUSTER_NAME' for the upinfo command or 
using 'tkctl use' to set tidb cluster first.
`
	UPDATED  = "updated"
	UPDATING = "updating"
	WAITING  = "waiting"
)

// UpInfoOptions contains the input to the list command.
type UpInfoOptions struct {
	TidbClusterName string
	Namespace       string

	TcCli   *versioned.Clientset
	KubeCli *kubernetes.Clientset

	genericclioptions.IOStreams
}

// NewUpInfoOptions returns a UpInfoOptions
func NewUpInfoOptions(streams genericclioptions.IOStreams) *UpInfoOptions {
	return &UpInfoOptions{
		IOStreams: streams,
	}
}

// NewCmdUpInfo creates the upinfo command which show the tidb cluster upgrade detail information
func NewCmdUpInfo(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewUpInfoOptions(streams)

	cmd := &cobra.Command{
		Use:     "upinfo",
		Short:   "Show tidb upgrade info.",
		Example: upinfoExample,
		Long:    upinfoLongDesc,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(tkcContext, cmd, args))
			cmdutil.CheckErr(o.Run())
		},
		SuggestFor: []string{"updateinfo", "upgradeinfo"},
	}

	return cmd
}

func (o *UpInfoOptions) Complete(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {

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

func (o *UpInfoOptions) Run() error {

	tc, err := o.TcCli.PingcapV1alpha1().
		TidbClusters(o.Namespace).
		Get(o.TidbClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	setName := controller.TiDBMemberName(tc.Name)
	set, err := o.KubeCli.AppsV1().StatefulSets(o.Namespace).Get(setName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	podList, err := o.KubeCli.CoreV1().Pods(o.Namespace).List(metav1.ListOptions{
		LabelSelector: label.New().Instance(tc.Name).TiDB().String(),
	})
	if err != nil {
		return err
	}
	svcName := tkctlUtil.GetTidbServiceName(tc.Name)
	svc, err := o.KubeCli.CoreV1().Services(o.Namespace).Get(svcName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	msg, err := renderTCUpgradeInfo(tc, set, podList, svc)
	if err != nil {
		return err
	}
	fmt.Fprint(o.Out, msg)
	return nil
}

func getState(updateReplicas int32, ordinal int32, tc *v1alpha1.TidbCluster, pod *v1.Pod) string {

	var state string

	if updateReplicas < ordinal {
		state = UPDATED
	} else if updateReplicas == ordinal {

		state = UPDATING

		if pod.Labels[apps.ControllerRevisionHashLabelKey] == tc.Status.TiDB.StatefulSet.UpdateRevision {
			if member, exist := tc.Status.TiDB.Members[pod.Name]; exist && member.Health {
				state = UPDATED
			}
		}

	} else {
		state = WAITING
	}

	return state
}

func getTiDBServerPort(svc *v1.Service) string {
	for _, port := range svc.Spec.Ports {
		if port.Name == "mysql-client" {
			if port.NodePort != 0 {
				return fmt.Sprintf("%d:%d/%s", port.Port, port.NodePort, port.Protocol)
			}
			return fmt.Sprintf("%d/%s", port.Port, port.Protocol)
		}
	}
	return "<none>"
}

func renderTCUpgradeInfo(tc *v1alpha1.TidbCluster, set *apps.StatefulSet, podList *v1.PodList, svc *v1.Service) (string, error) {
	return readable.TabbedString(func(out io.Writer) error {
		w := readable.NewPrefixWriter(out)
		dbPhase := tc.Status.TiDB.Phase
		w.WriteLine(readable.LEVEL_0, "Name:\t%s", tc.Name)
		w.WriteLine(readable.LEVEL_0, "Namespace:\t%s", tc.Namespace)
		w.WriteLine(readable.LEVEL_0, "CreationTimestamp:\t%s", tc.CreationTimestamp)
		w.WriteLine(readable.LEVEL_0, "Status:\t%s", dbPhase)
		if dbPhase == v1alpha1.UpgradePhase {
			if len(podList.Items) != 0 {
				pod := podList.Items[0]
				w.WriteLine(readable.LEVEL_0, "Image:\t%s ---> %s", pod.Spec.Containers[0].Image, tc.TiDBImage())
			}
		}
		{
			w.WriteLine(readable.LEVEL_1, "Name\tState\tNodeIP\tPodIP\tPort\t")
			w.WriteLine(readable.LEVEL_1, "----\t-----\t------\t-----\t----\t")
			{
				updateReplicas := set.Spec.UpdateStrategy.RollingUpdate.Partition

				if len(podList.Items) != 0 {
					for _, pod := range podList.Items {
						var state string
						ordinal, err := util.GetOrdinalFromPodName(pod.Name)
						if err != nil {
							return err
						}
						if dbPhase == v1alpha1.UpgradePhase {
							state = getState(*updateReplicas, ordinal, tc, &pod)
						} else {
							state = UPDATED
						}

						port := getTiDBServerPort(svc)

						w.WriteLine(readable.LEVEL_1, "%s\t%s\t%s\t%s\t%s\t", pod.Name, state, pod.Status.HostIP, pod.Status.PodIP, port)
					}
				} else {
					w.WriteLine(readable.LEVEL_1, "no resource found")
				}
			}
		}
		return nil
	})
}
