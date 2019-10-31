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
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/restore"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// NewRestoreCommand implements the restore command
func NewRestoreCommand() *cobra.Command {
	ro := restore.RestoreOpts{}

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore specific tidb cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			util.ValidCmdFlags(cmd.CommandPath(), cmd.LocalFlags())
			cmdutil.CheckErr(runRestore(ro, kubecfg))
		},
	}

	cmd.Flags().StringVarP(&ro.Namespace, "namespace", "n", "", "Tidb cluster's namespace")
	cmd.Flags().StringVarP(&ro.TcName, "tidbcluster", "t", "", "Tidb cluster name")
	cmd.Flags().StringVarP(&ro.Password, "password", "p", "", "Password to use when connecting to tidb cluster")
	cmd.Flags().StringVarP(&ro.TidbSvc, "tidbservice", "s", "", "Tidb cluster access service address")
	cmd.Flags().StringVarP(&ro.User, "user", "u", "", "User for login tidb cluster")
	cmd.Flags().StringVarP(&ro.RestoreName, "restoreName", "r", "", "Restore CRD object name")
	cmd.Flags().StringVarP(&ro.BackupName, "backupName", "b", "", "Backup CRD object name")
	cmd.Flags().StringVarP(&ro.BackupPath, "backupPath", "P", "", "The location of the backup")
	return cmd
}

func runRestore(restoreOpts restore.RestoreOpts, kubecfg string) error {
	kubeCli, cli, err := util.NewKubeAndCRCli(kubecfg)
	cmdutil.CheckErr(err)
	options := []informers.SharedInformerOption{
		informers.WithNamespace(restoreOpts.Namespace),
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(cli, constants.ResyncDuration, options...)
	recorder := util.NewEventRecorder(kubeCli, "restore")
	restoreInformer := informerFactory.Pingcap().V1alpha1().Restores()
	statusUpdater := controller.NewRealRestoreConditionUpdater(cli, restoreInformer.Lister(), recorder)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go informerFactory.Start(ctx.Done())

	// waiting for the shared informer's store has synced.
	cache.WaitForCacheSync(ctx.Done(), restoreInformer.Informer().HasSynced)

	glog.Infof("start to process restore %s", restoreOpts)
	rm := restore.NewRestoreManager(restoreInformer.Lister(), statusUpdater, restoreOpts)
	return rm.ProcessRestore()
}
