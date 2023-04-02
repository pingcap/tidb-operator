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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// NewRestoreCommand implements the restore command
func NewRestoreCommand() *cobra.Command {
	ro := restore.Options{}

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore specific tidb cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			util.ValidCmdFlags(cmd.CommandPath(), cmd.LocalFlags())
			cmdutil.CheckErr(runRestore(ro, kubecfg))
		},
	}

	cmd.Flags().StringVar(&ro.Namespace, "namespace", "", "Restore CR's namespace")
	cmd.Flags().StringVar(&ro.ResourceName, "restoreName", "", "Restore CRD object name")
	cmd.Flags().StringVar(&ro.TiKVVersion, "tikvVersion", util.DefaultVersion, "TiKV version")
	cmd.Flags().BoolVar(&ro.TLSClient, "client-tls", false, "Whether client tls is enabled")
	cmd.Flags().BoolVar(&ro.TLSCluster, "cluster-tls", false, "Whether cluster tls is enabled")
	cmd.Flags().BoolVar(&ro.SkipClientCA, "skipClientCA", false, "Whether to skip tidb server's certificates validation")
	cmd.Flags().StringVar(&ro.Mode, "mode", string(v1alpha1.RestoreModeSnapshot), "restore mode, which is pitr or snapshot(default)")
	cmd.Flags().StringVar(&ro.PitrRestoredTs, "pitrRestoredTs", "0", "The pitr restored ts")
	cmd.Flags().BoolVar(&ro.Prepare, "prepare", false, "Whether to prepare for restore")
	return cmd
}

func runRestore(restoreOpts restore.Options, kubecfg string) error {
	kubeCli, cli, err := util.NewKubeAndCRCli(kubecfg)
	if err != nil {
		return err
	}
	options := []informers.SharedInformerOption{
		informers.WithNamespace(restoreOpts.Namespace),
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(cli, constants.ResyncDuration, options...)
	recorder := util.NewEventRecorder(kubeCli, "restore")
	restoreInformer := informerFactory.Pingcap().V1alpha1().Restores()
	statusUpdater := controller.NewRealRestoreConditionUpdater(cli, restoreInformer.Lister(), recorder)
	restoreControl := controller.NewRealRestoreControl(cli, restoreInformer.Lister(), recorder)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go informerFactory.Start(ctx.Done())

	// waiting for the shared informer's store has synced.
	cache.WaitForCacheSync(ctx.Done(), restoreInformer.Informer().HasSynced)

	klog.Infof("start to process restore %s", restoreOpts.String())
	rm := restore.NewManager(restoreInformer.Lister(), statusUpdater, restoreControl, restoreOpts)
	return rm.ProcessRestore()
}
