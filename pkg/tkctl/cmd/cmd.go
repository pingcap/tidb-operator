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

package cmd

import (
	"flag"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubernetes/pkg/kubectl/cmd/templates"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

)

// NewTkcCommand creates the `tkc` command and its nested children.
func NewTkcCommand(streams genericclioptions.IOStreams) *cobra.Command {
	// Root command to which all subcommands are added.
	rootCmd := &cobra.Command{
		Use:   "tkc",
		Short: "TiDB kubernetes control.",
		Long: `TiDB kubernetes command line interface for cluster management and troubleshooting.
`,
		Run:   runHelp,
	}

	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	configFlags := genericclioptions.NewConfigFlags()
	configFlags.AddFlags(rootCmd.PersistentFlags())
	f := cmdutil.NewFactory(configFlags)
	f.KubernetesClientSet()

	groups := templates.CommandGroups{
		//{
		//	Message: "Cluster Management Commands:",
		//	Commands: []*cobra.Command{
		//
		//	},
		//},
		//{
		//	Message: "Volume and Backup Commands:",
		//	Commands: []*cobra.Command{
		//
		//	},
		//},
		//{
		//	Message: "Troubleshooting Commands:",
		//	Commands: []*cobra.Command{
		//		debug.NewCmdDebug(configFlags, streams),
		//		pdctl.NewCmdPdctl(configFlags, streams),
		//		ctop.NewCmdCtop(configFlags, streams),
		//	},
		//},
		//{
		//	Message: "Cluster Meta Commands:",
		//	Commands: []*cobra.Command{
		//
		//	},
		//},
	}
	groups.Add(rootCmd)
	templates.ActsAsRootCommand(rootCmd, []string{"options"}, groups...)

	return rootCmd
}

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}
