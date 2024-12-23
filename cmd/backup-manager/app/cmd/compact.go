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
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/compact/options"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/cache"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func NewCompactCommand() *cobra.Command {
	opts := options.CompactOpts{}

	cmd := &cobra.Command{
		Use:   "compact",
		Short: "Compact log backup.",
		Run: func(cmd *cobra.Command, args []string) {
			util.ValidCmdFlags(cmd.CommandPath(), cmd.LocalFlags())
			cmdutil.CheckErr(runCompact(opts, kubecfg))
		},
	}

	cmd.Flags().StringVar(&opts.Namespace, "namespace", "", "Backup CR's namespace")
	cmd.Flags().StringVar(&opts.ResourceName, "resourceName", "", "Backup CRD object name")
	return cmd
}

func runCompact(compactOpts options.CompactOpts, kubecfg string) error {
	kubeCli, cli, err := util.NewKubeAndCRCli(kubecfg)
	if err != nil {
		return err
	}
	options := []informers.SharedInformerOption{
		informers.WithNamespace(compactOpts.Namespace),
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(cli, constants.ResyncDuration, options...)
	recorder := util.NewEventRecorder(kubeCli, "compact-manager")
	compactInformer := informerFactory.Pingcap().V1alpha1().CompactBackups()
	statusUpdater := controller.NewCompactStatusUpdater(recorder, compactInformer.Lister(), cli)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go informerFactory.Start(ctx.Done())

	// waiting for the shared informer's store has synced.
	cache.WaitForCacheSync(ctx.Done(), compactInformer.Informer().HasSynced)

	// klog.Infof("start to process backup %s", compactOpts.String())
	cm := compact.NewManager(compactInformer.Lister(), statusUpdater, compactOpts)
	return cm.ProcessCompact()
}
