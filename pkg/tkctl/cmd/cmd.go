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

package cmd

import (
	"flag"
	"io"

	"github.com/pingcap/tidb-operator/pkg/tkctl/cmd/diagnose"

	"github.com/pingcap/tidb-operator/pkg/tkctl/cmd/completion"
	"github.com/pingcap/tidb-operator/pkg/tkctl/cmd/ctop"
	"github.com/pingcap/tidb-operator/pkg/tkctl/cmd/debug"
	"github.com/pingcap/tidb-operator/pkg/tkctl/cmd/get"
	"github.com/pingcap/tidb-operator/pkg/tkctl/cmd/info"
	"github.com/pingcap/tidb-operator/pkg/tkctl/cmd/list"
	"github.com/pingcap/tidb-operator/pkg/tkctl/cmd/upinfo"
	"github.com/pingcap/tidb-operator/pkg/tkctl/cmd/use"
	"github.com/pingcap/tidb-operator/pkg/tkctl/cmd/version"
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	// TODO: import azure auth plugin after updating to k8s 1.13+
	_ "k8s.io/client-go/plugin/pkg/client/auth/exec"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
	"k8s.io/kubectl/pkg/util/templates"
)

const (
	tkcLongDescription = `
		"tkctl"(TiDB kubernetes control) is a command line interface for cloud tidb management and troubleshooting.
`
)

// NewTkcCommand creates the root `tkc` command and its nested children.
func NewTkcCommand(streams genericclioptions.IOStreams) *cobra.Command {

	options := &config.TkcOptions{}

	// Root command that all the subcommands are added to
	rootCmd := &cobra.Command{
		Use:   "tkctl",
		Short: "TiDB kubernetes control.",
		Long:  tkcLongDescription,
		Run:   runHelp,
	}

	rootCmd.PersistentFlags().StringVarP(&options.TidbClusterName,
		"tidbcluster", "t", options.TidbClusterName, "Tidb cluster name")

	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	// Reuse kubectl global flags to provide namespace, context and credential options
	kubeFlags := genericclioptions.NewConfigFlags(true)
	kubeFlags.AddFlags(rootCmd.PersistentFlags())
	tkcContext := config.NewTkcContext(kubeFlags, options)

	groups := templates.CommandGroups{
		{
			Message: "Cluster Management Commands:",
			Commands: []*cobra.Command{
				list.NewCmdList(tkcContext, streams),
				get.NewCmdGet(tkcContext, streams),
				info.NewCmdInfo(tkcContext, streams),
				use.NewCmdUse(tkcContext, streams),
				version.NewCmdVersion(tkcContext, streams.Out),
				upinfo.NewCmdUpInfo(tkcContext, streams),
				diagnose.NewCmdDiagnoseInfo(tkcContext, streams),
			},
		},
		{
			Message: "Troubleshooting Commands:",
			Commands: []*cobra.Command{
				debug.NewCmdDebug(tkcContext, streams),
				ctop.NewCmdCtop(tkcContext, streams),
			},
		},
		{
			Message: "Settings Commands:",
			Commands: []*cobra.Command{
				completion.NewCmdCompletion(streams.Out),
			},
		},
	}
	groups.Add(rootCmd)
	templates.ActsAsRootCommand(rootCmd, []string{"options"}, groups...)
	rootCmd.AddCommand(NewCmdOptions(streams.Out))

	return rootCmd
}

func runHelp(cmd *cobra.Command, _ []string) {
	cmd.Help()
}

// NewCmdOptions implements the options command which shows all global options
func NewCmdOptions(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "options",
		Short: "Print the list of flags inherited by all commands",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
		},
	}

	cmd.SetOutput(out)

	templates.UseOptionsTemplates(cmd)
	return cmd
}
