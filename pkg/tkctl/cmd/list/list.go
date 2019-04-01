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

package list

import (
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	listLongDesc = `
		List all tidb clusters.

		Prints a table of the general information about each tidb cluster. By specifying
		namespace or label-selectors, you can filter clusters.
`
	listExample = `
		# list all clusters and sync to local config
		tkc list

		# filter by namespace
		tkc list --namespace=foo

		# choose json output format
		tkc list -o json
`
)

// ListOptions contains the input to the list command.
type ListOptions struct {
	IsHumanReadablePrinter bool
	PrintWithOpenAPICols   bool

	LabelSelector string
	AllNamespaces bool
	Namespace     string

	ServerPrint bool

	genericclioptions.IOStreams
}

// NewCmdList creates the list command which lists all the tidb cluster
// in the specified kubernetes cluster and sync to local config file.
// List only searches for pingcap.com/tidbclusters custom resources.
func NewCmdList(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "list all tidb clusters",
		Long:    listLongDesc,
		Example: listExample,
		Run: func(cmd *cobra.Command, args []string) {
		},
		SuggestFor: []string{"ls", "ps"},
	}
	return cmd
}

// NewListOptions returns a ListOptions.
func NewListOptions(streams genericclioptions.IOStreams) *ListOptions {
	return &ListOptions{

		IOStreams:   streams,
		ServerPrint: true,
	}
}

func (o *ListOptions) Complete(f cmdutil.Factory, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *ListOptions) Validate(cmd *cobra.Command) error {
	return nil
}

func (o *ListOptions) Run(f cmdutil.Factory, cmd *cobra.Command, args []string) error {
	return nil
}
