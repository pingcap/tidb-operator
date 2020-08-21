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
	"github.com/spf13/cobra"
)

var kubecfg string

func NewBackupMgrCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "tidb-backup-manager",
		Short: "Helper for backup manage",
		Long:  "Dump tidb cluster data, as well as backup and restore tidb cluster data",
		Run:   runHelp,
	}

	cmds.PersistentFlags().StringVarP(&kubecfg, "kubeconfig", "k", "", "Path to kubeconfig file, omit this if run in cluster.")

	cmds.AddCommand(NewBackupCommand())
	cmds.AddCommand(NewExportCommand())
	cmds.AddCommand(NewRestoreCommand())
	cmds.AddCommand(NewImportCommand())
	cmds.AddCommand(NewCleanCommand())
	return cmds
}

func runHelp(cmd *cobra.Command, _ []string) {
	cmd.Help()
}
