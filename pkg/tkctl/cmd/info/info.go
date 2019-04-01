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
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	infoLongDesc = `
		Get tidb cluster information of a specified cluster.

		You can omit --tidbcluster=<name> option and <clusterName> flag by 
		running 'tkc use <clusterName>',
`
	infoExample = `
		# get cluster information
		tkc info demo
`
)

// InfoOptions contains the input to the list command.
type InfoOptions struct {
	IsHumanReadablePrinter bool
	PrintWithOpenAPICols   bool

	LabelSelector string
	AllNamespaces bool
	Namespace     string

	ServerPrint bool

	genericclioptions.IOStreams
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
			cmdutil.CheckErr(o.Validate(cmd))
			cmdutil.CheckErr(o.Run(tkcContext, cmd, args))
		},
		SuggestFor: []string{"inspect", "explain"},
	}

	return cmd

}

// NewInfoOptions returns a InfoOptions
func NewInfoOptions(streams genericclioptions.IOStreams) *InfoOptions {
	return &InfoOptions{

		IOStreams:   streams,
		ServerPrint: true,
	}
}

func (o *InfoOptions) Complete(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *InfoOptions) Validate(cmd *cobra.Command) error {
	return nil
}

func (o *InfoOptions) Run(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {
	return nil
}
