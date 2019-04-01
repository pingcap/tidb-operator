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
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	getLongDesc = `
		Get tidb component detail.

		Available components include: cluster, pd, tidb, tikv, volume
		You can omit --tidbcluster=<name> option by running 'tkc use <name>',
`
	getExample = `
		# get cluster details
		tkc get cluster

		# get PD details 
		tkc get pd

		# get a certain tikv server details
		tkc get tikv 1

		# get volume details and choose different format
		tkc get volume -o yaml
`
)

// GetOptions contains the input to the list command.
type GetOptions struct {
	IsHumanReadablePrinter bool
	PrintWithOpenAPICols   bool

	LabelSelector string
	AllNamespaces bool
	Namespace     string

	ServerPrint bool

	genericclioptions.IOStreams
}

// NewCmdGet creates the get command which get the tidb component detail
func NewCmdGet(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewGetOptions(streams)

	cmd := &cobra.Command{
		Use:     "get",
		Short:   "get all tidb clusters",
		Long:    getLongDesc,
		Example: getExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(tkcContext, cmd, args))
			cmdutil.CheckErr(o.Validate(cmd))
			cmdutil.CheckErr(o.Run(tkcContext, cmd, args))
		},
		SuggestFor: []string{"show", "ps"},
	}

	return cmd
}

// NewGetOptions returns a GetOptions
func NewGetOptions(streams genericclioptions.IOStreams) *GetOptions {
	return &GetOptions{

		IOStreams:   streams,
		ServerPrint: true,
	}
}

func (o *GetOptions) Complete(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {

	return nil
}

func (o *GetOptions) Validate(cmd *cobra.Command) error {
	return nil
}

func (o *GetOptions) Run(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {
	return nil
}
