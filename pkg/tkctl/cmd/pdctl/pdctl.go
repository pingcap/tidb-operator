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

package pdctl

import (
	"fmt"
	"os"

	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// TODO: implementation
// NewCmdPdctl creates the pdctl subcommand
func NewCmdPdctl(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "pdctl",
		Short: "Not implemented",
		Run: func(_ *cobra.Command, args []string) {
			fmt.Fprint(os.Stdout, "not implemented")
		},
	}
}
