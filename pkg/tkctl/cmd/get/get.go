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

package get

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/tkctl/alias"
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/pingcap/tidb-operator/pkg/tkctl/readable"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	apicore "k8s.io/kubernetes/pkg/apis/core"
)

const (
	getLongDesc = `
		Get tidb component detail.

		Available components include: all, pd, tidb, tikv, volume
		You can omit --tidbcluster=<name> option by running 'tkctl use <name>',
`
	getExample = `
		# get PD details 
		tkctl get pd

		# get multiple kinds of resources
		tkctl get tikv,tidb,volume

		# get volume details and choose different format
		tkctl get volume -o yaml

		# get details of a specific pd/tikv/tidb/volume
		tkctl get volume <volume-name> -oyaml

		# output all columns, including omitted columns
		tkctl get pd,volume -owide

		# get all components
		tkctl get all
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
	ResourceName    string
	GetPD           bool
	GetTiKV         bool
	GetTiDB         bool
	GetVolume       bool

	IsHumanReadablePrinter bool
	PrintFlags             *readable.PrintFlags

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

	// human readable printers have special conversion rules, so we determine if we're using one.
	if len(o.PrintFlags.OutputFormat) == 0 || o.PrintFlags.OutputFormat == "wide" {
		o.IsHumanReadablePrinter = true
	}

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

	if len(args) > 1 {
		o.ResourceName = args[1]
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

	multiTidbCluster := len(tcs) > 1
	var errs []error
	for i, tc := range tcs {
		if multiTidbCluster {
			o.Out.Write([]byte(fmt.Sprintf("Cluster: %s/%s\n", tc.Namespace, tc.Name)))
		}

		// TODO: do a big batch or steadily print parts in minor step?
		if err := o.PrintOutput(&tc, kindPD, o.GetPD); err != nil {
			errs = append(errs, err)
		}

		if err := o.PrintOutput(&tc, kindTiKV, o.GetTiKV); err != nil {
			errs = append(errs, err)
		}

		if err := o.PrintOutput(&tc, kindTiDB, o.GetTiDB); err != nil {
			errs = append(errs, err)
		}

		if err := o.PrintOutput(&tc, kindVolume, o.GetVolume); err != nil {
			errs = append(errs, err)
		}

		if multiTidbCluster && i != len(tcs)-1 {
			o.Out.Write([]byte("\n"))
		}
	}
	return utilerrors.NewAggregate(errs)
}

func (o *GetOptions) PrintOutput(tc *v1alpha1.TidbCluster, resourceType string, needPrint bool) error {
	if !needPrint {
		return nil
	}

	switch resourceType {
	case kindPD, kindTiDB, kindTiKV:
		var objs []runtime.Object
		var listOptions metav1.ListOptions
		if len(o.ResourceName) == 0 {
			listOptions = metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s", label.InstanceLabelKey, tc.Name, label.ComponentLabelKey, resourceType),
			}
		} else {
			listOptions = metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", o.ResourceName),
			}
		}

		podList, err := o.kubeCli.CoreV1().Pods(tc.Namespace).List(listOptions)
		if err != nil {
			return err
		}
		for _, pod := range podList.Items {
			pod.GetObjectKind().SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Pod"))
			objs = append(objs, &pod)
		}

		if !o.IsHumanReadablePrinter {
			return o.printGeneric(objs)
		}

		printer, err := o.PrintFlags.ToPrinter(false, o.AllClusters)
		if err != nil {
			return err
		}
		switch resourceType {
		case kindTiKV:
			tc, err := o.tcCli.PingcapV1alpha1().
				TidbClusters(o.Namespace).
				Get(o.TidbClusterName, metav1.GetOptions{})
			tikvList := alias.TikvList{
				PodList:    podList,
				TikvStatus: nil,
			}
			if err == nil {
				tikvStatus := tc.Status.TiKV
				tikvList.TikvStatus = &tikvStatus
			}
			return printer.PrintObj(&tikvList, o.Out)
		default:
			break
		}
		return printer.PrintObj(podList, o.Out)
	case kindVolume:
		var objs []runtime.Object
		var listOptions metav1.ListOptions
		if len(o.ResourceName) == 0 {
			listOptions = metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s", label.InstanceLabelKey, tc.Name, label.NamespaceLabelKey, tc.Namespace),
			}
		} else {
			listOptions = metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", o.ResourceName),
			}
		}

		volumeList, err := o.kubeCli.CoreV1().PersistentVolumes().List(listOptions)
		if err != nil {
			return err
		}
		for _, volume := range volumeList.Items {
			volume.GetObjectKind().SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("PersistentVolume"))
			objs = append(objs, &volume)
		}

		if !o.IsHumanReadablePrinter {
			return o.printGeneric(objs)
		}

		// PersistentVolume without namespace concept
		printer, err := o.PrintFlags.ToPrinter(false, false)
		if err != nil {
			return err
		}

		return printer.PrintObj(volumeList, o.Out)
	}
	return fmt.Errorf("Unknow resource type %s", resourceType)
}

func (o *GetOptions) printGeneric(objs []runtime.Object) error {
	printer, err := o.PrintFlags.ToPrinter(false, false)
	if err != nil {
		return err
	}

	if len(objs) == 0 {
		return nil
	}

	var allObj runtime.Object
	if len(objs) > 1 {
		list := apicore.List{
			TypeMeta: metav1.TypeMeta{
				Kind:       "List",
				APIVersion: "v1",
			},
			ListMeta: metav1.ListMeta{},
		}
		for _, obj := range objs {
			list.Items = append(list.Items, obj)
		}

		listData, err := json.Marshal(list)
		if err != nil {
			return err
		}

		converted, err := runtime.Decode(unstructured.UnstructuredJSONScheme, listData)
		if err != nil {
			return err
		}

		allObj = converted
	} else {
		allObj = objs[0]
	}

	isList := meta.IsListType(allObj)
	if !isList {
		return printer.PrintObj(allObj, o.Out)
	}

	items, err := meta.ExtractList(allObj)
	if err != nil {
		return err
	}

	// take the items and create a new list for display
	list := &unstructured.UnstructuredList{
		Object: map[string]interface{}{
			"kind":       "List",
			"apiVersion": "v1",
			"metadata":   map[string]interface{}{},
		},
	}
	if listMeta, err := meta.ListAccessor(allObj); err == nil {
		list.Object["metadata"] = map[string]interface{}{
			"selfLink":        listMeta.GetSelfLink(),
			"resourceVersion": listMeta.GetResourceVersion(),
		}
	}

	for _, item := range items {
		list.Items = append(list.Items, *item.(*unstructured.Unstructured))
	}
	return printer.PrintObj(list, o.Out)
}
