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

package list

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/pingcap/tidb-operator/pkg/tkctl/readable"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	kubeprinters "k8s.io/kubernetes/pkg/printers"
)

const (
	listLongDesc = `
		List all tidb clusters.

		Prints a table of the general information about each tidb cluster. By specifying
		namespace or label-selectors, you can filter clusters.
`
	listExample = `
		# list all clusters and sync to local config
		tkctl list

		# filter by namespace
		tkctl list --namespace=foo

		# get tidb cluster in all namespaces
		tkctl list -A
`
)

// ListOptions contains the input to the list command.
type ListOptions struct {
	AllNamespaces bool
	Namespace     string

	PrintFlags *readable.PrintFlags

	genericclioptions.IOStreams
}

// NewListOptions returns a ListOptions.
func NewListOptions(streams genericclioptions.IOStreams) *ListOptions {
	return &ListOptions{
		PrintFlags: readable.NewPrintFlags(),

		IOStreams: streams,
	}
}

// NewCmdList creates the list command which lists all the tidb cluster
// in the specified kubernetes cluster and sync to local config file.
// List only searches for pingcap.com/tidbclusters custom resources.
func NewCmdList(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	options := NewListOptions(streams)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "list all tidb clusters",
		Long:    listLongDesc,
		Example: listExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(options.Complete(tkcContext, cmd, args))
			cmdutil.CheckErr(options.Run(tkcContext, cmd, args))
		},
		SuggestFor: []string{"ls", "ps"},
	}

	options.PrintFlags.AddFlags(cmd)

	cmd.Flags().BoolVarP(&options.AllNamespaces, "all-namespaces", "A", false,
		"whether list tidb clusters in all namespaces")
	return cmd
}

func (o *ListOptions) Complete(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {
	clientConfig, err := tkcContext.ToTkcClientConfig()
	if err != nil {
		return err
	}
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return err
	}
	o.Namespace = namespace
	return nil
}

func (o *ListOptions) Run(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {
	r := tkcContext.TkcBuilder().
		NamespaceParam(o.Namespace).DefaultNamespace().AllNamespaces(o.AllNamespaces).
		SingleResourceType().ResourceTypes("tidbcluster").
		SelectAllParam(true).
		Unstructured().
		ContinueOnError().
		Latest().
		Flatten().
		Do()

	if err := r.Err(); err != nil {
		return err
	}

	printer, err := o.PrintFlags.ToPrinter(false, o.AllNamespaces)
	if err != nil {
		return err
	}

	infos, err := r.Infos()
	if err != nil {
		return err
	}

	w := kubeprinters.GetNewTabWriter(o.Out)
	for _, info := range infos {
		internalObj, err := v1alpha1.Scheme.ConvertToVersion(info.Object, v1alpha1.SchemeGroupVersion)
		if err != nil {
			klog.V(1).Info(err)
			printer.PrintObj(info.Object, w)
		} else {
			printer.PrintObj(internalObj, w)
		}
	}
	w.Flush()

	return nil
}
