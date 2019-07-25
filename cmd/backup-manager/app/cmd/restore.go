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

// restoreOpts contains the input arguments to the restore command
type restoreOpts struct {
	namespace  string
	tcName     string
	password   string
	user       string
	restore    string
	backupPath string
}

// NewRestoreCommand implements the restore command
func NewRestoreCommand(kubecfg string) *cobra.Command {
	ro := &restoreOpts{}

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore specific tidb cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			kubeCli, cli, err := util.NewKubeAndCRCli(kubecfg)
			cmdutil.CheckErr(err)
			options := []informers.SharedInformerOption{
				informers.WithNamespace(ro.namespace),
			}
			informerFactory := informers.NewSharedInformerFactoryWithOptions(cli, constants.ResyncDuration, options...)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go informerFactory.Start(ctx.Done())
			recorder := util.NewEventRecorder(kubeCli, "restore")
			backupInformer := informerFactory.Pingcap().V1alpha1().Backups()
			statusUpdater := controller.NewRealBackupConditionUpdater(cli, backupInformer.Lister(), recorder)
			cmdutil.CheckErr(ro.Run(kubeCli, statusUpdater))
		},
	}

	cmd.Flags().StringVarP(&ro.namespace, "namespace", "n", "", "Tidb cluster's namespace")
	cmd.Flags().StringVarP(&ro.tcName, "tidbcluster", "t", "", "Tidb cluster name")
	cmd.Flags().StringVarP(&ro.password, "password", "p", "", "Password to use when connecting to tidb cluster")
	cmd.Flags().StringVarP(&ro.user, "user", "u", "", "User for login tidb cluster")
	cmd.Flags().StringVarP(&ro.restore, "restore", "r", "", "Restore CRD object name")
	cmd.Flags().StringVarP(&ro.backupPath, "backupPath", "b", "", "BackupPath is the location of the backup")
	return cmd
}

func (ro *restoreOpts) Run(kubeCli kubernetes.Interface, statusUpdater controller.BackupConditionUpdaterInterface) error {
	return nil
}
