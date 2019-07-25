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
	"context"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

// cleanOpts contains the input arguments to the clean command
type cleanOpts struct {
	namespace string
	tcName    string
	backup    string
}

// NewCleanCommand implements the clean command
func NewCleanCommand(kubecfg string) *cobra.Command {
	co := &cleanOpts{}

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Clean specific tidb cluster backup.",
		Run: func(cmd *cobra.Command, args []string) {
			kubeCli, cli, err := util.NewKubeAndCRCli(kubecfg)
			cmdutil.CheckErr(err)
			options := []informers.SharedInformerOption{
				informers.WithNamespace(co.namespace),
			}
			informerFactory := informers.NewSharedInformerFactoryWithOptions(cli, constants.ResyncDuration, options...)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go informerFactory.Start(ctx.Done())
			recorder := util.NewEventRecorder(kubeCli, "backup")
			backupInformer := informerFactory.Pingcap().V1alpha1().Backups()
			statusUpdater := controller.NewRealBackupConditionUpdater(cli, backupInformer.Lister(), recorder)
			cmdutil.CheckErr(co.Run(kubeCli, statusUpdater))
		},
	}

	cmd.Flags().StringVarP(&co.namespace, "namespace", "n", "", "Tidb cluster's namespace")
	cmd.Flags().StringVarP(&co.tcName, "tidbcluster", "t", "", "Tidb cluster name")
	cmd.Flags().StringVarP(&co.backup, "backup", "b", "", "Backup CRD object name")
	return cmd
}

func (co *cleanOpts) Run(kubeCli kubernetes.Interface, statusUpdater controller.BackupConditionUpdaterInterface) error {
	return nil
}
