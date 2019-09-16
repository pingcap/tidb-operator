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

	// registry mysql drive
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/backup"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/cache"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

// NewBackupCommand implements the backup command
func NewBackupCommand() *cobra.Command {
	bo := backup.BackupOpts{}

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup specific tidb cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			util.ValidCmdFlags(cmd.CommandPath(), cmd.LocalFlags())
			cmdutil.CheckErr(runBackup(bo, kubecfg))
		},
	}

	cmd.Flags().StringVarP(&bo.Namespace, "namespace", "n", "", "Tidb cluster's namespace")
	cmd.Flags().StringVarP(&bo.TcName, "tidbcluster", "t", "", "Tidb cluster name")
	cmd.Flags().StringVarP(&bo.TidbSvc, "tidbservice", "s", "", "Tidb cluster access service address")
	cmd.Flags().StringVarP(&bo.Password, "password", "p", "", "Password to use when connecting to tidb cluster")
	cmd.Flags().StringVarP(&bo.User, "user", "u", "", "User for login tidb cluster")
	cmd.Flags().StringVarP(&bo.StorageType, "storageType", "S", "", "Backend storage type")
	cmd.Flags().StringVarP(&bo.BackupName, "backupName", "b", "", "Backup CRD object name")
	return cmd
}

func runBackup(backupOpts backup.BackupOpts, kubecfg string) error {
	kubeCli, cli, err := util.NewKubeAndCRCli(kubecfg)
	cmdutil.CheckErr(err)
	options := []informers.SharedInformerOption{
		informers.WithNamespace(backupOpts.Namespace),
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(cli, constants.ResyncDuration, options...)
	recorder := util.NewEventRecorder(kubeCli, "backup")
	backupInformer := informerFactory.Pingcap().V1alpha1().Backups()
	statusUpdater := controller.NewRealBackupConditionUpdater(cli, backupInformer.Lister(), recorder)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go informerFactory.Start(ctx.Done())

	// waiting for the shared informer's store has synced.
	cache.WaitForCacheSync(ctx.Done(), backupInformer.Informer().HasSynced)

	glog.Infof("start to process backup %s", backupOpts)
	bm := backup.NewBackupManager(backupInformer.Lister(), statusUpdater, backupOpts)
	return bm.ProcessBackup()
}
