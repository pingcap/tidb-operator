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

package get

import (
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/tkctl/alias"
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/pingcap/tidb-operator/pkg/tkctl/readable"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	kubeprinters "k8s.io/kubernetes/pkg/printers"
	"strings"
)

const (
	getLongDesc = `
		Get tidb component detail.

		Available components include: all, pd, tidb, tikv, volume
		You can omit --tidbcluster=<name> option by running 'tkc use <name>',
`
	getExample = `
		# get PD details 
		tkc get pd

		# get multiple kinds of resources
		tkc get tikv,tidb,volume

		# get volume details and choose different format
		tkc get volume -o yaml

		# get all components
		tkc get all
`
	getUsage = "expect 'get -t=CLUSTER_NAME kind | get -A kind' for get command or set tidb cluster by 'use' first"
)

const (
	kindPD     = "pd"
	kindTiKV   = "tikv"
	kindTiDB   = "tidb"
	kindVolume = "volume"
	kindAll    = "all"
)

// GetOptions contains the input to the list command.
type GetOptions struct {
	LabelSelector   string
	AllClusters     bool
	Namespace       string
	TidbClusterName string
	GetPD           bool
	GetTiKV         bool
	GetTiDB         bool
	GetVolume       bool

	PrintFlags *readable.PrintFlags

	tcCli   *versioned.Clientset
	kubeCli *kubernetes.Clientset

	genericclioptions.IOStreams
}

// NewCmdGet creates the get command which get the tidb component detail
func NewCmdGet(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	options := NewGetOptions(streams)

	cmd := &cobra.Command{
		Use:     "get",
		Short:   "get pd|tikv|tidb|volume|all",
		Long:    getLongDesc,
		Example: getExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(options.Complete(tkcContext, cmd, args))
			cmdutil.CheckErr(options.Run(tkcContext, cmd, args))
		},
		SuggestFor: []string{"show", "ps"},
	}

	options.PrintFlags.AddFlags(cmd)
	cmd.Flags().BoolVarP(&options.AllClusters, "all-clusters", "A", false,
		"whether list components of all tidb clusters")

	return cmd
}

// NewGetOptions returns a GetOptions
func NewGetOptions(streams genericclioptions.IOStreams) *GetOptions {
	return &GetOptions{
		PrintFlags: readable.NewPrintFlags(),

		IOStreams: streams,
	}
}

func (o *GetOptions) Complete(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {

	clientConfig, err := tkcContext.ToTkcClientConfig()
	if err != nil {
		return err
	}
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return err
	}
	o.Namespace = namespace
	if tcName, ok := clientConfig.TidbClusterName(); ok {
		o.TidbClusterName = tcName
	} else if !o.AllClusters {
		return cmdutil.UsageErrorf(cmd, getUsage)
	}
	if len(args) < 1 {
		return cmdutil.UsageErrorf(cmd, getUsage)
	}

	restConfig, err := clientConfig.RestConfig()
	if err != nil {
		return err
	}
	tcClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	o.tcCli = tcClient
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	o.kubeCli = kubeClient

	resources := args[0]
	for _, resource := range strings.Split(resources, ",") {
		switch resource {
		case kindAll:
			o.GetPD = true
			o.GetTiKV = true
			o.GetTiDB = true
			o.GetVolume = true
			break
		case kindPD:
			o.GetPD = true
		case kindTiKV:
			o.GetTiKV = true
		case kindTiDB:
			o.GetTiDB = true
		case kindVolume:
			o.GetVolume = true
		}
	}
	return nil
}

func (o *GetOptions) Run(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {

	var tcs []v1alpha1.TidbCluster
	if o.AllClusters {
		tcList, err := o.tcCli.PingcapV1alpha1().
			TidbClusters(v1.NamespaceAll).
			List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		tcs = tcList.Items
	} else {
		tc, err := o.tcCli.PingcapV1alpha1().
			TidbClusters(o.Namespace).
			Get(o.TidbClusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		tcs = []v1alpha1.TidbCluster{*tc}
	}

	printer, err := o.PrintFlags.ToPrinter(false, o.AllClusters)
	if err != nil {
		return err
	}
	w := kubeprinters.GetNewTabWriter(o.Out)
	printTidbInfo := len(tcs) > 1
	var errs []error
	for i := range tcs {
		tc := tcs[i]
		if printTidbInfo {
			w.Write([]byte(fmt.Sprintf("Cluster: %s/%s\n", tc.Namespace, tc.Name)))
			w.Flush()
		}
		flushPods := func(kind string) {
			podList, err := o.kubeCli.CoreV1().Pods(tc.Namespace).List(metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s", label.InstanceLabelKey, tc.Name, label.ComponentLabelKey, kind),
			})
			if err != nil {
				errs = append(errs, err)
			}
			switch kind {
			case kindTiKV:
				tikvList := &alias.TikvList{}
				tikvList.FromPodList(podList)
				printer.PrintObj(tikvList, w)
				break
			default:
				printer.PrintObj(podList, w)
				break
			}
			w.Flush()
		}
		// TODO: do a big batch or steadily print parts in minor step?
		if o.GetPD {
			flushPods(kindPD)
		}
		if o.GetTiKV {
			flushPods(kindTiKV)
		}
		if o.GetTiDB {
			flushPods(kindTiDB)
		}
		if o.GetVolume {
			volumeList, err := o.kubeCli.CoreV1().PersistentVolumes().List(metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s", label.InstanceLabelKey, tc.Name,
					label.NamespaceLabelKey, tc.Namespace),
			})
			if err != nil {
				return err
			}
			printer.PrintObj(volumeList, w)
			w.Flush()
		}
	}
	return nil
}
