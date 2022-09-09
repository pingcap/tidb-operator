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

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/backup"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// NewBackupCommand implements the backup command
func NewBackupCommand() *cobra.Command {
	bo := backup.Options{}

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup specific tidb cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			util.ValidCmdFlags(cmd.CommandPath(), cmd.LocalFlags())
			cmdutil.CheckErr(runBackup(bo, kubecfg))
		},
	}

	cmd.Flags().StringVar(&bo.Namespace, "namespace", "", "Backup CR's namespace")
	cmd.Flags().StringVar(&bo.ResourceName, "backupName", "", "Backup CRD object name")
	cmd.Flags().StringVar(&bo.TiKVVersion, "tikvVersion", util.DefaultVersion, "TiKV version")
	cmd.Flags().BoolVar(&bo.TLSClient, "client-tls", false, "Whether client tls is enabled")
	cmd.Flags().BoolVar(&bo.TLSCluster, "cluster-tls", false, "Whether cluster tls is enabled")
	cmd.Flags().BoolVar(&bo.SkipClientCA, "skipClientCA", false, "Whether to skip tidb server's certificates validation")
	cmd.Flags().StringVar(&bo.Mode, "mode", string(v1alpha1.BackupModeSnapshot), "backup mode, which is log or snapshot(default)")
	cmd.Flags().StringVar(&bo.SubCommand, "subcommand", string(v1alpha1.LogStartCommand), "the log backup subcommand")
	cmd.Flags().StringVar(&bo.CommitTS, "commit-ts", "0", "the log backup start ts")
	cmd.Flags().StringVar(&bo.TruncateUntil, "truncate-until", "0", "the log backup truncate until")
	return cmd
}

func runBackup(backupOpts backup.Options, kubecfg string) error {
	kubeCli, cli, err := util.NewKubeAndCRCli(kubecfg)
	if err != nil {
		return err
	}
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

	klog.Infof("start to process backup %s", backupOpts.String())
	bm := backup.NewManager(backupInformer.Lister(), statusUpdater, backupOpts)
	return bm.ProcessBackup()
}
