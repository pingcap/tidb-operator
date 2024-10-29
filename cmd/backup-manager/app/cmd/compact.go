// Copyright 2024 PingCAP, Inc.
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
	"context"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/compact"
	coptions "github.com/pingcap/tidb-operator/cmd/backup-manager/app/compact/options"
	"github.com/spf13/cobra"
)

func NewCompactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compact",
		Short: "Compact log backup.",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := coptions.KubeOpts{}
			if err := opts.ParseFromFlags(cmd.Flags()); err != nil {
				return err
			}
			opts.Kubeconfig = kubecfg

			ctx := context.Background()
			link, err := compact.NewKubelink(opts.Kubeconfig)
			if err != nil {
				return err
			}
			cx := compact.New(opts, link)
			return cx.Run(ctx)
		},
	}

	coptions.DefineFlags(cmd.Flags())
	return cmd
}
